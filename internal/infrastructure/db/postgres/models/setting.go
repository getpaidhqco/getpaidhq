package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Setting struct {
	OrgId     string          `json:"org_id"`
	ParentId  string          `json:"parent_id"`
	Id        string          `json:"id"`
	ValueType string          `json:"value_type"`
	Value     json.RawMessage `json:"value"`
	CreatedAt pgtype.Date     `json:"created_at"`
	UpdatedAt pgtype.Date     `json:"updated_at"`
}

func (s *Setting) ToEntity() entities.Setting {
	return entities.Setting{
		OrgId:    s.OrgId,
		ParentId: s.ParentId,
		Id:       s.Id,
		Type:     s.ValueType,
		Value:    string(s.Value),
	}
}