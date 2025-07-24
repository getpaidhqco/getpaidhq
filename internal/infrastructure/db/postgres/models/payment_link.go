package models

import (
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
	CreatedAt pgtype.Date `json:"created_at"`
	UpdatedAt pgtype.Date `json:"updated_at"`
	UsedAt    pgtype.Date `json:"used_at,omitempty"`
	ExpiresAt pgtype.Date `json:"expires_at,omitempty"`
}

func (p *PaymentLink) ToEntity() entities.PaymentLink {
	return entities.PaymentLink{
		OrgId:     p.OrgId,
		Id:        p.Id,
		Slug:      p.Slug,
		Data:      p.Data,
		Config:    p.Config,
		SingleUse: p.SingleUse,
		Status:    p.Status,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
		UsedAt:    p.UsedAt.Time,
		ExpiresAt: p.ExpiresAt.Time,
	}
}
