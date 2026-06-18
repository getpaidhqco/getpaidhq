package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// paymentMethodRow is the postgres on-the-wire shape of a PaymentMethod.
// Package-internal.
type paymentMethodRow struct {
	OrgId          string                     `gorm:"column:org_id;primaryKey"`
	Id             string                     `gorm:"column:id;primaryKey"`
	Status         domain.PaymentMethodStatus `gorm:"column:status"`
	Psp            string                     `gorm:"column:psp"`
	Name           string                     `gorm:"column:name"`
	CustomerId     string                     `gorm:"column:customer_id"`
	BillingAddress domain.Address             `gorm:"column:billing_address;serializer:json"`
	Type           domain.PaymentMethodType   `gorm:"column:type"`
	Token          string                     `gorm:"column:token"`
	Details        any                        `gorm:"column:details;serializer:json"`
	Metadata       map[string]string          `gorm:"column:metadata;serializer:json"`
	ExpireAt       time.Time                  `gorm:"column:expire_at"`
	CreatedAt      time.Time                  `gorm:"column:created_at"`
	UpdatedAt      time.Time                  `gorm:"column:updated_at"`
}

func (paymentMethodRow) TableName() string { return "payment_methods" }

func (r paymentMethodRow) toDomain() domain.PaymentMethod {
	return domain.PaymentMethod{
		OrgId:          r.OrgId,
		Id:             r.Id,
		Status:         r.Status,
		Psp:            r.Psp,
		Name:           r.Name,
		CustomerId:     r.CustomerId,
		BillingAddress: r.BillingAddress,
		Type:           r.Type,
		Token:          domain.Secret(r.Token),
		Details:        r.Details,
		Metadata:       r.Metadata,
		ExpireAt:       r.ExpireAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func paymentMethodRowFromDomain(pm domain.PaymentMethod) paymentMethodRow {
	return paymentMethodRow{
		OrgId:          pm.OrgId,
		Id:             pm.Id,
		Status:         pm.Status,
		Psp:            pm.Psp,
		Name:           pm.Name,
		CustomerId:     pm.CustomerId,
		BillingAddress: pm.BillingAddress,
		Type:           pm.Type,
		Token:          pm.Token.Reveal(),
		Details:        pm.Details,
		Metadata:       pm.Metadata,
		ExpireAt:       pm.ExpireAt,
		CreatedAt:      pm.CreatedAt,
		UpdatedAt:      pm.UpdatedAt,
	}
}

func paymentMethodRowsToDomain(rows []paymentMethodRow) []domain.PaymentMethod {
	out := make([]domain.PaymentMethod, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
