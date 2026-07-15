package billing

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Clock generates ids and timestamps (implemented by main.CommonUtils).
type Clock interface {
	GenerateUniqueID(prefix *string) string
	GetCurrentTimestamp() int64
}

type Service struct {
	repo *Repository
}

// TableOpenError is returned when creating a session for a table that already has a live or billing session.
type TableOpenError struct {
	TableID   string
	SessionID string
}

func (e *TableOpenError) Error() string {
	return fmt.Sprintf("table %s already has an open session; add orders to session %s", e.TableID, e.SessionID)
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func defaultVenueID() string {
	v := strings.TrimSpace(os.Getenv("TANGIFY_VENUE_ID"))
	if v == "" {
		return "default"
	}
	return v
}

func sumLineItems(items []LineItem) int64 {
	return SumLineItems(items)
}

func tableInSession(tableIDs []string, tableID string) bool {
	for _, t := range tableIDs {
		if t == tableID {
			return true
		}
	}
	return false
}

func (s *Service) findLiveSessionForTable(ctx context.Context, venueID, tableID string) (*TableSession, error) {
	sessions, err := s.repo.QuerySessionsByVenue(ctx, venueID, 500)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sess := &sessions[i]
		if sess.Status != SessionStatusLive && sess.Status != SessionStatusBilling {
			continue
		}
		if tableInSession(sess.TableIDs, tableID) {
			return sess, nil
		}
	}
	return nil, nil
}

// findOpenSessionForAnyTable returns an existing live/billing session if any requested table is already part of one.
func (s *Service) findOpenSessionForAnyTable(ctx context.Context, venueID string, tableIDs []string) (*TableSession, string, error) {
	sessions, err := s.repo.QuerySessionsByVenue(ctx, venueID, 500)
	if err != nil {
		return nil, "", err
	}
	want := make(map[string]struct{})
	for _, t := range tableIDs {
		t = strings.TrimSpace(t)
		if t != "" {
			want[t] = struct{}{}
		}
	}
	if len(want) == 0 {
		return nil, "", nil
	}
	for i := range sessions {
		sess := &sessions[i]
		if sess.Status != SessionStatusLive && sess.Status != SessionStatusBilling {
			continue
		}
		for _, tid := range sess.TableIDs {
			tid = strings.TrimSpace(tid)
			if _, ok := want[tid]; ok {
				return sess, tid, nil
			}
		}
	}
	return nil, "", nil
}

// --- Waiter ---

func (s *Service) LiveOrdersGrouped(ctx context.Context, venueID string) (*LiveOrdersGroupedResponse, error) {
	if venueID == "" {
		venueID = defaultVenueID()
	}
	sessions, err := s.repo.QuerySessionsByVenue(ctx, venueID, 500)
	if err != nil {
		return nil, err
	}
	var bundles []SessionWithOrders
	for i := range sessions {
		sess := sessions[i]
		if sess.Status != SessionStatusLive && sess.Status != SessionStatusBilling {
			continue
		}
		orders, err := s.repo.QueryOrdersBySession(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, SessionWithOrders{Session: sess, Orders: orders})
	}
	return &LiveOrdersGroupedResponse{Sessions: bundles}, nil
}

