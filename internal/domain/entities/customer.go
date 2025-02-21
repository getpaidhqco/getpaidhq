package entities

import "time"

type Customer struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Email          string            `json:"email"`
	Phone          string            `json:"phone"`
	BillingAddress Address           `json:"billing_address"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
