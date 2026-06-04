package domain

import "time"

// OrderItem is a line on an Order. The Price field is populated by the
// repository when a Preload-equivalent is used; for code paths that don't
// hydrate it, only PriceId is reliable.
type OrderItem struct {
	OrgId         string
	Id            string
	OrderId       string
	ProductId     string
	VariantId     string
	PriceId       string
	Price         Price
	Description   string
	Quantity      int
	TaxTotal      int64
	DiscountTotal int64
	Subtotal      int64
	Total         int64
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
