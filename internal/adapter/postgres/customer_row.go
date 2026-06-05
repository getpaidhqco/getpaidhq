package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// customerRow is the postgres on-the-wire shape of a Customer. Package-internal.
type customerRow struct {
	OrgId                  string            `gorm:"column:org_id;primaryKey"`
	Id                     string            `gorm:"column:id;primaryKey"`
	ExternalId             string            `gorm:"column:external_id"`
	FirstName              string            `gorm:"column:first_name"`
	LastName               string            `gorm:"column:last_name"`
	Email                  string            `gorm:"column:email"`
	Phone                  string            `gorm:"column:phone"`
	DefaultPaymentMethodId string            `gorm:"column:default_payment_method_id"`
	BillingAddress         domain.Address    `gorm:"column:billing_address;serializer:json"`
	Metadata               map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt              time.Time         `gorm:"column:created_at"`
	UpdatedAt              time.Time         `gorm:"column:updated_at"`
}

func (customerRow) TableName() string { return "customers" }

func (r customerRow) toDomain() domain.Customer {
	return domain.Customer{
		OrgId:                  r.OrgId,
		Id:                     r.Id,
		ExternalId:             r.ExternalId,
		FirstName:              r.FirstName,
		LastName:               r.LastName,
		Email:                  r.Email,
		Phone:                  r.Phone,
		DefaultPaymentMethodId: r.DefaultPaymentMethodId,
		BillingAddress:         r.BillingAddress,
		Metadata:               r.Metadata,
		CreatedAt:              r.CreatedAt,
		UpdatedAt:              r.UpdatedAt,
	}
}

func customerRowFromDomain(c domain.Customer) customerRow {
	return customerRow{
		OrgId:                  c.OrgId,
		Id:                     c.Id,
		ExternalId:             c.ExternalId,
		FirstName:              c.FirstName,
		LastName:               c.LastName,
		Email:                  c.Email,
		Phone:                  c.Phone,
		DefaultPaymentMethodId: c.DefaultPaymentMethodId,
		BillingAddress:         c.BillingAddress,
		Metadata:               c.Metadata,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
	}
}

// customerCohortRow is the postgres shape of a CustomerCohort join row.
type customerCohortRow struct {
	OrgId       string    `gorm:"column:org_id;primaryKey"`
	CustomerId  string    `gorm:"column:customer_id;primaryKey"`
	CohortId    string    `gorm:"column:cohort_id;primaryKey"`
	CohortValue string    `gorm:"column:cohort_value"`
	JoinedAt    time.Time `gorm:"column:joined_at"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (customerCohortRow) TableName() string { return "customer_cohorts" }

func customerCohortRowFromDomain(cc domain.CustomerCohort) customerCohortRow {
	return customerCohortRow{
		OrgId:       cc.OrgId,
		CustomerId:  cc.CustomerId,
		CohortId:    cc.CohortId,
		CohortValue: cc.CohortValue,
		JoinedAt:    cc.JoinedAt,
		CreatedAt:   cc.CreatedAt,
		UpdatedAt:   cc.UpdatedAt,
	}
}

// cohortRow is the postgres shape of a Cohort.
type cohortRow struct {
	OrgId     string            `gorm:"column:org_id;primaryKey"`
	Id        string            `gorm:"column:id;primaryKey"`
	Name      string            `gorm:"column:name"`
	Type      domain.CohortType `gorm:"column:type"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt time.Time         `gorm:"column:created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at"`
}

func (cohortRow) TableName() string { return "cohorts" }

func (r cohortRow) toDomain() domain.Cohort {
	return domain.Cohort{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Name:      r.Name,
		Type:      r.Type,
		Metadata:  r.Metadata,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func cohortRowFromDomain(c domain.Cohort) cohortRow {
	return cohortRow{
		OrgId:     c.OrgId,
		Id:        c.Id,
		Name:      c.Name,
		Type:      c.Type,
		Metadata:  c.Metadata,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
