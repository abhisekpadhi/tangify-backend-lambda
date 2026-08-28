// Package billing — billing_models_v2.go defines flat shapes for DynamoDB: sessions, orders, bills.
// Legacy types live in billing_models.go; migrate callers gradually.
package billing

// --- Where the order was placed (channel) ---

const (
	OrderChannelDiningTable            = "dining_table"
	OrderChannelTakeaway               = "takeaway"
	OrderChannelWhatsAppQuickDelivery  = "whatsapp_quickdelivery"
	OrderChannelWhatsAppNormalDelivery = "whatsapp_normaldelivery"
	OrderChannelNeighbourDelivery      = "neighbour_delivery"
)

// --- Table / party session (open → billing → closed) ---

const (
	SessionStatusLive    = "live"    // seats occupied; orders can be added
	SessionStatusBilling = "billing" // waiter opened checkout; Bill row exists
	SessionStatusClosed  = "closed"  // payment settled; table(s) available again
)

// SessionServiceFlags are table-level service markers (mirrors UI TOrder flags).
type SessionServiceFlags struct {
	WelcomeDrinkServed  bool `json:"welcome_drink_served,omitempty"`
	ComplementaryServed bool `json:"complementary_served,omitempty"`
	KidMenuEnabled      bool `json:"kid_menu_enabled,omitempty"`
	KidMenuServed       bool `json:"kid_menu_served,omitempty"`
}

// TableSession is one seated party: may span one or many physical tables (joined).
// Opened when the first order is placed; closed after billing is done.
type TableSession struct {
	ID           string               `json:"id"`
	TableIDs     []string             `json:"table_ids"`
	Status       string               `json:"status"`
	BillID       string               `json:"bill_id,omitempty"`
	Pax          int                  `json:"pax,omitempty"`
	GroupNotes   string               `json:"group_notes,omitempty"`
	ServiceFlags *SessionServiceFlags `json:"service_flags,omitempty"`
	OpenedAt     int64                `json:"opened_at"`
	ClosedAt     int64                `json:"closed_at,omitempty"`
	UpdatedAt    int64                `json:"updated_at,omitempty"`
	VenueID      string               `json:"venue_id,omitempty"`
}

// --- Kitchen: per line item (kitchen view: counts by dish × order) ---

const (
	LineItemStatusPending   = "pending"
	LineItemStatusPreparing = "preparing"
	LineItemStatusReady     = "ready"
	LineItemStatusServed    = "served"
	LineItemStatusCancelled = "cancelled"
)

// UnitState is one fulfillable unit on a line item (mirrors UI unitStates[]).
type UnitState struct {
	Status       string `json:"status"` // UnitState*
	CancelReason string `json:"cancel_reason,omitempty"`
	CancelledAt  int64  `json:"cancelled_at,omitempty"`
}

// LineItem is one row on a ticket; kitchen progress is tracked per unit in unit_states.
type LineItem struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	MenuItemID   string            `json:"menu_item_id,omitempty"`
	Category     string            `json:"category,omitempty"`
	InternalName string            `json:"internal_name,omitempty"`
	Quantity     int               `json:"quantity"`
	Price        int64             `json:"price"` // paise per unit
	UnitStates   []UnitState       `json:"unit_states,omitempty"`
	ParcelUnits  []bool            `json:"parcel_units,omitempty"`
	UserOverride *LineItemOverride `json:"user_override,omitempty"`
	Removed      bool              `json:"removed,omitempty"`
	Status       string            `json:"status"` // aggregate LineItemStatus*; derived from unit_states
}

type LineItemV0 struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"` // paise (per line or per unit — document in API)
}

type LineItemOverride struct {
	Quantity *int   `json:"quantity,omitempty"`
	Price    *int64 `json:"price,omitempty"`
}

// --- Order-level kitchen / plating (FIFO by table uses order + ordered_at) ---

const (
	KitchenStatusPending   = "pending"
	KitchenStatusPreparing = "preparing"
	KitchenStatusReady     = "ready"
	KitchenStatusServed    = "served"
	KitchenStatusCancelled = "cancelled"
)

