package billing

// DynamoDB table names (see dynamodb/billing-v2/*.json).
const (
	TableNameSessions           = "tangify_sessions"
	TableNameOrders             = "tangify_orders"
	TableNameBills              = "tangify_bills"
	TableNameBillsWithLineItems = "tangify_bills_with_line_items"
)

const (
	GSIVenueOpened    = "GSI_VenueOpened"
	GSISessionOrdered = "GSI_SessionOrdered"
	GSIVenueOrdered   = "GSI_VenueOrdered"
	GSISessionBill    = "GSI_SessionBill"
	GSIStateKey       = "GSI_StateKey"
)

// ID prefixes for GenerateUniqueID.
const (
	PrefixSession = "sess"
	PrefixOrder   = "ord"
	PrefixBill    = "bill"
	PrefixLine    = "line"
)

const (
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"

	PaymentMethodCash         = "cash"
	PaymentMethodCard         = "card"
	PaymentMethodUPI          = "upi"
	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodCheque       = "cheque"
	PaymentMethodOther        = "other"
)

const (
	DiscountTypePoints     = "points"
	DiscountTypeMembership = "membership"
	DiscountTypeComp       = "comp"
)

const (
	ChargeTypeStaffWelfare = "staff_welfare"
	ChargeTypePackaging    = "packaging"
	ChargeTypeWater        = "water"
)

// Per-unit kitchen states (mirrors UI unitStates).
const (
	UnitStatePending    = "pending"
	UnitStateFulfilled  = "fulfilled"
	UnitStateCancelled  = "cancelled"
)

// ItemCancelReason matches houseofodia-menu ItemCancelReason.
const (
	CancelReasonCustomerCancel         = "customer_cancel"
	CancelReasonWaiterCancel           = "waiter_cancel"
	CancelReasonKitchenOutOfStock      = "kitchen_out_of_stock"
	CancelReasonKitchenUnableToPrepare = "kitchen_unable_to_prepare"
	CancelReasonWrongOrder             = "wrong_order"
	CancelReasonDuplicateOrder         = "duplicate_order"
	CancelReasonQualityIssue           = "quality_issue"
	CancelReasonManagerVoid            = "manager_void"
)

// UI OrderKind aliases accepted by NormalizeChannel.
const (
	OrderChannelTable    = "table"
	OrderChannelDelivery = "delivery"
)

// PaisePerPoint is the redemption value of one loyalty point (Rs 3).
const PaisePerPoint = int64(300)

// PaisePerEarnedPoint is spend (after discounts, before tax) required to earn 1 point (Rs 50).
const PaisePerEarnedPoint = int64(5000)

type DiscountType struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // DiscountType*
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type TaxType struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RateInBps     int    `json:"rate_in_bps"`
	AmountInPaise int64  `json:"amount_in_paise"`
}
