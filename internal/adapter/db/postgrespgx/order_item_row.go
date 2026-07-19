package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderItemRow is the postgres on-the-wire shape of an OrderItem. Price is
// NOT embedded here — composition is a service-layer concern; see
// service.OrderItemDetails and PriceRepository.FindByIds for the read-model
// path.
type orderItemRow struct {
	OrgId     string
	Id        string
	OrderId   string
	ProductId string
	// variant_id is a nullable FK to variants; NULL when absent (never "") —
	// writing "" would violate the FK.
	VariantId *string
	PriceId   string
	// subscription_id is a nullable FK to subscriptions; NULL when absent
	// (never "") — items have no subscription until the order completes, and
	// writing "" would violate the FK.
	SubscriptionId *string
	Description    string
	Quantity       int
	TaxTotal       int64
	DiscountTotal  int64
	Subtotal       int64
	Total          int64
	Metadata       jsonCol[map[string]string]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const orderItemColumns = `org_id, id, order_id, product_id, variant_id, price_id, subscription_id, description, quantity, tax_total, discount_total, sub_total, total, metadata, created_at, updated_at`

func (r *orderItemRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.OrderId, &r.ProductId, &r.VariantId, &r.PriceId,
		&r.SubscriptionId, &r.Description, &r.Quantity, &r.TaxTotal, &r.DiscountTotal,
		&r.Subtotal, &r.Total, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r orderItemRow) toDomain() domain.OrderItem {
	return domain.OrderItem{
		OrgId:          r.OrgId,
		Id:             r.Id,
		OrderId:        r.OrderId,
		ProductId:      r.ProductId,
		VariantId:      strOrEmpty(r.VariantId),
		PriceId:        r.PriceId,
		SubscriptionId: strOrEmpty(r.SubscriptionId),
		Description:    r.Description,
		Quantity:       r.Quantity,
		TaxTotal:       r.TaxTotal,
		DiscountTotal:  r.DiscountTotal,
		Subtotal:       r.Subtotal,
		Total:          r.Total,
		Metadata:       r.Metadata.V,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func orderItemRowFromDomain(i domain.OrderItem) orderItemRow {
	return orderItemRow{
		OrgId:          i.OrgId,
		Id:             i.Id,
		OrderId:        i.OrderId,
		ProductId:      i.ProductId,
		VariantId:      nilIfEmpty(i.VariantId),
		PriceId:        i.PriceId,
		SubscriptionId: nilIfEmpty(i.SubscriptionId),
		Description:    i.Description,
		Quantity:       i.Quantity,
		TaxTotal:       i.TaxTotal,
		DiscountTotal:  i.DiscountTotal,
		Subtotal:       i.Subtotal,
		Total:          i.Total,
		Metadata:       newJSON(emptyIfNil(i.Metadata)),
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}
}