func (s *Service) CreateSessionAndFirstOrder(ctx context.Context, req CreateSessionAndFirstOrderRequest, staffID string, clock Clock) (*SessionWithOrders, error) {
	if len(req.TableIDs) == 0 {
		return nil, fmt.Errorf("table_ids required")
	}
	if req.Channel == "" {
		return nil, fmt.Errorf("channel required")
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("items required")
	}
	venueID := defaultVenueID()
	if existing, tableID, err := s.findOpenSessionForAnyTable(ctx, venueID, req.TableIDs); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, &TableOpenError{TableID: tableID, SessionID: existing.ID}
	}
	now := clock.GetCurrentTimestamp()
	pSess := PrefixSession
	sid := clock.GenerateUniqueID(&pSess)
	pOrd := PrefixOrder
	oid := clock.GenerateUniqueID(&pOrd)

	items := ensureLineItemIDs(req.Items)
	cust := ""
	if req.CustomerID != nil {
		cust = *req.CustomerID
	}
	st := staffID
	if req.StaffID != nil && *req.StaffID != "" {
		st = *req.StaffID
	}
	orderedAt := now
	if req.OrderedAt != nil && *req.OrderedAt != 0 {
		orderedAt = *req.OrderedAt
	}

	session := TableSession{
		ID:        sid,
		TableIDs:  req.TableIDs,
		Status:    SessionStatusLive,
		OpenedAt:  now,
		UpdatedAt: now,
		VenueID:   venueID,
	}
	if req.Pax != nil {
		session.Pax = *req.Pax
	}
	if req.GroupNotes != nil {
		session.GroupNotes = strings.TrimSpace(*req.GroupNotes)
	}
	if req.ServiceFlags != nil {
		session.ServiceFlags = req.ServiceFlags
	}
	order := Order{
		ID:            oid,
		SessionID:     sid,
		VenueID:       venueID,
		Channel:       NormalizeChannel(req.Channel),
		CustomerID:    cust,
		StaffID:       st,
		Items:         items,
		TotalPrice:    sumLineItems(items),
		KitchenStatus: KitchenStatusPending,
		OrderedAt:     orderedAt,
		UpdatedAt:     now,
	}
	if req.CustomerName != nil {
		order.CustomerName = strings.TrimSpace(*req.CustomerName)
	}
	if req.CustomerPhone != nil {
		order.CustomerPhone = strings.TrimSpace(*req.CustomerPhone)
	}
	if req.Notes != nil {
		order.Notes = strings.TrimSpace(*req.Notes)
	}
	if err := s.repo.PutSession(ctx, &session); err != nil {
		return nil, err
	}
	if err := s.repo.PutOrder(ctx, &order); err != nil {
		return nil, err
	}
	return &SessionWithOrders{Session: session, Orders: []Order{order}}, nil
}

func (s *Service) AddOrder(ctx context.Context, req AddOrderToSessionRequest, staffID string, clock Clock) (*Order, error) {
	if req.SessionID == "" || req.Channel == "" || len(req.Items) == 0 {
		return nil, fmt.Errorf("session_id, channel, and items required")
	}
	sess, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	if sess.Status != SessionStatusLive && sess.Status != SessionStatusBilling {
		return nil, fmt.Errorf("session is not open for orders")
	}
	venueID := sess.VenueID
	if venueID == "" {
		venueID = defaultVenueID()
	}
	now := clock.GetCurrentTimestamp()
	pOrd := PrefixOrder
	oid := clock.GenerateUniqueID(&pOrd)
	items := ensureLineItemIDs(req.Items)
	st := staffID
	if req.StaffID != nil && *req.StaffID != "" {
		st = *req.StaffID
	}
	orderedAt := now
	if req.OrderedAt != nil && *req.OrderedAt != 0 {
		orderedAt = *req.OrderedAt
	}
	src := ""
	if req.SourceTableID != nil {
		src = *req.SourceTableID
	}
	order := Order{
		ID:            oid,
		SessionID:     req.SessionID,
		VenueID:       venueID,
		Channel:       NormalizeChannel(req.Channel),
		SourceTableID: src,
		StaffID:       st,
		Items:         items,
		TotalPrice:    sumLineItems(items),
		KitchenStatus: KitchenStatusPending,
		OrderedAt:     orderedAt,
		UpdatedAt:     now,
	}
	if req.CustomerName != nil {
		order.CustomerName = strings.TrimSpace(*req.CustomerName)
	}
	if req.CustomerPhone != nil {
		order.CustomerPhone = strings.TrimSpace(*req.CustomerPhone)
	}
	if req.Notes != nil {
		order.Notes = strings.TrimSpace(*req.Notes)
	}
	if sess.BillID != "" {
		order.BillID = sess.BillID
	}
	if err := s.repo.PutOrder(ctx, &order); err != nil {
		return nil, err
	}
	sess.UpdatedAt = now
	if err := s.repo.PutSession(ctx, sess); err != nil {
		return nil, err
	}
	return &order, nil
}

