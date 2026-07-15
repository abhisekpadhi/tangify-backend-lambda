package billing

import (
	"strings"

	"github.com/google/uuid"
)

// NormalizeChannel maps UI OrderKind aliases to canonical OrderChannel* values.
func NormalizeChannel(channel string) string {
	switch strings.TrimSpace(strings.ToLower(channel)) {
	case OrderChannelTable:
		return OrderChannelDiningTable
	case OrderChannelTakeaway:
		return OrderChannelTakeaway
	case OrderChannelDelivery:
		return OrderChannelNeighbourDelivery
	case OrderChannelDiningTable, OrderChannelWhatsAppQuickDelivery, OrderChannelWhatsAppNormalDelivery, OrderChannelNeighbourDelivery:
		return strings.TrimSpace(channel)
	default:
		return strings.TrimSpace(channel)
	}
}

func validCancelReason(reason string) bool {
	switch reason {
	case CancelReasonCustomerCancel, CancelReasonWaiterCancel, CancelReasonKitchenOutOfStock,
		CancelReasonKitchenUnableToPrepare, CancelReasonWrongOrder, CancelReasonDuplicateOrder,
		CancelReasonQualityIssue, CancelReasonManagerVoid:
		return true
	default:
		return false
	}
}

// NormalizeLineItem ensures unit_states and parcel_units match quantity and syncs aggregate status.
func NormalizeLineItem(li LineItem) LineItem {
	if li.Quantity <= 0 {
		li.Quantity = 1
	}
	if len(li.UnitStates) == 0 {
		li.UnitStates = legacyUnitStates(li)
	}
	li.UnitStates = padUnitStates(li.UnitStates, li.Quantity, li.Status)
	li.ParcelUnits = padParcelUnits(li.ParcelUnits, li.Quantity)
	li.Status = aggregateLineStatus(li.UnitStates)
	if li.Removed {
		li.Status = LineItemStatusCancelled
	}
	return li
}

func legacyUnitStates(li LineItem) []UnitState {
	states := make([]UnitState, li.Quantity)
	for i := range states {
		switch li.Status {
		case LineItemStatusCancelled:
			states[i] = UnitState{Status: UnitStateCancelled, CancelReason: CancelReasonManagerVoid}
		case LineItemStatusReady, LineItemStatusServed:
			states[i] = UnitState{Status: UnitStateFulfilled}
		default:
			states[i] = UnitState{Status: UnitStatePending}
		}
	}
	return states
}

func padUnitStates(states []UnitState, qty int, legacyStatus string) []UnitState {
	out := make([]UnitState, qty)
	for i := 0; i < qty; i++ {
		if i < len(states) {
			out[i] = states[i]
			if out[i].Status == "" {
				out[i].Status = UnitStatePending
			}
			continue
		}
		if legacyStatus == LineItemStatusCancelled {
			out[i] = UnitState{Status: UnitStateCancelled, CancelReason: CancelReasonManagerVoid}
		} else {
			out[i] = UnitState{Status: UnitStatePending}
		}
	}
	return out
}

func padParcelUnits(units []bool, qty int) []bool {
	out := make([]bool, qty)
	for i := 0; i < qty; i++ {
		if i < len(units) {
			out[i] = units[i]
		}
	}
	return out
}

func aggregateLineStatus(states []UnitState) string {
	if len(states) == 0 {
		return LineItemStatusPending
	}
	allCancelled := true
	allFulfilled := true
	anyFulfilled := false
	for _, s := range states {
		switch s.Status {
		case UnitStateCancelled:
			allFulfilled = false
		case UnitStateFulfilled:
			anyFulfilled = true
			allCancelled = false
		default:
			allCancelled = false
			allFulfilled = false
		}
	}
	if allCancelled {
		return LineItemStatusCancelled
	}
	if allFulfilled {
		return LineItemStatusReady
	}
	if anyFulfilled {
		return LineItemStatusPreparing
	}
	return LineItemStatusPending
}

// BillableQty returns non-cancelled unit count for a line item.
func BillableQty(li LineItem) int {
	li = NormalizeLineItem(li)
	if li.Removed {
		return 0
	}
	n := 0
	for _, s := range li.UnitStates {
		if s.Status != UnitStateCancelled {
			n++
		}
	}
	return n
}

func lineItemEffectiveQty(li LineItem) int {
	qty := li.Quantity
	if li.UserOverride != nil && li.UserOverride.Quantity != nil && *li.UserOverride.Quantity > 0 {
		qty = *li.UserOverride.Quantity
	}
	return qty
}

func lineItemEffectivePrice(li LineItem) int64 {
	if li.UserOverride != nil && li.UserOverride.Price != nil {
		return *li.UserOverride.Price
	}
	return li.Price
}

// SumLineItems totals billable units × price across line items.
func SumLineItems(items []LineItem) int64 {
	var total int64
	for _, li := range items {
		if li.Removed {
			continue
		}
		li = NormalizeLineItem(li)
		billable := BillableQty(li)
		if billable <= 0 {
			continue
		}
		price := lineItemEffectivePrice(li)
		total += price * int64(billable)
	}
	return total
}

func ensureLineItemIDs(items []LineItem) []LineItem {
	out := make([]LineItem, len(items))
	for i, li := range items {
		out[i] = NormalizeLineItem(li)
		if out[i].ID == "" {
			out[i].ID = PrefixLine + "_" + uuid.NewString()
		}
	}
	return out
}

func applyKitchenStatusToAllLineItems(items []LineItem, kitchenStatus string) {
	for i := range items {
		items[i] = NormalizeLineItem(items[i])
		for j := range items[i].UnitStates {
			switch kitchenStatus {
			case KitchenStatusCancelled:
				items[i].UnitStates[j] = UnitState{Status: UnitStateCancelled, CancelReason: CancelReasonManagerVoid}
			case KitchenStatusReady, KitchenStatusServed:
				if items[i].UnitStates[j].Status != UnitStateCancelled {
					items[i].UnitStates[j].Status = UnitStateFulfilled
				}
			default:
				if items[i].UnitStates[j].Status != UnitStateCancelled {
					items[i].UnitStates[j].Status = UnitStatePending
				}
			}
		}
		items[i].Status = aggregateLineStatus(items[i].UnitStates)
	}
}

func orderHasPendingUnits(o Order) bool {
	for _, li := range o.Items {
		li = NormalizeLineItem(li)
		for _, s := range li.UnitStates {
			if s.Status == UnitStatePending {
				return true
			}
		}
	}
	return false
}

func recomputeOrderKitchenState(o *Order, now int64) {
	if orderHasPendingUnits(*o) {
		o.KitchenStatus = KitchenStatusPending
		o.ReadyAt = 0
		return
	}
	o.KitchenStatus = KitchenStatusReady
	if o.ReadyAt == 0 {
		o.ReadyAt = now
	}
}
