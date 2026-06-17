package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// invoiceRow is the postgres on-the-wire shape of an Invoice. LineItems are populated
// via Preload at the row level.
type invoiceRow struct {
	OrgId          string               `gorm:"column:org_id;primaryKey"`
	Id             string               `gorm:"column:id;primaryKey"`
	SubscriptionId string               `gorm:"column:subscription_id"`
	CustomerId     string               `gorm:"column:customer_id"`
	OrderId        string               `gorm:"column:order_id"`
	Status         domain.InvoiceStatus `gorm:"column:status"`
	Currency       string               `gorm:"column:currency"`
	Subtotal       int64                `gorm:"column:subtotal"`
	DiscountTotal  int64                `gorm:"column:discount_total"`
	Total          int64                `gorm:"column:total"`
	LineItems      []invoiceLineItemRow `gorm:"foreignKey:InvoiceId,OrgId;references:Id,OrgId"`
	Cycle          int                  `gorm:"column:cycle"`
	PeriodStart    time.Time            `gorm:"column:period_start"`
	PeriodEnd      time.Time            `gorm:"column:period_end"`
	Metadata       map[string]string    `gorm:"column:metadata;serializer:json"`
	CreatedAt      time.Time            `gorm:"column:created_at"`
	UpdatedAt      time.Time            `gorm:"column:updated_at"`
}

func (invoiceRow) TableName() string { return "invoices" }

func (r invoiceRow) toDomain() domain.Invoice {
	return domain.Invoice{
		OrgId:          r.OrgId,
		Id:             r.Id,
		SubscriptionId: r.SubscriptionId,
		CustomerId:     r.CustomerId,
		OrderId:        r.OrderId,
		Status:         r.Status,
		Currency:       r.Currency,
		Subtotal:       r.Subtotal,
		DiscountTotal:  r.DiscountTotal,
		Total:          r.Total,
		LineItems:      invoiceLineItemRowsToDomain(r.LineItems),
		Cycle:          r.Cycle,
		PeriodStart:    r.PeriodStart,
		PeriodEnd:      r.PeriodEnd,
		Metadata:       r.Metadata,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func invoiceRowsToDomain(rows []invoiceRow) []domain.Invoice {
	out := make([]domain.Invoice, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out
}

func invoiceRowFromDomain(inv domain.Invoice) invoiceRow {
	return invoiceRow{
		OrgId:          inv.OrgId,
		Id:             inv.Id,
		SubscriptionId: inv.SubscriptionId,
		CustomerId:     inv.CustomerId,
		OrderId:        inv.OrderId,
		Status:         inv.Status,
		Currency:       inv.Currency,
		Subtotal:       inv.Subtotal,
		DiscountTotal:  inv.DiscountTotal,
		Total:          inv.Total,
		LineItems:      invoiceLineItemRowsFromDomain(inv.LineItems),
		Cycle:          inv.Cycle,
		PeriodStart:    inv.PeriodStart,
		PeriodEnd:      inv.PeriodEnd,
		Metadata:       inv.Metadata,
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
