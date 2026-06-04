package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// OrgResponse is the HTTP response shape for an Org.
type OrgResponse struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	Country   string            `json:"country"`
	Timezone  string            `json:"timezone,omitempty"`
	Status    domain.OrgStatus  `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// NewOrgResponse maps an Org aggregate to its response DTO.
func NewOrgResponse(o domain.Org) OrgResponse {
	return OrgResponse{
		Id:        o.Id,
		Name:      o.Name,
		Country:   o.Country,
		Timezone:  o.Timezone,
		Status:    o.Status,
		Metadata:  o.Metadata,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
