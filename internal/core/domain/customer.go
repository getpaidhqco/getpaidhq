package domain

import "time"

type Customer struct {
	OrgId                  string            `json:"org_id"`
	Id                     string            `json:"id"`
	FirstName              string            `json:"first_name,omitempty"`
	LastName               string            `json:"last_name,omitempty"`
	Email                  string            `json:"email,omitempty"`
	Phone                  string            `json:"phone,omitempty"`
	DefaultPaymentMethodId string            `json:"default_payment_method_id,omitempty"`
	BillingAddress         Address           `json:"billing_address,omitempty"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

type CustomerCohort struct {
	OrgId       string            `json:"org_id"`
	CustomerId  string            `json:"customer_id"`
	CohortId    string            `json:"cohort_id"`
	CohortValue string            `json:"cohort_value"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	JoinedAt    time.Time         `json:"joined_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CohortType string

const (
	CohortTypeSignupDate CohortType = "signup_date"
	CohortTypeProduct    CohortType = "product"
)

type Cohort struct {
	OrgId     string            `json:"org_id"`
	Id        string            `json:"id"`
	Name      string            `json:"name,omitempty"`
	Type      CohortType        `json:"type,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
