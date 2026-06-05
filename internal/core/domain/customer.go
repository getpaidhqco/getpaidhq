package domain

import "time"

// Customer is a person or organization billed by a tenant.
type Customer struct {
	OrgId                  string
	Id                     string
	ExternalId             string // the merchant's own id for this customer; matches usage events' external_customer_id
	FirstName              string
	LastName               string
	Email                  string
	Phone                  string
	DefaultPaymentMethodId string
	BillingAddress         Address
	Metadata               map[string]string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// CustomerCohort joins a Customer to a Cohort.
type CustomerCohort struct {
	OrgId       string
	CustomerId  string
	CohortId    string
	CohortValue string
	// Metadata is in-memory only — not persisted on the join table.
	Metadata  map[string]string
	JoinedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CohortType string

const (
	CohortTypeSignupDate CohortType = "signup_date"
	CohortTypeProduct    CohortType = "product"
)

// Cohort is a named segment of Customers within an Org.
type Cohort struct {
	OrgId     string
	Id        string
	Name      string
	Type      CohortType
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}
