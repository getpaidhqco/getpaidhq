package handler

import "getpaidhq/internal/core/port"

// CreateOrgRequest is the HTTP request body for POST /organizations.
type CreateOrgRequest struct {
	Name     string            `json:"name" validate:"required"`
	Country  string            `json:"country" validate:"required"`
	Timezone string            `json:"timezone" validate:"required"`
	Metadata map[string]string `json:"metadata"`
}

// ToInput maps the request to a service-layer input. Owner is supplied by
// the handler from the authenticated user context.
func (r CreateOrgRequest) ToInput(owner port.AuthUser) port.CreateOrgInput {
	return port.CreateOrgInput{
		Owner:    owner,
		Name:     r.Name,
		Country:  r.Country,
		Timezone: r.Timezone,
		Metadata: r.Metadata,
	}
}
