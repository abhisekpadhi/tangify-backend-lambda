package billing

func sumTaxesBill(taxes []TaxType) int64 {
	var total int64
	for _, t := range taxes {
		total += t.AmountInPaise
	}
	return total
}

func sumDiscountsBill(discounts []DiscountType) int64 {
	var total int64
	for _, d := range discounts {
		total += d.Amount
	}
	return total
}

func recomputeBillPayable(b *Bill) {
	if b == nil {
		return
	}
	discount := sumDiscountsBill(b.Discounts)
	tax := sumTaxesBill(b.Taxes)
	b.TotalDiscountInPaise = discount
	b.TotalTaxInPaise = tax
	payable := b.SubtotalInPaise - discount + tax + b.StaffWelfareInPaise
	if payable < 0 {
		payable = 0
	}
	b.TotalAmountInPaise = payable
}