// Order is one ticket (kitchen + line items). Persisted as its own item; links to TableSession and optional Bill.
type Order struct {
	ID            string     `json:"id"`
	SessionID     string     `json:"session_id"`
	VenueID       string     `json:"venue_id"`
	Channel       string     `json:"channel"` // OrderChannel*; UI aliases table/delivery normalized on write
	BillID        string     `json:"bill_id,omitempty"`
	SourceTableID string     `json:"source_table_id,omitempty"`
	CustomerID    string     `json:"customer_id,omitempty"`
	CustomerName  string     `json:"customer_name,omitempty"`
	CustomerPhone string     `json:"customer_phone,omitempty"`
	StaffID       string     `json:"staff_id,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	Items         []LineItem `json:"items"`
	TotalPrice    int64      `json:"total_price"` // paise; billable units only
	KitchenStatus string     `json:"kitchen_status"`
	OrderedAt     int64      `json:"ordered_at"`
	ReadyAt       int64      `json:"ready_at,omitempty"`
	MarkedDoneAt  int64      `json:"marked_done_at,omitempty"` // waiter service done; locks edits
	CompletedAt   int64      `json:"completed_at,omitempty"`   // kitchen served
	UpdatedAt     int64      `json:"updated_at,omitempty"`
}

// Bill is created when the waiter starts closing the table (checkout); payment totals live here.
type Bill struct {
	ID                     string         `json:"id"`
	SessionID              string         `json:"session_id"`
	InvoiceNumber          string         `json:"invoice_number,omitempty"`
	LoyaltyUserID          string         `json:"loyalty_user_id,omitempty"`
	LoyaltyPointsProcessed bool           `json:"loyalty_points_processed,omitempty"`
	LoyaltyPointsEarned    int64          `json:"loyalty_points_earned,omitempty"`
	LoyaltyPointsRedeemed  int64          `json:"loyalty_points_redeemed,omitempty"`
	LoyaltyDiscountApplied int64          `json:"loyalty_discount_applied,omitempty"`
	TableIDs               []string       `json:"table_ids"` // snapshot for receipt / audit
	CustomerID             string         `json:"customer_id,omitempty"`
	StaffID                string         `json:"staff_id,omitempty"`
	PaymentMethod          string         `json:"payment_method"` // PaymentMethod*
	PaymentStatus          string         `json:"payment_status"` // PaymentStatus*
	CreatedAt              int64          `json:"created_at"`
	UpdatedAt              int64          `json:"updated_at"`
	Discounts              []DiscountType `json:"discounts,omitempty"`
	Taxes                  []TaxType      `json:"taxes,omitempty"`
	SubtotalInPaise        int64          `json:"subtotal_in_paise"`
	TotalTaxInPaise        int64          `json:"total_tax_in_paise"`
	TotalDiscountInPaise   int64          `json:"total_discount_in_paise"`
	StaffWelfareInPaise    int64          `json:"staff_welfare_in_paise,omitempty"`
	TotalAmountInPaise     int64          `json:"total_amount_in_paise"`
}

// BillWithLineItems
type BillWithLineItems struct {
	ID                   string         `json:"id"` // invoice number - unique globally, autoincrement using cloudflare sql
	SessionID            string         `json:"session_id"`
	TableIDs             []string       `json:"table_ids"`
	StaffID              string         `json:"staff_id"`
	CustomerID           string         `json:"customer_id"` // canonical 91 + 10-digit phone
	PaymentMethod        string         `json:"payment_method"` // PaymentMethod*
	PaymentStatus        string         `json:"payment_status"` // PaymentStatus*
	CreatedAt            int64          `json:"created_at"`
	UpdatedAt            int64          `json:"updated_at"`
	LineItems            []LineItemV0   `json:"line_items"`
	Discounts            []DiscountType `json:"discounts"`
	Taxes                []TaxType      `json:"taxes"`
	TotalTaxInPaise      int64          `json:"total_tax_in_paise"`
	TotalDiscountInPaise int64          `json:"total_discount_in_paise"`
	TotalAmountInPaise   int64          `json:"total_amount_in_paise"`
	StateKey             string         `json:"state_key"` // unique key for the state of the bill - used to generate the invoice number
	Settled              bool           `json:"settled,omitempty"`
	SettledAt            int64          `json:"settled_at,omitempty"`
	LoyaltyPointsProcessed bool         `json:"loyalty_points_processed,omitempty"`
	LoyaltyPointsEarned  int64          `json:"loyalty_points_earned,omitempty"`
	LoyaltyPointsRedeemed int64         `json:"loyalty_points_redeemed,omitempty"`
}

type UpsertBillWithLineItemsRequest struct {
	StateKey      string         `json:"state_key,omitempty"`
	ID            string         `json:"id,omitempty"`
	SessionID     string         `json:"session_id,omitempty"`
	TableIDs      []string       `json:"table_ids,omitempty"`
	StaffID           string         `json:"staff_id,omitempty"`
	CustomerID        string         `json:"customer_id,omitempty"`
	LoyaltyCustomerID string         `json:"loyalty_customer_id,omitempty"`
	LineItems     []LineItemV0   `json:"line_items"`
	Discounts     []DiscountType `json:"discounts,omitempty"`
	Taxes         []TaxType      `json:"taxes,omitempty"`
	PaymentMethod *string        `json:"payment_method,omitempty"`
	PaymentStatus *string        `json:"payment_status,omitempty"`
	Settled       bool           `json:"settled,omitempty"`
}

// --- Read models (assembled in app; not one Dynamo item) ---

// SessionWithOrders is one live (or billing) party with its orders — waiter “by table” / joined table UI.
type SessionWithOrders struct {
	Session TableSession `json:"session"`
	Orders  []Order      `json:"orders"`
}

// LiveOrdersGroupedResponse is all open sessions with orders, grouped for waiter board.
type LiveOrdersGroupedResponse struct {
	Sessions []SessionWithOrders `json:"sessions"`
}

// KitchenDishCount aggregates one menu line for one order (item-wise count × order id).
type KitchenDishCount struct {
	OrderID    string `json:"order_id"`
	LineItemID string `json:"line_item_id"`
	Name       string `json:"name"`
	Category   string `json:"category,omitempty"`
	Quantity   int    `json:"quantity"`
	Status     string `json:"status"`
}

// KitchenUnitRow is one pending/fulfilled unit for FCFS kitchen board.
type KitchenUnitRow struct {
	OrderID     string `json:"order_id"`
	LineItemID  string `json:"line_item_id"`
	UnitIndex   int    `json:"unit_index"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	UnitStatus  string `json:"unit_status"`
	OrderedAt   int64  `json:"ordered_at"`
	OrderLabel  string `json:"order_label,omitempty"`
}

