package billing

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

type BillWithLineItemsService struct {
	repo   *BillWithLineItemsRepository
	wallet PointsWalletProvider
}

func NewBillWithLineItemsService(repo *BillWithLineItemsRepository, wallet PointsWalletProvider) *BillWithLineItemsService {
	return &BillWithLineItemsService{repo: repo, wallet: wallet}
}

func (s *BillWithLineItemsService) Get(ctx context.Context, billID string) (*BillWithLineItems, error) {
	billID = strings.TrimSpace(billID)
	if billID == "" {
		return nil, fmt.Errorf("bill_id required")
	}
	return s.repo.Get(ctx, billID)
}

func (s *BillWithLineItemsService) Upsert(
	ctx context.Context,
	req UpsertBillWithLineItemsRequest,
	staffID string,
	now int64,
) (*BillWithLineItems, error) {
	id := strings.TrimSpace(req.ID)
	stateKey := strings.TrimSpace(req.StateKey)

	if id != "" {
		return s.update(ctx, req, id, staffID, now)
	}
	if stateKey == "" {
		return nil, errStateKeyRequired
	}
	return s.create(ctx, req, stateKey, staffID, now)
}

const pointsWalletTableName = "tangify_points_wallet"

func invoiceWorkerBillID(stateKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(stateKey)))
	return fmt.Sprintf("%s_state_%x", PrefixBill, sum[:16])
}

func (s *BillWithLineItemsService) create(
	ctx context.Context,
	req UpsertBillWithLineItemsRequest,
	stateKey string,
	staffID string,
	now int64,
) (*BillWithLineItems, error) {
	existing, err := s.repo.GetByStateKey(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	var pointsBalance int64
	if hasPointsDiscount(req.Discounts) {
		customerID := strings.TrimSpace(req.CustomerID)
		if customerID == "" {
			return nil, errCustomerIDRequiredForPoints
		}
		pointsBalance, err = s.wallet.GetPointsBalance(ctx, customerID)
		if err != nil {
			return nil, err
		}
	}

	totals, err := computeBillTotals(
		req.LineItems,
		req.Discounts,
		req.Taxes,
		req.CustomerID,
		pointsBalance,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// The invoice worker is idempotent by bill_id. Deriving that key from the
	// checkout state makes concurrent clients and ambiguous retries converge on
	// the same invoice number.
	workerBillID := invoiceWorkerBillID(stateKey)
	inv, err := FetchInvoiceNumber(ctx, workerBillID)
	if err != nil {
		return nil, err
	}

	bill := requestToBill(req, staffID, now)
	bill.ID = inv.InvoiceNumber
	bill.StateKey = stateKey
	bill.CreatedAt = now
	bill.UpdatedAt = now
	bill.Discounts = totals.Discounts
	bill.TotalDiscountInPaise = totals.TotalDiscountInPaise
	bill.TotalTaxInPaise = totals.TotalTaxInPaise
	bill.TotalAmountInPaise = totals.TotalAmountInPaise
	if bill.PaymentMethod == "" {
		bill.PaymentMethod = PaymentMethodCash
	}
	if bill.PaymentStatus == "" {
		bill.PaymentStatus = PaymentStatusPending
	}

	err = s.repo.TransactCreate(
		ctx,
		bill,
		strings.TrimSpace(req.CustomerID),
		totals.PointsRedeemed,
		pointsWalletTableName,
		now,
	)
	if err != nil {
		// A concurrent request with the same state key receives the same invoice
		// number. Its conditional put can lose the race, so return the winner.
		existing, getErr := s.repo.Get(ctx, inv.InvoiceNumber)
		if getErr == nil && existing != nil && existing.StateKey == stateKey {
			return existing, nil
		}
		return nil, err
	}
	return bill, nil
}

func (s *BillWithLineItemsService) update(
	ctx context.Context,
	req UpsertBillWithLineItemsRequest,
	id string,
	staffID string,
	now int64,
) (*BillWithLineItems, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrBillNotFound
	}

	totals, err := computeBillTotals(
		req.LineItems,
		req.Discounts,
		req.Taxes,
		req.CustomerID,
		0,
		true,
		existing.Discounts,
	)
	if err != nil {
		return nil, err
	}

	bill := requestToBill(req, staffID, now)
	bill.ID = id
	bill.StateKey = existing.StateKey
	bill.CreatedAt = existing.CreatedAt
	bill.UpdatedAt = now
	bill.Discounts = totals.Discounts
	bill.TotalDiscountInPaise = totals.TotalDiscountInPaise
	bill.TotalTaxInPaise = totals.TotalTaxInPaise
	bill.TotalAmountInPaise = totals.TotalAmountInPaise
	if bill.PaymentMethod == "" {
		bill.PaymentMethod = existing.PaymentMethod
	}
	if bill.PaymentStatus == "" {
		bill.PaymentStatus = existing.PaymentStatus
	}
	if bill.CustomerID == "" {
		bill.CustomerID = existing.CustomerID
	}
	if bill.SessionID == "" {
		bill.SessionID = existing.SessionID
	}
	if len(bill.TableIDs) == 0 {
		bill.TableIDs = existing.TableIDs
	}

	if err := s.repo.Put(ctx, bill); err != nil {
		return nil, err
	}
	return bill, nil
}

func requestToBill(req UpsertBillWithLineItemsRequest, staffID string, now int64) *BillWithLineItems {
	bill := &BillWithLineItems{
		SessionID:  req.SessionID,
		TableIDs:   append([]string(nil), req.TableIDs...),
		CustomerID: strings.TrimSpace(req.CustomerID),
		LineItems:  append([]LineItemV0(nil), req.LineItems...),
		Discounts:  append([]DiscountType(nil), req.Discounts...),
		Taxes:      append([]TaxType(nil), req.Taxes...),
		UpdatedAt:  now,
	}
	if strings.TrimSpace(req.StaffID) != "" {
		bill.StaffID = strings.TrimSpace(req.StaffID)
	} else {
		bill.StaffID = staffID
	}
	if req.PaymentMethod != nil {
		bill.PaymentMethod = *req.PaymentMethod
	}
	if req.PaymentStatus != nil {
		bill.PaymentStatus = *req.PaymentStatus
	}
	return bill
}
