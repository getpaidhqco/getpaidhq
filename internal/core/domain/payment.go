package domain

import (
	"maps"
	"time"
)

// Payment is the record of one charge attempt at the PSP. It does NOT
// represent the calculated total owed (that's the Invoice — separate
// concept; see docs/adr/0002).
type Payment struct {
	OrgId          string
	Id             string
	Psp            Gateway
	PspId          string
	Reference      string
	OrderId        string
	SubscriptionId string
	InvoiceId      string // the per-cycle Invoice this payment settles (ADR 0002)
	Status         PaymentStatus
	Recurring      bool
	Currency       string
	Amount         int64
	PspFee         int64
	PlatformFee    int64
	NetAmount      int64
	Metadata       map[string]string
	CompletedAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SetMetadata merges the existing metadata with the specified values.
func (p *Payment) SetMetadata(meta map[string]string) *Payment {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}
	maps.Copy(p.Metadata, meta)
	return p
}
