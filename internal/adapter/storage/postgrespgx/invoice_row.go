package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// invoiceRow is the postgres on-the-wire shape of an Invoice. LineItems are
// hydrated separately (a second query keyed by invoice id), mirroring the gorm
// adapter's Preload. period_start / period_end are nullable timestamp columns,
// so they are carried as *time.Time and mapped to/from the domain's zero-time
// "unset" sentinel via nullTime / timeOrZero. metadata is a NOT-NULL-intent
// jsonb column written as `{}` (emptyIfNil before the jsonCol wrap on the
// Create / Update paths).
type invoiceRow struct {
	OrgId          string
	Id             string
	SubscriptionId string
	CustomerId     string
	OrderId        string
	Status         string
	Currency       string
	Subtotal       int64
	DiscountTotal  int64
	Total          int64
	Cycle          int
	PeriodStart    *time.Time
	PeriodEnd      *time.Time
	Metadata       jsonCol[map[string]string]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const invoiceColumns = `org_id, id, subscription_id, customer_id, order_id, status, currency, subtotal, discount_total, total, cycle, period_start, period_end, metadata, created_at, updated_at`

func (r *invoiceRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.SubscriptionId, &r.CustomerId, &r.OrderId,
		&r.Status, &r.Currency, &r.Subtotal, &r.DiscountTotal, &r.Total, &r.Cycle,
		&r.PeriodStart, &r.PeriodEnd, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

// toDomain builds the domain.Invoice from the row. LineItems are filled in by
// the repo after a separate query, so they are left nil here.
func (r invoiceRow) toDomain() domain.Invoice {
	return domain.Invoice{
		OrgId:          r.OrgId,
		Id:             r.Id,
		SubscriptionId: r.SubscriptionId,
		CustomerId:     r.CustomerId,
		OrderId:        r.OrderId,
		Status:         domain.InvoiceStatus(r.Status),
		Currency:       r.Currency,
		Subtotal:       r.Subtotal,
		DiscountTotal:  r.DiscountTotal,
		Total:          r.Total,
		Cycle:          r.Cycle,
		PeriodStart:    timeOrZero(r.PeriodStart),
		PeriodEnd:      timeOrZero(r.PeriodEnd),
		Metadata:       r.Metadata.V,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func invoiceRowFromDomain(inv domain.Invoice) invoiceRow {
	return invoiceRow{
		OrgId:          inv.OrgId,
		Id:             inv.Id,
		SubscriptionId: inv.SubscriptionId,
		CustomerId:     inv.CustomerId,
		OrderId:        inv.OrderId,
		Status:         string(inv.Status),
		Currency:       inv.Currency,
		Subtotal:       inv.Subtotal,
		DiscountTotal:  inv.DiscountTotal,
		Total:          inv.Total,
		Cycle:          inv.Cycle,
		PeriodStart:    nullTime(inv.PeriodStart),
		PeriodEnd:      nullTime(inv.PeriodEnd),
		Metadata:       newJSON(emptyIfNil(inv.Metadata)),
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
