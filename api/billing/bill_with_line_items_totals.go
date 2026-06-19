package billing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	errEmptyLineItems              = errors.New("line_items required")
	errCustomerIDRequiredForPoints = errors.New("customer_id required when applying points discount")
	errStateKeyRequired            = errors.New("state_key required for create")
	errBillIDRequired              = errors.New("id required for update")
	ErrBillNotFound                = errors.New("bill not found")
	errInsufficientPoints          = errors.New("insufficient points balance")
)

func lineItemsSubtotal(items []LineItemV0) int64 {
	var total int64
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}
		total += item.Price * int64(item.Quantity)
	}
	return total
}

func sumTaxes(taxes []TaxType) int64 {
	var total int64
	for _, tax := range taxes {
		total += tax.AmountInPaise
	}
	return total
}

func hasPointsDiscount(discounts []DiscountType) bool {
	for _, d := range discounts {
		if d.Type == DiscountTypePoints {
			return true
		}
	}
	return false
}

type computedTotals struct {
	Discounts            []DiscountType
	TotalDiscountInPaise int64
	TotalTaxInPaise      int64
	TotalAmountInPaise   int64
	PointsRedeemed       int64
}

func computeBillTotals(
	lineItems []LineItemV0,
	discounts []DiscountType,
	taxes []TaxType,
	customerID string,
	pointsBalance int64,
	freezePointsDiscount bool,
	existingDiscounts []DiscountType,
) (computedTotals, error) {
	if len(lineItems) == 0 {
		return computedTotals{}, errEmptyLineItems
	}
	subtotal := lineItemsSubtotal(lineItems)

	outDiscounts := make([]DiscountType, 0, len(discounts))
	var totalDiscount int64
	var pointsRedeemed int64

	if freezePointsDiscount {
		for _, d := range existingDiscounts {
			if d.Type == DiscountTypePoints {
				outDiscounts = append(outDiscounts, d)
				totalDiscount += d.Amount
				if PaisePerPoint > 0 {
					pointsRedeemed = d.Amount / PaisePerPoint
				}
			}
		}
		for _, d := range discounts {
			if d.Type == DiscountTypePoints {
				continue
			}
			outDiscounts = append(outDiscounts, d)
			totalDiscount += d.Amount
		}
	} else {
		var nonPointsDiscount int64
		for _, d := range discounts {
			if d.Type == DiscountTypePoints {
				continue
			}
			outDiscounts = append(outDiscounts, d)
			nonPointsDiscount += d.Amount
		}
		totalDiscount = nonPointsDiscount

		if hasPointsDiscount(discounts) {
			customerID = strings.TrimSpace(customerID)
			if customerID == "" {
				return computedTotals{}, errCustomerIDRequiredForPoints
			}
			pointsDiscount := pointsBalance * PaisePerPoint
			remaining := subtotal - nonPointsDiscount
			if pointsDiscount > remaining {
				pointsDiscount = remaining
			}
			if pointsDiscount < 0 {
				pointsDiscount = 0
			}
			pointsToRedeem := int64(0)
			if PaisePerPoint > 0 {
				pointsToRedeem = pointsDiscount / PaisePerPoint
			}
			if pointsToRedeem > pointsBalance {
				return computedTotals{}, errInsufficientPoints
			}
			if pointsToRedeem > 0 {
				pointsDiscount = pointsToRedeem * PaisePerPoint
				outDiscounts = append(outDiscounts, DiscountType{
					ID:          "points",
					Type:        DiscountTypePoints,
					Amount:      pointsDiscount,
					Description: fmt.Sprintf("%s points redeemed", strconv.FormatInt(pointsToRedeem, 10)),
				})
				totalDiscount += pointsDiscount
				pointsRedeemed = pointsToRedeem
			}
		}
	}

	totalDiscount = capDiscount(totalDiscount, subtotal)
	totalTax := sumTaxes(taxes)
	return computedTotals{
		Discounts:            outDiscounts,
		TotalDiscountInPaise: totalDiscount,
		TotalTaxInPaise:      totalTax,
		TotalAmountInPaise:   subtotal - totalDiscount + totalTax,
		PointsRedeemed:       pointsRedeemed,
	}, nil
}

func capDiscount(discount, subtotal int64) int64 {
	if discount > subtotal {
		return subtotal
	}
	if discount < 0 {
		return 0
	}
	return discount
}
