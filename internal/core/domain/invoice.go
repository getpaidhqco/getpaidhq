package domain

import "time"

type InvoiceStatus string

const (
	InvoiceStatusDraft  InvoiceStatus = "draft"  // built, not yet settled
	InvoiceStatusOpen   InvoiceStatus = "open"   // a Payment attempt is outstanding
	InvoiceStatusPaid   InvoiceStatus = "paid"   // settled by a succeeded Payment
	InvoiceStatusUnpaid InvoiceStatus = "unpaid" // settlement failed / exhausted
	InvoiceStatusVoid   InvoiceStatus = "void"   // cancelled before settlement
)

// Invoice is the per-cycle record of what is owed for a subscription's period:
// the calculated line-item totals a Payment then attempts to settle. One per cycle.
type Invoice struct {
	OrgId          string
	Id             string
	SubscriptionId string
	CustomerId     string
	OrderId        string
	Status         InvoiceStatus
	Currency       string
	Subtotal       int64 // cents — sum of line Totals
	DiscountTotal  int64 // cents — total coupon/discount across all lines
	Total          int64 // cents — amount a Payment attempts to settle (Subtotal - DiscountTotal)
	LineItems      []InvoiceLineItem
	Cycle          int // the subscription cycle this invoice bills (CyclesProcessed at build time)
	PeriodStart    time.Time
	PeriodEnd      time.Time
	Metadata       map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
