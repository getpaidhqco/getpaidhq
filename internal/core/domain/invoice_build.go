package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/lib"
)

// BaseLineFromPrice builds the fixed base line for a subscription's period from its
// linked Price. Fixed scheme today: Total = round(UnitPrice × quantity / UnitCount)
// in cents. Quantity is whole for a base line; UnitAmount is the effective per-unit
// cents (fractional when UnitCount > 1).
// Exported so Spec B's usage path can sit alongside it on the same invoice.
func BaseLineFromPrice(orgId, invoiceId string, p Price, quantity decimal.Decimal) InvoiceLineItem {
	total, unit := priceFixed(p, quantity)
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
	inv := NewInvoice(sub, periodStart, periodEnd)
	inv.AddLine(BaseLineFromPrice(sub.OrgId, inv.Id, p, quantity))
	return inv
}

// NewInvoice returns an empty draft invoice skeleton for a subscription's period.
// Callers append base/usage lines via AddLine.
func NewInvoice(sub Subscription, periodStart, periodEnd time.Time) Invoice {
	now := time.Now().UTC()
	return Invoice{
		OrgId:          sub.OrgId,
		Id:             lib.GenerateId("inv"),
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
}

// UsageLineFromPrice builds a usage line for a metered Price from aggregated units.
func UsageLineFromPrice(orgId, invoiceId string, p Price, units decimal.Decimal) InvoiceLineItem {
	amt, unit := PriceUsage(p, units)
	now := time.Now().UTC()
	return InvoiceLineItem{
		OrgId:       orgId,
		Id:          lib.GenerateId("ili"),
		InvoiceId:   invoiceId,
		PriceId:     p.Id,
		Kind:        InvoiceLineKindUsage,
		Description: p.Label,
		Quantity:    units,
		UnitAmount:  unit,
		Total:       amt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UsageLineFromPriceGrouped builds one usage line for a single group segment of a
// metered Price: the same rate as the ungrouped line, but only this segment's units,
// with the group key/value recorded in the line's Metadata and appended to the
// description. Group splits a priced line into one line per discovered value at the
// same rate (usage-filters-and-groups.md).
func UsageLineFromPriceGrouped(orgId, invoiceId string, p Price, groupKey, groupValue string, units decimal.Decimal) InvoiceLineItem {
	line := UsageLineFromPrice(orgId, invoiceId, p, units)
	line.Metadata = map[string]string{groupKey: groupValue}
	line.Description = p.Label + " (" + groupKey + "=" + groupValue + ")"
	return line
}

// AddLine appends a line item and recomputes the invoice totals.
func (inv *Invoice) AddLine(l InvoiceLineItem) {
	inv.LineItems = append(inv.LineItems, l)
	inv.recalculate()
}

func (inv *Invoice) recalculate() {
	var subtotal, discount int64
	for _, l := range inv.LineItems {
		subtotal += l.Total
		discount += l.DiscountTotal
	}
	inv.Subtotal = subtotal
	inv.DiscountTotal = discount
	inv.Total = subtotal - discount
}