// UpdateSession patches group-level session fields (pax, notes, service flags, tables).
func (s *Service) UpdateSession(ctx context.Context, req UpdateSessionRequest, clock Clock) (*TableSession, error) {
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id required")
	}
	sess, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	if sess.Status == SessionStatusClosed {
		return nil, fmt.Errorf("session is closed")
	}
	if req.Pax != nil {
		sess.Pax = *req.Pax
	}
	if req.GroupNotes != nil {
		sess.GroupNotes = strings.TrimSpace(*req.GroupNotes)
	}
	if req.ServiceFlags != nil {
		if sess.ServiceFlags == nil {
			sess.ServiceFlags = &SessionServiceFlags{}
		}
		if req.ServiceFlags.WelcomeDrinkServed {
			sess.ServiceFlags.WelcomeDrinkServed = true
		}
		if req.ServiceFlags.ComplementaryServed {
			sess.ServiceFlags.ComplementaryServed = true
		}
		if req.ServiceFlags.KidMenuEnabled {
			sess.ServiceFlags.KidMenuEnabled = true
		}
		if req.ServiceFlags.KidMenuServed {
			sess.ServiceFlags.KidMenuServed = true
		}
	}
	if len(req.TableIDs) > 0 {
		sess.TableIDs = req.TableIDs
	}
	sess.UpdatedAt = clock.GetCurrentTimestamp()
	if err := s.repo.PutSession(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// UpdateOrderWithClock updates items / kitchen status on an order.
func (s *Service) UpdateOrderWithClock(ctx context.Context, req UpdateOrderRequestV2, clock Clock) (*Order, error) {
	o, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("order not found")
	}
	if o.MarkedDoneAt != 0 {
		return nil, fmt.Errorf("order is marked done and cannot be edited")
	}
	now := clock.GetCurrentTimestamp()
	if len(req.Items) > 0 {
		o.Items = ensureLineItemIDs(req.Items)
		o.TotalPrice = sumLineItems(o.Items)
	}
	if req.Notes != nil {
		o.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.KitchenStatus != nil {
		o.KitchenStatus = *req.KitchenStatus
		applyKitchenStatusToAllLineItems(o.Items, *req.KitchenStatus)
		if *req.KitchenStatus == KitchenStatusReady {
			o.ReadyAt = now
		}
		if *req.KitchenStatus == KitchenStatusServed {
			o.CompletedAt = now
		}
	}
	if req.MarkDone != nil && *req.MarkDone {
		o.MarkedDoneAt = now
	}
	if len(req.RemoveLineItemIDs) > 0 {
		toRemove := make(map[string]struct{}, len(req.RemoveLineItemIDs))
		for _, id := range req.RemoveLineItemIDs {
			if id != "" {
				toRemove[id] = struct{}{}
			}
		}
		for i := range o.Items {
			if _, ok := toRemove[o.Items[i].ID]; ok {
				o.Items[i].Removed = true
				o.Items[i] = NormalizeLineItem(o.Items[i])
				for j := range o.Items[i].UnitStates {
					o.Items[i].UnitStates[j] = UnitState{
						Status:       UnitStateCancelled,
						CancelReason: CancelReasonManagerVoid,
						CancelledAt:  now,
					}
				}
				o.Items[i].Status = LineItemStatusCancelled
			}
		}
		o.TotalPrice = sumLineItems(o.Items)
	}
	recomputeOrderKitchenState(o, now)
	if req.TotalPrice != nil {
		o.TotalPrice = *req.TotalPrice
	}
	o.UpdatedAt = now
	if err := s.repo.PutOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Service) ListOrdersBySession(ctx context.Context, sessionID string) ([]Order, error) {
	return s.repo.QueryOrdersBySession(ctx, sessionID)
}

func (s *Service) ListOrdersByTable(ctx context.Context, venueID, tableID string) ([]Order, error) {
	if venueID == "" {
		venueID = defaultVenueID()
	}
	sess, err := s.findLiveSessionForTable(ctx, venueID, tableID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return []Order{}, nil
	}
	return s.repo.QueryOrdersBySession(ctx, sess.ID)
}

func (s *Service) StartBill(ctx context.Context, req StartBillForSessionRequest, staffID string, clock Clock) (*Bill, error) {
	sess, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	if sess.BillID != "" {
		b, err := s.repo.GetBill(ctx, sess.BillID)
		if err != nil {
			return nil, err
		}
		if b != nil {
			return b, nil
		}
	}
	// Idempotency fallback: if session already moved out of live and has a bill row,
	// return the existing bill instead of trying to create a duplicate.
	if sess.Status != SessionStatusLive {
		bills, err := s.repo.QueryBillsBySession(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		if len(bills) > 0 {
			return &bills[0], nil
		}
		return nil, fmt.Errorf("session is not live and no existing bill found")
	}
	now := clock.GetCurrentTimestamp()
	pB := PrefixBill
	bid := clock.GenerateUniqueID(&pB)
	st := staffID
	if req.StaffID != nil && *req.StaffID != "" {
		st = *req.StaffID
	}
	bill := Bill{
		ID:                   bid,
		SessionID:            sess.ID,
		TableIDs:             append([]string(nil), sess.TableIDs...),
		StaffID:              st,
		PaymentMethod:        PaymentMethodCash,
		PaymentStatus:        PaymentStatusPending,
		CreatedAt:            now,
		UpdatedAt:            now,
		Discounts:            nil,
		Taxes:                nil,
		TotalTaxInPaise:      0,
		TotalDiscountInPaise: 0,
		TotalAmountInPaise:   0,
	}
	sess.BillID = bid
	sess.Status = SessionStatusBilling
	sess.UpdatedAt = now
	if err := s.repo.PutBill(ctx, &bill); err != nil {
		return nil, err
	}
	if err := s.repo.PutSession(ctx, sess); err != nil {
		return nil, err
	}
	orders, err := s.repo.QueryOrdersBySession(ctx, sess.ID)
	if err != nil {
		return nil, err
	}
	var total int64
	for i := range orders {
		orders[i].BillID = bid
		total += orders[i].TotalPrice
		if err := s.repo.PutOrder(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}
	bill.TotalAmountInPaise = total
	bill.SubtotalInPaise = total
	bill.UpdatedAt = clock.GetCurrentTimestamp()
	if err := s.repo.PutBill(ctx, &bill); err != nil {
		return nil, err
	}
	return &bill, nil
}

func (s *Service) UpdateBill(ctx context.Context, req UpdateBillRequestV2, clock Clock) (*Bill, error) {
	b, err := s.repo.GetBill(ctx, req.BillID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("bill not found")
	}
	if req.PaymentMethod != nil {
		b.PaymentMethod = *req.PaymentMethod
	}
	if req.PaymentStatus != nil {
		b.PaymentStatus = *req.PaymentStatus
	}

	orders, err := s.repo.QueryOrdersBySession(ctx, b.SessionID)
	if err != nil {
		return nil, err
	}
	if len(req.LineItemUpdates) > 0 {
		orderByID := make(map[string]*Order, len(orders))
		for i := range orders {
			orderByID[orders[i].ID] = &orders[i]
		}
		for _, upd := range req.LineItemUpdates {
			order := orderByID[upd.OrderID]
			if order == nil {
				return nil, fmt.Errorf("order not found in bill session: %s", upd.OrderID)
			}
			found := false
			for i := range order.Items {
				if order.Items[i].ID != upd.LineItemID {
					continue
				}
				found = true
				if upd.UserOverride != nil {
					if upd.UserOverride.Quantity != nil && *upd.UserOverride.Quantity <= 0 {
						return nil, fmt.Errorf("user_override.quantity must be > 0 for line item %s", upd.LineItemID)
					}
					order.Items[i].UserOverride = upd.UserOverride
				}
				if upd.Removed != nil {
					order.Items[i].Removed = *upd.Removed
					if *upd.Removed {
						order.Items[i].Status = LineItemStatusCancelled
					}
				}
				break
			}
			if !found {
				return nil, fmt.Errorf("line item not found in order: %s", upd.LineItemID)
			}
		}

		for i := range orders {
			orders[i].TotalPrice = sumLineItems(orders[i].Items)
			orders[i].UpdatedAt = clock.GetCurrentTimestamp()
			if err := s.repo.PutOrder(ctx, &orders[i]); err != nil {
				return nil, err
			}
		}
	}
	var recomputedTotal int64
	for i := range orders {
		recomputedTotal += sumLineItems(orders[i].Items)
	}
	b.SubtotalInPaise = recomputedTotal
	if len(req.Discounts) > 0 {
		b.Discounts = req.Discounts
	}
	if len(req.Taxes) > 0 {
		b.Taxes = req.Taxes
	}
	if req.StaffWelfareInPaise != nil {
		b.StaffWelfareInPaise = *req.StaffWelfareInPaise
	}
	recomputeBillPayable(b)
	b.UpdatedAt = clock.GetCurrentTimestamp()
	if err := s.repo.PutBill(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) CloseTable(ctx context.Context, req CloseTableRequest, clock Clock) error {
	sess, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return fmt.Errorf("session not found")
	}
	if req.BillID == "" || sess.BillID != req.BillID {
		return fmt.Errorf("bill_id mismatch")
	}
	b, err := s.repo.GetBill(ctx, req.BillID)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("bill not found")
	}
	now := clock.GetCurrentTimestamp()
	b.PaymentStatus = PaymentStatusPaid
	b.UpdatedAt = now
	sess.Status = SessionStatusClosed
	sess.ClosedAt = now
	sess.UpdatedAt = now
	if err := s.repo.PutBill(ctx, b); err != nil {
		return err
	}
	return s.repo.PutSession(ctx, sess)
}

// --- Kitchen ---

func (s *Service) KitchenItemBoard(ctx context.Context, venueID string) ([]KitchenDishCount, error) {
	if venueID == "" {
		venueID = defaultVenueID()
	}
	orders, err := s.repo.QueryOrdersByVenue(ctx, venueID, 500)
	if err != nil {
		return nil, err
	}
	pendingByKey := make(map[string]*KitchenDishCount)
	for _, o := range orders {
		if o.KitchenStatus == KitchenStatusServed || o.KitchenStatus == KitchenStatusCancelled {
			continue
		}
		for _, li := range o.Items {
			li = NormalizeLineItem(li)
			pending := 0
			for _, u := range li.UnitStates {
				if u.Status == UnitStatePending {
					pending++
				}
			}
			if pending == 0 {
				continue
			}
			key := o.ID + "::" + li.ID
			row, ok := pendingByKey[key]
			if !ok {
				pendingByKey[key] = &KitchenDishCount{
					OrderID:    o.ID,
					LineItemID: li.ID,
					Name:       li.Name,
					Category:   li.Category,
					Quantity:   pending,
					Status:     LineItemStatusPending,
				}
				continue
			}
			row.Quantity += pending
		}
	}
	rows := make([]KitchenDishCount, 0, len(pendingByKey))
	for _, row := range pendingByKey {
		rows = append(rows, *row)
	}
	return rows, nil
}

// KitchenUnitBoard returns one row per pending unit for FCFS kitchen fulfillment.
func (s *Service) KitchenUnitBoard(ctx context.Context, venueID string) ([]KitchenUnitRow, error) {
	if venueID == "" {
		venueID = defaultVenueID()
	}
	orders, err := s.repo.QueryOrdersByVenue(ctx, venueID, 500)
	if err != nil {
		return nil, err
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].OrderedAt < orders[j].OrderedAt })
	var rows []KitchenUnitRow
	for _, o := range orders {
		if o.KitchenStatus == KitchenStatusServed || o.KitchenStatus == KitchenStatusCancelled {
			continue
		}
		for _, li := range o.Items {
			li = NormalizeLineItem(li)
			for idx, u := range li.UnitStates {
				if u.Status != UnitStatePending {
					continue
				}
				rows = append(rows, KitchenUnitRow{
					OrderID:    o.ID,
					LineItemID: li.ID,
					UnitIndex:  idx,
					Name:       li.Name,
					Category:   li.Category,
					UnitStatus: u.Status,
					OrderedAt:  o.OrderedAt,
					OrderLabel: o.ID,
				})
			}
		}
	}
	return rows, nil
}