// PlatingQueueOrder is a FIFO row for plating by table/session, including order items.
type PlatingQueueOrder struct {
	OrderID       string     `json:"order_id"`
	SessionID     string     `json:"session_id"`
	TableIDs      []string   `json:"table_ids"`
	Items         []LineItem `json:"items"`
	KitchenStatus string     `json:"kitchen_status"`
	OrderedAt     int64      `json:"ordered_at"`
}

// --- Waiter flows ---

type CreateSessionAndFirstOrderRequest struct {
	TableIDs      []string              `json:"table_ids"`
	Items         []LineItem            `json:"items"`
	Channel       string                `json:"channel"`
	Pax           *int                  `json:"pax,omitempty"`
	GroupNotes    *string               `json:"group_notes,omitempty"`
	ServiceFlags  *SessionServiceFlags  `json:"service_flags,omitempty"`
	CustomerID    *string               `json:"customer_id,omitempty"`
	CustomerName  *string               `json:"customer_name,omitempty"`
	CustomerPhone *string               `json:"customer_phone,omitempty"`
	Notes         *string               `json:"notes,omitempty"`
	StaffID       *string               `json:"staff_id,omitempty"`
	OrderedAt     *int64                `json:"ordered_at,omitempty"`
}

type UpdateSessionRequest struct {
	SessionID    string               `json:"session_id"`
	Pax          *int                 `json:"pax,omitempty"`
	GroupNotes   *string              `json:"group_notes,omitempty"`
	ServiceFlags *SessionServiceFlags `json:"service_flags,omitempty"`
	TableIDs     []string             `json:"table_ids,omitempty"`
}

