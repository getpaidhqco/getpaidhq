package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"time"
)

// Org represents an organization in the database
type Org struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	Country   string            `json:"country"`
	Timezone  pgtype.Text       `json:"timezone"`
	Status    string            `json:"status"`
	Metadata  json.RawMessage   `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ToEntity converts the model to an entity
func (o Org) ToEntity() entities.Org {
	org := entities.Org{
		Id:        o.Id,
		Name:      o.Name,
		Country:   o.Country,
		Status:    entities.OrgStatus(o.Status),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}

	if o.Timezone.Valid {
		org.Timezone = o.Timezone.String
	}

	if len(o.Metadata) > 0 {
		var metadata map[string]string
		if err := json.Unmarshal(o.Metadata, &metadata); err == nil {
			org.Metadata = metadata
		}
	}

	return org
}