func (s *Service) PatchLineItemStatus(ctx context.Context, req PatchLineItemStatusRequest, clock Clock) (*Order, error) {
	o, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("order not found")
	}
	found := false
	for i := range o.Items {
		if o.Items[i].ID != req.LineItemID {
			continue
		}
		found = true
		o.Items[i] = NormalizeLineItem(o.Items[i])
		for j := range o.Items[i].UnitStates {
			switch req.Status {
			case LineItemStatusCancelled:
				o.Items[i].UnitStates[j] = UnitState{Status: UnitStateCancelled, CancelReason: CancelReasonManagerVoid}
			case LineItemStatusReady, LineItemStatusServed:
				if o.Items[i].UnitStates[j].Status != UnitStateCancelled {
					o.Items[i].UnitStates[j].Status = UnitStateFulfilled
				}
			default:
				if o.Items[i].UnitStates[j].Status != UnitStateCancelled {
					o.Items[i].UnitStates[j].Status = UnitStatePending
				}
			}
		}
		o.Items[i].Status = aggregateLineStatus(o.Items[i].UnitStates)
		break
	}
	if !found {
		return nil, fmt.Errorf("line item not found")
	}
	now := clock.GetCurrentTimestamp()
	recomputeOrderKitchenState(o, now)
	o.UpdatedAt = now
	if err := s.repo.PutOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Service) PatchLineItemUnit(ctx context.Context, req PatchLineItemUnitRequest, clock Clock) (*Order, error) {
	if req.OrderID == "" || req.LineItemID == "" || req.UnitIndex < 0 {
		return nil, fmt.Errorf("order_id, line_item_id, and unit_index required")
	}
	o, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("order not found")
	}
	if o.MarkedDoneAt != 0 {
		return nil, fmt.Errorf("order is marked done and cannot be edited")
	}
	found := false
	now := clock.GetCurrentTimestamp()
	for i := range o.Items {
		if o.Items[i].ID != req.LineItemID {
			continue
		}
		found = true
		o.Items[i] = NormalizeLineItem(o.Items[i])
		if req.UnitIndex >= len(o.Items[i].UnitStates) {
			return nil, fmt.Errorf("unit_index out of range")
		}
		switch req.Action {
		case "fulfill":
			if o.Items[i].UnitStates[req.UnitIndex].Status != UnitStateCancelled {
				o.Items[i].UnitStates[req.UnitIndex].Status = UnitStateFulfilled
			}
		case "unfulfill":
			if o.Items[i].UnitStates[req.UnitIndex].Status == UnitStateFulfilled {
				o.Items[i].UnitStates[req.UnitIndex].Status = UnitStatePending
			}
		case "cancel":
			reason := req.CancelReason
			if reason == "" {
				reason = CancelReasonWaiterCancel
			}
			if !validCancelReason(reason) {
				return nil, fmt.Errorf("invalid cancel_reason")
			}
			o.Items[i].UnitStates[req.UnitIndex] = UnitState{
				Status:       UnitStateCancelled,
				CancelReason: reason,
				CancelledAt:  now,
			}
		case "toggle_parcel":
			o.Items[i].ParcelUnits = padParcelUnits(o.Items[i].ParcelUnits, o.Items[i].Quantity)
			o.Items[i].ParcelUnits[req.UnitIndex] = !o.Items[i].ParcelUnits[req.UnitIndex]
		default:
			return nil, fmt.Errorf("unknown action: %s", req.Action)
		}
		o.Items[i].Status = aggregateLineStatus(o.Items[i].UnitStates)
		break
	}
	if !found {
		return nil, fmt.Errorf("line item not found")
	}
	o.TotalPrice = sumLineItems(o.Items)
	recomputeOrderKitchenState(o, now)
	o.UpdatedAt = now
	if err := s.repo.PutOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// --- Plating ---

