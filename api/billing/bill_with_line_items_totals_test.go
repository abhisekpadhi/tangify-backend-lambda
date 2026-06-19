package billing

import (
	"testing"
)

func TestComputeBillTotals_subtotalAndTax(t *testing.T) {
	items := []LineItemV0{
		{Name: "Dal", Quantity: 2, Price: 15000},
		{Name: "Rice", Quantity: 1, Price: 8000},
	}
	taxes := []TaxType{{ID: "gst", Name: "GST", RateInBps: 500, AmountInPaise: 1900}}

	got, err := computeBillTotals(items, nil, taxes, "", 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSubtotal := int64(38000)
	if got.TotalAmountInPaise != wantSubtotal+1900 {
		t.Fatalf("total=%d want=%d", got.TotalAmountInPaise, wantSubtotal+1900)
	}
	if got.TotalTaxInPaise != 1900 {
		t.Fatalf("tax=%d", got.TotalTaxInPaise)
	}
}

func TestComputeBillTotals_pointsFullBalance(t *testing.T) {
	items := []LineItemV0{{Name: "Thali", Quantity: 1, Price: 100000}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints, Amount: 99999}}

	got, err := computeBillTotals(items, discounts, nil, "cust-1", 3, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 3 points * 2500 = 7500 paise discount
	if got.TotalDiscountInPaise != 7500 {
		t.Fatalf("discount=%d want=7500", got.TotalDiscountInPaise)
	}
	if got.PointsRedeemed != 3 {
		t.Fatalf("points=%d want=3", got.PointsRedeemed)
	}
	if got.TotalAmountInPaise != 92500 {
		t.Fatalf("total=%d want=92500", got.TotalAmountInPaise)
	}
}

func TestComputeBillTotals_pointsCappedBySubtotal(t *testing.T) {
	items := []LineItemV0{{Name: "Snack", Quantity: 1, Price: 5000}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints}}

	got, err := computeBillTotals(items, discounts, nil, "cust-1", 10, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 10 points = 25000 paise but subtotal is only 5000
	if got.TotalDiscountInPaise != 5000 {
		t.Fatalf("discount=%d want=5000", got.TotalDiscountInPaise)
	}
	if got.PointsRedeemed != 2 {
		t.Fatalf("points=%d want=2", got.PointsRedeemed)
	}
}

func TestComputeBillTotals_updateFreezesPoints(t *testing.T) {
	existing := []DiscountType{{
		ID: "points", Type: DiscountTypePoints, Amount: 5000, Description: "2 points redeemed",
	}}
	items := []LineItemV0{{Name: "Snack", Quantity: 1, Price: 10000}}
	// Caller tries to change points discount on update — should be ignored.
	discounts := []DiscountType{
		{ID: "points", Type: DiscountTypePoints, Amount: 99999},
		{ID: "comp", Type: "comp", Amount: 1000},
	}

	got, err := computeBillTotals(items, discounts, nil, "", 0, true, existing)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalDiscountInPaise != 6000 {
		t.Fatalf("discount=%d want=6000", got.TotalDiscountInPaise)
	}
}

func TestComputeBillTotals_emptyLineItems(t *testing.T) {
	_, err := computeBillTotals(nil, nil, nil, "", 0, false, nil)
	if err != errEmptyLineItems {
		t.Fatalf("err=%v", err)
	}
}

func TestComputeBillTotals_customerRequiredForPoints(t *testing.T) {
	items := []LineItemV0{{Name: "Meal", Quantity: 1, Price: 50000}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints}}

	_, err := computeBillTotals(items, discounts, nil, "", 5, false, nil)
	if err != errCustomerIDRequiredForPoints {
		t.Fatalf("err=%v", err)
	}
}
