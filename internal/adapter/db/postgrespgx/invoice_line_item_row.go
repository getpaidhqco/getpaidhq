package postgrespgx

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// invoiceLineItemRow is the postgres on-the-wire shape of an InvoiceLineItem.
// quantity / unit_amount are numeric columns scanned straight into
// decimal.Decimal. metadata carries no emptyIfNil on the write path, so a nil
// map serialises to JSON null.
type invoiceLineItemRow struct {
	OrgId         string
	Id            string
	InvoiceId     string
	PriceId       string
	Kind          string
	Description   string
	Quantity      decimal.Decimal
	UnitAmount    decimal.Decimal
	Total         int64
	DiscountTotal int64
	Metadata      jsonCol[map[string]string]
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

const invoiceLineItemColumns = `org_id, id, invoice_id, price_id, kind, description, quantity, unit_amount, total, discount_total, metadata, created_at, updated_at`

func (r *invoiceLineItemRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.InvoiceId, &r.PriceId, &r.Kind, &r.Description,
		&r.Quantity, &r.UnitAmount, &r.Total, &r.DiscountTotal, &r.Metadata,
		&r.CreatedAt, &r.UpdatedAt)
}

func (r invoiceLineItemRow) toDomain() domain.InvoiceLineItem {
	return domain.InvoiceLineItem{
		OrgId:         r.OrgId,
		Id:            r.Id,
		InvoiceId:     r.InvoiceId,
		PriceId:       r.PriceId,
		Kind:          domain.InvoiceLineItemKind(r.Kind),
		Description:   r.Description,
		Quantity:      r.Quantity,
		UnitAmount:    r.UnitAmount,
		Total:         r.Total,
		DiscountTotal: r.DiscountTotal,
		Metadata:      r.Metadata.V,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func invoiceLineItemRowFromDomain(l domain.InvoiceLineItem) invoiceLineItemRow {
	return invoiceLineItemRow{
		OrgId:         l.OrgId,
		Id:            l.Id,
		InvoiceId:     l.InvoiceId,
		PriceId:       l.PriceId,
		Kind:          string(l.Kind),
		Description:   l.Description,
		Quantity:      l.Quantity,
		UnitAmount:    l.UnitAmount,
		Total:         l.Total,
		DiscountTotal: l.DiscountTotal,
		Metadata:      newJSON(l.Metadata),
		CreatedAt:     l.CreatedAt,
		UpdatedAt:     l.UpdatedAt,
	}
}

func invoiceLineItemRowsToDomain(rows []invoiceLineItemRow) []domain.InvoiceLineItem {
	out := make([]domain.InvoiceLineItem, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out
}