func (s *Service) PlatingFIFO(ctx context.Context, venueID, tableID, sessionID string, limit int) ([]PlatingQueueOrder, error) {
	if venueID == "" {
		venueID = defaultVenueID()
	}
	if limit <= 0 {
		limit = 100
	}
	var orders []Order
	var err error
	if sessionID != "" {
		orders, err = s.repo.QueryOrdersBySession(ctx, sessionID)
	} else if tableID != "" {
		sess, e := s.findLiveSessionForTable(ctx, venueID, tableID)
		if e != nil {
			return nil, e
		}
		if sess == nil {
			return nil, nil
		}
		orders, err = s.repo.QueryOrdersBySession(ctx, sess.ID)
	} else {
		orders, err = s.repo.QueryOrdersByVenue(ctx, venueID, int32(limit))
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].OrderedAt < orders[j].OrderedAt })
	if len(orders) > limit {
		orders = orders[:limit]
	}
	sessionTables := make(map[string][]string)
	out := make([]PlatingQueueOrder, 0, len(orders))
	for _, o := range orders {
		if o.KitchenStatus == KitchenStatusServed {
			continue
		}
		tids, ok := sessionTables[o.SessionID]
		if !ok {
			sess, e := s.repo.GetSession(ctx, o.SessionID)
			if e != nil {
				return nil, e
			}
			if sess != nil {
				tids = sess.TableIDs
			}
			sessionTables[o.SessionID] = tids
		}
		out = append(out, PlatingQueueOrder{
			OrderID:       o.ID,
			SessionID:     o.SessionID,
			TableIDs:      tids,
			Items:         o.Items,
			KitchenStatus: o.KitchenStatus,
			OrderedAt:     o.OrderedAt,
		})
	}
	return out, nil
}

func (s *Service) PatchOrderKitchenStatus(ctx context.Context, req PatchOrderKitchenStatusRequestV2, clock Clock) (*Order, error) {
	o, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("order not found")
	}
	o.KitchenStatus = req.KitchenStatus
	applyKitchenStatusToAllLineItems(o.Items, req.KitchenStatus)
	now := clock.GetCurrentTimestamp()
	o.UpdatedAt = now
	if req.KitchenStatus == KitchenStatusReady {
		o.ReadyAt = now
	}
	if req.KitchenStatus == KitchenStatusServed {
		o.CompletedAt = now
	}
	if err := s.repo.PutOrder(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}
