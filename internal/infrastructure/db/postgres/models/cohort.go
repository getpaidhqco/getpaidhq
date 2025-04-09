package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Cohort struct {
	OrgId     string            `json:"org_id"`
	Id        string            `json:"id"`
	Name      pgtype.Text       `json:"name"`
	Type      pgtype.Text       `json:"type"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt pgtype.Date       `json:"created_at"`
	UpdatedAt pgtype.Date       `json:"updated_at"`
}

func (c *Cohort) ToEntity() entities.Cohort {
	return entities.Cohort{
		OrgId:     c.OrgId,
		Id:        c.Id,
		Name:      c.Name.String,
		Type:      entities.CohortType(c.Type.String),
		Metadata:  c.Metadata,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}
