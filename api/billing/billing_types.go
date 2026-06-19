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
	DiscountTypePoints = "points"
)

// PaisePerPoint is the redemption value of one loyalty point (Rs 25).
const PaisePerPoint = int64(2500)

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
