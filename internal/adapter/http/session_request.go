package handler

import "getpaidhq/internal/core/port"

// CreateSessionRequest is the HTTP request body for POST /sessions.
type CreateSessionRequest struct {
	Currency string            `json:"currency" validate:"required"`
	Country  string            `json:"country" validate:"required"`
	Metadata map[string]string `json:"metadata"`
}

// ToInput maps the request to a service-layer input. The orgId comes from
// the authenticated user context at the handler.
func (r CreateSessionRequest) ToInput(orgId string) port.CreateSessionInput {
	return port.CreateSessionInput{
		OrgId:    orgId,
		Currency: r.Currency,
		Country:  r.Country,
		Metadata: r.Metadata,
	}
}
