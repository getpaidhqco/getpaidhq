package domain

import "time"

type Customer struct {
	OrgId                  string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id                     string            `gorm:"column:id;primaryKey" json:"id"`
	FirstName              string            `gorm:"column:first_name" json:"first_name,omitempty"`
	LastName               string            `gorm:"column:last_name" json:"last_name,omitempty"`
	Email                  string            `gorm:"column:email" json:"email,omitempty"`
	Phone                  string            `gorm:"column:phone" json:"phone,omitempty"`
	DefaultPaymentMethodId string            `gorm:"column:default_payment_method_id" json:"default_payment_method_id,omitempty"`
	BillingAddress         Address           `gorm:"column:billing_address;serializer:json" json:"billing_address,omitempty"`
	Metadata               map[string]string `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	CreatedAt              time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt              time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Customer) TableName() string { return "customers" }

type CustomerCohort struct {
	OrgId       string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	CustomerId  string            `gorm:"column:customer_id;primaryKey" json:"customer_id"`
	CohortId    string            `gorm:"column:cohort_id;primaryKey" json:"cohort_id"`
	CohortValue string            `gorm:"column:cohort_value" json:"cohort_value"`
	Metadata    map[string]string `gorm:"-" json:"metadata,omitempty"`
	JoinedAt    time.Time         `gorm:"column:joined_at" json:"joined_at"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (CustomerCohort) TableName() string { return "customer_cohorts" }

type CohortType string

const (
	CohortTypeSignupDate CohortType = "signup_date"
	CohortTypeProduct    CohortType = "product"
)

type Cohort struct {
	OrgId     string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id        string            `gorm:"column:id;primaryKey" json:"id"`
	Name      string            `gorm:"column:name" json:"name,omitempty"`
	Type      CohortType        `gorm:"column:type" json:"type,omitempty"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	CreatedAt time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Cohort) TableName() string { return "cohorts" }
