package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type PaymentLink struct {
	OrgId     string      `json:"org_id"`
	Id        string      `json:"id"`
	Slug      string      `json:"slug"`
	Data      []byte      `json:"data"`
	Config    []byte      `json:"config"`
	SingleUse bool        `json:"single_use"`
	Status    string      `json:"status"`
	TokenHash pgtype.Text `json:"token_hash,omitempty"` // SHA256 hash of access token
	CreatedAt pgtype.Date `json:"created_at"`
	UpdatedAt pgtype.Date `json:"updated_at"`
	UsedAt    pgtype.Date `json:"used_at,omitempty"`
	ExpiresAt pgtype.Date `json:"expires_at,omitempty"`
}

func (p *PaymentLink) ToEntity() entities.PaymentLink {
	var data map[string]interface{}
	var config map[string]interface{}

	// Unmarshal JSON data
	if len(p.Data) > 0 {
		json.Unmarshal(p.Data, &data)
	}

	if len(p.Config) > 0 {
		json.Unmarshal(p.Config, &config)
	}

	return entities.PaymentLink{
		OrgId:     p.OrgId,
		Id:        p.Id,
		Slug:      p.Slug,
		Data:      data,
		Config:    config,
		SingleUse: p.SingleUse,
		Status:    p.Status,
		TokenHash: p.TokenHash.String,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
		UsedAt:    p.UsedAt.Time,
		ExpiresAt: p.ExpiresAt.Time,
	}
}
