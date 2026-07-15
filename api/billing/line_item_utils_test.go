package billing

import "testing"

func TestNormalizeLineItem_unitStates(t *testing.T) {
	li := NormalizeLineItem(LineItem{
		Name:     "Dal",
		Quantity: 3,
		Price:    10000,
		UnitStates: []UnitState{
			{Status: UnitStatePending},
			{Status: UnitStateFulfilled},
			{Status: UnitStateCancelled, CancelReason: CancelReasonCustomerCancel},
		},
	})
	if len(li.UnitStates) != 3 {
		t.Fatalf("len=%d", len(li.UnitStates))
	}
	if BillableQty(li) != 2 {
		t.Fatalf("billable=%d want 2", BillableQty(li))
	}
	if li.Status != LineItemStatusPreparing {
		t.Fatalf("status=%s", li.Status)
	}
}

func TestSumLineItems_skipsCancelledUnits(t *testing.T) {
	total := SumLineItems([]LineItem{{
		Name:     "Thali",
		Quantity: 2,
		Price:    50000,
		UnitStates: []UnitState{
			{Status: UnitStateFulfilled},
			{Status: UnitStateCancelled, CancelReason: CancelReasonKitchenOutOfStock},
		},
	}})
	if total != 50000 {
		t.Fatalf("total=%d want 50000", total)
	}
}

func TestNormalizeChannel(t *testing.T) {
	if got := NormalizeChannel("table"); got != OrderChannelDiningTable {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeChannel("delivery"); got != OrderChannelNeighbourDelivery {
		t.Fatalf("got %q", got)
	}
}

func TestRecomputeBillPayable(t *testing.T) {
	b := &Bill{
		SubtotalInPaise:     100000,
		StaffWelfareInPaise: 10000,
		Discounts:           []DiscountType{{Amount: 5000}},
		Taxes:               []TaxType{{AmountInPaise: 2500}},
	}
	recomputeBillPayable(b)
	if b.TotalAmountInPaise != 107500 {
		t.Fatalf("payable=%d want 107500", b.TotalAmountInPaise)
	}
}
