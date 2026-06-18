package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// customerRow is the postgres on-the-wire shape of a Customer. external_id and
// default_payment_method_id are nullable (NULL, never "") — external_id is the
// merchant id deduped by a partial unique index, default_payment_method_id is a
// nullable FK to payment_methods.
type customerRow struct {
	OrgId                  string
	Id                     string
	ExternalId             *string
	FirstName              string
	LastName               string
	Email                  string
	Phone                  string
	DefaultPaymentMethodId *string
	BillingAddress         jsonCol[domain.Address]
	Metadata               jsonCol[map[string]string]
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

const customerColumns = `org_id, id, external_id, first_name, last_name, email, phone, default_payment_method_id, billing_address, metadata, created_at, updated_at`

func (r *customerRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.ExternalId, &r.FirstName, &r.LastName, &r.Email,
		&r.Phone, &r.DefaultPaymentMethodId, &r.BillingAddress, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r customerRow) toDomain() domain.Customer {
	return domain.Customer{
		OrgId:                  r.OrgId,
		Id:                     r.Id,
		ExternalId:             strOrEmpty(r.ExternalId),
		FirstName:              r.FirstName,
		LastName:               r.LastName,
		Email:                  r.Email,
		Phone:                  r.Phone,
		DefaultPaymentMethodId: strOrEmpty(r.DefaultPaymentMethodId),
		BillingAddress:         r.BillingAddress.V,
		Metadata:               r.Metadata.V,
		CreatedAt:              r.CreatedAt,
		UpdatedAt:              r.UpdatedAt,
	}
}

func customerRowFromDomain(c domain.Customer) customerRow {
	return customerRow{
		OrgId:                  c.OrgId,
		Id:                     c.Id,
		ExternalId:             nilIfEmpty(c.ExternalId),
		FirstName:              c.FirstName,
		LastName:               c.LastName,
		Email:                  c.Email,
		Phone:                  c.Phone,
		DefaultPaymentMethodId: nilIfEmpty(c.DefaultPaymentMethodId),
		BillingAddress:         newJSON(c.BillingAddress),
		Metadata:               newJSON(c.Metadata),
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
	}
}

// cohortRow is the postgres shape of a Cohort.
type cohortRow struct {
	OrgId     string
	Id        string
	Name      string
	Type      string
	Metadata  jsonCol[map[string]string]
	CreatedAt time.Time
	UpdatedAt time.Time
}

const cohortColumns = `org_id, id, name, type, metadata, created_at, updated_at`

func (r *cohortRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Name, &r.Type, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r cohortRow) toDomain() domain.Cohort {
	return domain.Cohort{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Name:      r.Name,
		Type:      domain.CohortType(r.Type),
		Metadata:  r.Metadata.V,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func cohortRowFromDomain(c domain.Cohort) cohortRow {
	return cohortRow{
		OrgId:     c.OrgId,
		Id:        c.Id,
		Name:      c.Name,
		Type:      string(c.Type),
		Metadata:  newJSON(c.Metadata),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
