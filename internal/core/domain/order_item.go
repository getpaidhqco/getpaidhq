package domain

import "time"

// OrderItem is a line on an Order. Cross-aggregate references are by ID only:
// Price is a separate aggregate. Use service.OrderItemDetails (or
// service.OrderDetails) when a query needs the composed view.
type OrderItem struct {
	OrgId         string
	Id            string
	OrderId       string
	ProductId     string
	VariantId     string
	PriceId       string
	// SubscriptionId is the subscription that bills this recurring line; empty
	// for one-time lines. A subscription owns the order's recurring lines that
	// share one billing cadence.
	SubscriptionId string
	Description    string
	Quantity      int
	TaxTotal      int64
	DiscountTotal int64
	Subtotal      int64
	Total         int64
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
