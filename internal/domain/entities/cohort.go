package entities

import "time"

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
