package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/lib"
)

// BaseLineFromPrice builds the fixed base line for a subscription's period from its
// linked Price. Fixed scheme today: Total = round(UnitPrice × quantity) in cents.
// Quantity is whole for a base line; UnitAmount is the price's per-unit cents.
// Exported so Spec B's usage path can sit alongside it on the same invoice.
func BaseLineFromPrice(orgId, invoiceId string, p Price, quantity decimal.Decimal) InvoiceLineItem {
	unit := decimal.NewFromInt(p.UnitPrice) // cents per unit
	total := unit.Mul(quantity).Round(0).IntPart()
	now := time.Now().UTC()
	return InvoiceLineItem{
		OrgId:       orgId,
		Id:          lib.GenerateId("ili"),
		InvoiceId:   invoiceId,
		PriceId:     p.Id,
		Kind:        InvoiceLineKindBase,
		Description: p.Label,
		Quantity:    quantity,
		UnitAmount:  unit,
		Total:       total,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// BuildInvoiceForPeriod assembles a draft Invoice for a subscription's period with a
// single base line from the linked Price, and totals it. Spec B appends usage lines
// before persistence via AddLine.
func BuildInvoiceForPeriod(sub Subscription, p Price, quantity decimal.Decimal, periodStart, periodEnd time.Time) Invoice {
	id := lib.GenerateId("inv")
	now := time.Now().UTC()
	inv := Invoice{
		OrgId:          sub.OrgId,
		Id:             id,
		SubscriptionId: sub.Id,
		CustomerId:     sub.CustomerId,
		OrderId:        sub.OrderId,
		Status:         InvoiceStatusDraft,
		Currency:       sub.Currency,
		Cycle:          sub.CyclesProcessed,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	inv.AddLine(BaseLineFromPrice(sub.OrgId, id, p, quantity))
	return inv
}

// AddLine appends a line item and recomputes the invoice totals.
func (inv *Invoice) AddLine(l InvoiceLineItem) {
	inv.LineItems = append(inv.LineItems, l)
	inv.recalculate()
}

func (inv *Invoice) recalculate() {
	var subtotal int64
	for _, l := range inv.LineItems {
		subtotal += l.Total
	}
	inv.Subtotal = subtotal
	inv.Total = subtotal
}
