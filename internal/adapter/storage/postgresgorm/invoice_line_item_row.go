package postgresgorm

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// invoiceLineItemRow is the postgres on-the-wire shape of an InvoiceLineItem.
type invoiceLineItemRow struct {
	OrgId         string                     `gorm:"column:org_id;primaryKey"`
	Id            string                     `gorm:"column:id;primaryKey"`
	InvoiceId     string                     `gorm:"column:invoice_id"`
	PriceId       string                     `gorm:"column:price_id"`
	Kind          domain.InvoiceLineItemKind `gorm:"column:kind"`
	Description   string                     `gorm:"column:description"`
	Quantity      decimal.Decimal            `gorm:"column:quantity;type:numeric"`
	UnitAmount    decimal.Decimal            `gorm:"column:unit_amount;type:numeric"`
	Total         int64                      `gorm:"column:total"`
	DiscountTotal int64                      `gorm:"column:discount_total"`
	Metadata      map[string]string          `gorm:"column:metadata;serializer:json"`
	CreatedAt     time.Time                  `gorm:"column:created_at"`
	UpdatedAt     time.Time                  `gorm:"column:updated_at"`
}

func (invoiceLineItemRow) TableName() string { return "invoice_line_items" }

func (r invoiceLineItemRow) toDomain() domain.InvoiceLineItem {
	return domain.InvoiceLineItem{
		OrgId:         r.OrgId,
		Id:            r.Id,
		InvoiceId:     r.InvoiceId,
		PriceId:       r.PriceId,
		Kind:          r.Kind,
		Description:   r.Description,
		Quantity:      r.Quantity,
		UnitAmount:    r.UnitAmount,
		Total:         r.Total,
		DiscountTotal: r.DiscountTotal,
		Metadata:      r.Metadata,
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
		Kind:          l.Kind,
		Description:   l.Description,
		Quantity:      l.Quantity,
		UnitAmount:    l.UnitAmount,
		Total:         l.Total,
		DiscountTotal: l.DiscountTotal,
		Metadata:      l.Metadata,
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

func invoiceLineItemRowsFromDomain(ls []domain.InvoiceLineItem) []invoiceLineItemRow {
	out := make([]invoiceLineItemRow, len(ls))
	for i, l := range ls {
		out[i] = invoiceLineItemRowFromDomain(l)
	}
	return out
}
