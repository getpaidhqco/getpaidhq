package handler

import "getpaidhq/internal/core/domain"

// CreateSessionResponse is the HTTP response body for POST /sessions.
type CreateSessionResponse struct {
	Id     string `json:"id"`
	CartId string `json:"cart_id"`
}

// NewCreateSessionResponse maps a session aggregate to its create-response
// shape.
func NewCreateSessionResponse(s domain.Session) CreateSessionResponse {
	return CreateSessionResponse{
		Id:     s.Id,
		CartId: s.CartId,
	}
}
