package entities

import "time"

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