type AddOrderToSessionRequest struct {
	SessionID     string     `json:"session_id"`
	Items         []LineItem `json:"items"`
	Channel       string     `json:"channel"`
	SourceTableID *string    `json:"source_table_id,omitempty"`
	CustomerName  *string    `json:"customer_name,omitempty"`
	CustomerPhone *string    `json:"customer_phone,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	StaffID       *string    `json:"staff_id,omitempty"`
	OrderedAt     *int64     `json:"ordered_at,omitempty"`
}

type UpdateOrderRequestV2 struct {
	OrderID           string     `json:"order_id"`
	Items             []LineItem `json:"items,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	TotalPrice        *int64     `json:"total_price,omitempty"`
	KitchenStatus     *string    `json:"kitchen_status,omitempty"`
	MarkDone          *bool      `json:"mark_done,omitempty"`
	RemoveLineItemIDs []string   `json:"remove_line_item_ids,omitempty"`
}

type PatchLineItemStatusRequest struct {
	OrderID    string `json:"order_id"`
	LineItemID string `json:"line_item_id"`
	Status     string `json:"status"` // LineItemStatus*; legacy whole-line update
}

type PatchLineItemUnitRequest struct {
	OrderID      string `json:"order_id"`
	LineItemID   string `json:"line_item_id"`
	UnitIndex    int    `json:"unit_index"`
	Action       string `json:"action"` // fulfill, unfulfill, cancel, toggle_parcel
	CancelReason string `json:"cancel_reason,omitempty"`
}

type PatchOrderKitchenStatusRequestV2 struct {
	OrderID       string `json:"order_id"`
	KitchenStatus string `json:"kitchen_status"` // KitchenStatus*
}

type ListOrdersForSessionRequest struct {
	SessionID string `json:"session_id"`
}

type StartBillForSessionRequest struct {
	SessionID string  `json:"session_id"`
	StaffID   *string `json:"staff_id,omitempty"`
}

type UpdateBillRequestV2 struct {
	BillID              string               `json:"bill_id"`
	PaymentMethod       *string              `json:"payment_method,omitempty"`
	PaymentStatus       *string              `json:"payment_status,omitempty"`
	Discounts           []DiscountType       `json:"discounts,omitempty"`
	Taxes               []TaxType            `json:"taxes,omitempty"`
	StaffWelfareInPaise *int64               `json:"staff_welfare_in_paise,omitempty"`
	LineItemUpdates     []BillLineItemUpdate `json:"line_item_updates,omitempty"`
}

type GenerateInvoiceNumberRequest struct {
	BillID string `json:"bill_id"`
}

type BillLineItemUpdate struct {
	OrderID      string            `json:"order_id"`
	LineItemID   string            `json:"line_item_id"`
	UserOverride *LineItemOverride `json:"user_override,omitempty"`
	Removed      *bool             `json:"removed,omitempty"`
}

type CloseTableRequest struct {
	SessionID string `json:"session_id"`
	BillID    string `json:"bill_id"`
}

// --- Kitchen / plating queries ---

type KitchenItemBoardQuery struct {
	VenueID string `json:"venue_id,omitempty"`
	// Filter to non-terminal line items in app or via GSI on LineItemStatus
}

type PlatingFIFOByTableQuery struct {
	SessionID string `json:"session_id,omitempty"` // if set, FIFO for this party
	TableID   string `json:"table_id,omitempty"`   // else resolve sessions covering this table
	Limit     int32  `json:"limit,omitempty"`
}

type ListOpenKitchenOrdersQueryV2 struct {
	VenueID           string `json:"venue_id,omitempty"`
	TableID           string `json:"table_id,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
	Limit             int32  `json:"limit,omitempty"`
	ExclusiveStartKey string `json:"exclusive_start_key,omitempty"`
}
