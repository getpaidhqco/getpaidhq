package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// paymentMethodRow is the postgres on-the-wire shape of a PaymentMethod.
type paymentMethodRow struct {
	OrgId          string
	Id             string
	Status         string
	Psp            string
	Name           string
	CustomerId     string
	BillingAddress jsonCol[domain.Address]
	Type           string
	Token          string
	Details        jsonCol[any]
	Metadata       jsonCol[map[string]string]
	ExpireAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const paymentMethodColumns = `org_id, id, status, psp, name, customer_id, billing_address, type, token, details, metadata, expire_at, created_at, updated_at`

func (r *paymentMethodRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Status, &r.Psp, &r.Name, &r.CustomerId, &r.BillingAddress,
		&r.Type, &r.Token, &r.Details, &r.Metadata, &r.ExpireAt, &r.CreatedAt, &r.UpdatedAt)
}

func (r paymentMethodRow) toDomain() domain.PaymentMethod {
	return domain.PaymentMethod{
		OrgId:          r.OrgId,
		Id:             r.Id,
		Status:         domain.PaymentMethodStatus(r.Status),
		Psp:            r.Psp,
		Name:           r.Name,
		CustomerId:     r.CustomerId,
		BillingAddress: r.BillingAddress.V,
		Type:           domain.PaymentMethodType(r.Type),
		Token:          domain.Secret(r.Token),
		Details:        r.Details.V,
		Metadata:       r.Metadata.V,
		ExpireAt:       r.ExpireAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func paymentMethodRowFromDomain(pm domain.PaymentMethod) paymentMethodRow {
	return paymentMethodRow{
		OrgId:          pm.OrgId,
		Id:             pm.Id,
		Status:         string(pm.Status),
		Psp:            pm.Psp,
		Name:           pm.Name,
		CustomerId:     pm.CustomerId,
		BillingAddress: newJSON(pm.BillingAddress),
		Type:           string(pm.Type),
		Token:          pm.Token.Reveal(),
		Details:        newJSON(pm.Details),
		Metadata:       newJSON(pm.Metadata),
		ExpireAt:       pm.ExpireAt,
		CreatedAt:      pm.CreatedAt,
		UpdatedAt:      pm.UpdatedAt,
	}
}
