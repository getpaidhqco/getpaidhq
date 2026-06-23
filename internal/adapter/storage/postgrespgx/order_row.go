package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderRow is the postgres on-the-wire shape of an Order. Customer and Items
// are NOT embedded here — composition is a service-layer concern; see
// service.OrderDetails and the *Repository.FindByIds batch primitives.
type orderRow struct {
	OrgId          string
	Id             string
	CustomerId     string
	Reference      string
	Status         string
	SessionId      string
	CartId         string
	Currency       string
	Total          int64
	Metadata       jsonCol[map[string]string]
	PaymentSession jsonCol[any]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const orderColumns = `org_id, id, customer_id, reference, status, session_id, cart_id, currency, total, metadata, created_at, updated_at, payment_session`

func (r *orderRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CustomerId, &r.Reference, &r.Status, &r.SessionId,
		&r.CartId, &r.Currency, &r.Total, &r.Metadata, &r.CreatedAt, &r.UpdatedAt, &r.PaymentSession)
}

func (r orderRow) toDomain() domain.Order {
	return domain.Order{
		OrgId:          r.OrgId,
		Id:             r.Id,
		CustomerId:     r.CustomerId,
		Reference:      r.Reference,
		Status:         domain.OrderStatus(r.Status),
		SessionId:      r.SessionId,
		CartId:         r.CartId,
		Currency:       r.Currency,
		Total:          r.Total,
		Metadata:       r.Metadata.V,
		PaymentSession: r.PaymentSession.V,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func orderRowFromDomain(o domain.Order) orderRow {
	return orderRow{
		OrgId:          o.OrgId,
		Id:             o.Id,
		CustomerId:     o.CustomerId,
		Reference:      o.Reference,
		Status:         string(o.Status),
		SessionId:      o.SessionId,
		CartId:         o.CartId,
		Currency:       o.Currency,
		Total:          o.Total,
		Metadata:       newJSON(emptyIfNil(o.Metadata)),
		PaymentSession: newJSON(o.PaymentSession),
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
}
