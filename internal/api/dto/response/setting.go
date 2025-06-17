package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// Setting is the response body for a setting
type Setting struct {
	OrgId     string    `json:"org_id"`
	ParentId  string    `json:"parent_id"`
	Id        string    `json:"id"`
	Type      string    `json:"value_type"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSettingFromEntity creates a new Setting response from an entity
func NewSettingFromEntity(entity entities.Setting) Setting {
	return Setting{
		OrgId:     entity.OrgId,
		ParentId:  entity.ParentId,
		Id:        entity.Id,
		Type:      entity.Type,
		Value:     entity.Value,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

// NewSettingsFromEntities creates a new slice of Setting responses from a slice of entities
func NewSettingsFromEntities(entities []entities.Setting) []Setting {
	settings := make([]Setting, len(entities))
	for i, entity := range entities {
		settings[i] = NewSettingFromEntity(entity)
	}
	return settings
}