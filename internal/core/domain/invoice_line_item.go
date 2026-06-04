package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvoiceLineItemKind string

const (
	InvoiceLineKindBase  InvoiceLineItemKind = "base"  // fixed/recurring base charge
	InvoiceLineKindUsage InvoiceLineItemKind = "usage" // metered usage
)

// InvoiceLineItem is one line on an Invoice. Quantity is decimal (whole for base
// lines, fractional for usage lines); UnitAmount is decimal cents (sub-cent rates are
// possible for usage); Total is int64 cents, the actually-charged amount, rounded once.
type InvoiceLineItem struct {
	OrgId       string
	Id          string
	InvoiceId   string
	PriceId     string
	Kind        InvoiceLineItemKind
	Description string
	Quantity    decimal.Decimal
	UnitAmount  decimal.Decimal // cents (may be fractional)
	Total       int64           // cents
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
