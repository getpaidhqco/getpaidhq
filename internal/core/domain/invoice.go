package domain

import (
	"errors"
	"time"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"         // built, not yet charged
	InvoiceStatusOpen          InvoiceStatus = "open"          // finalized, a charge is outstanding
	InvoiceStatusPaid          InvoiceStatus = "paid"          // settled by a succeeded Payment
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible" // collection given up (terminal)
	InvoiceStatusVoid          InvoiceStatus = "void"          // cancelled, never collected (terminal)
)

// ErrInvalidInvoiceTransition is returned when a status change is not allowed
// from the invoice's current state.
var ErrInvalidInvoiceTransition = errors.New("invalid invoice status transition")

// Invoice is the per-cycle record of what is owed for a subscription's period:
// the calculated line-item totals a Payment then attempts to settle. One per cycle.
type Invoice struct {
	OrgId          string
	Id             string
	Number         int64
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

// MarkOpen finalizes a draft invoice for collection. Idempotent from open.
func (inv *Invoice) MarkOpen() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusOpen
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// MarkPaid settles an outstanding invoice.
func (inv *Invoice) MarkPaid() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusPaid
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// MarkUncollectible writes off an outstanding invoice (collection abandoned).
func (inv *Invoice) MarkUncollectible() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusUncollectible
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// Void cancels an invoice that should never be collected.
func (inv *Invoice) Void() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusVoid
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}
