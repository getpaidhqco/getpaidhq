package domain

import "time"

// PspConfig represents a payment service provider configuration for an organization.
// Named PspConfig (not Gateway) to avoid collision with the Gateway string type.
type PspConfig struct {
	OrgId     string
	Id        string
	PspId     Gateway
	Name      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
