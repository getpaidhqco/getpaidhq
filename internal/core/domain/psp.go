package domain

import "time"

// PspConfig represents a payment service provider configuration for an organization.
// Named PspConfig (not Gateway) to avoid collision with the Gateway string type.
type PspConfig struct {
	OrgId     string    `json:"org_id"`
	Id        string    `json:"id"`
	PspId     Gateway   `json:"psp_id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
