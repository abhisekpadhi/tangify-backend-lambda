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

func TestComputeBillTotals_pointsHonorsRequestedCappedByWallet(t *testing.T) {
	items := []LineItemV0{{Name: "Thali", Quantity: 1, Price: 100000}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints, Amount: 99999}}

	got, err := computeBillTotals(items, discounts, nil, "cust-1", 3, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 3 points * 300 = 900 paise
	if got.TotalDiscountInPaise != 900 {
		t.Fatalf("discount=%d want=900", got.TotalDiscountInPaise)
	}
	if got.PointsRedeemed != 3 {
		t.Fatalf("points=%d want=3", got.PointsRedeemed)
	}
	if got.TotalAmountInPaise != 99100 {
		t.Fatalf("total=%d want=99100", got.TotalAmountInPaise)
	}
}

func TestComputeBillTotals_pointsHonorsExactAmount(t *testing.T) {
	items := []LineItemV0{{Name: "Thali", Quantity: 1, Price: 100000}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints, Amount: 600}}

	got, err := computeBillTotals(items, discounts, nil, "cust-1", 10, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.PointsRedeemed != 2 {
		t.Fatalf("points=%d want=2", got.PointsRedeemed)
	}
	if got.TotalDiscountInPaise != 600 {
		t.Fatalf("discount=%d want=600", got.TotalDiscountInPaise)
	}
}

func TestComputeBillTotals_pointsCappedBySubtotal(t *testing.T) {
	items := []LineItemV0{{Name: "Snack", Quantity: 1, Price: 500}}
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints, Amount: 99999}}

	got, err := computeBillTotals(items, discounts, nil, "cust-1", 10, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 500 paise / 300 = 1 point
	if got.TotalDiscountInPaise != 300 {
		t.Fatalf("discount=%d want=300", got.TotalDiscountInPaise)
	}
	if got.PointsRedeemed != 1 {
		t.Fatalf("points=%d want=1", got.PointsRedeemed)
	}
}

func TestComputeBillTotals_pointsExclusive(t *testing.T) {
	items := []LineItemV0{{Name: "Thali", Quantity: 1, Price: 100000}}
	discounts := []DiscountType{
		{ID: "points", Type: DiscountTypePoints, Amount: 600},
		{ID: "mem", Type: DiscountTypeMembership, Amount: 1000},
	}
	_, err := computeBillTotals(items, discounts, nil, "cust-1", 10, false, nil)
	if err != errPointsExclusive {
		t.Fatalf("err=%v want=%v", err, errPointsExclusive)
	}
}

func TestPointsEarnedFromDiscountedSubtotal(t *testing.T) {
	if got := PointsEarnedFromDiscountedSubtotal(100000, 900); got != 19 {
		t.Fatalf("earned=%d want=19", got)
	}
	if got := PointsEarnedFromDiscountedSubtotal(4900, 0); got != 0 {
		t.Fatalf("earned=%d want=0", got)
	}
}

func TestComputeBillTotals_updateFreezesPoints(t *testing.T) {
	existing := []DiscountType{{
		ID: "points", Type: DiscountTypePoints, Amount: 5000, Description: "2 points redeemed",
	}}
	items := []LineItemV0{{Name: "Snack", Quantity: 1, Price: 10000}}
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
	discounts := []DiscountType{{ID: "points", Type: DiscountTypePoints, Amount: 600}}

	_, err := computeBillTotals(items, discounts, nil, "", 5, false, nil)
	if err != errCustomerIDRequiredForPoints {
		t.Fatalf("err=%v", err)
	}
}
