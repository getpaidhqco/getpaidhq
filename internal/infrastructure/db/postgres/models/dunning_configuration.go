package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
)

// DunningConfiguration represents a configuration for dunning campaigns
type DunningConfiguration struct {
	OrgId             string          `json:"org_id"`
	Id                string          `json:"id"`
	
	// Configuration hierarchy
	Name              string          `json:"name"`
	Description       pgtype.Text     `json:"description"`
	Priority          int             `json:"priority"`
	
	// Targeting rules
	AppliesTo         string          `json:"applies_to"`
	TargetRules       json.RawMessage `json:"target_rules"`
	
	// The actual configuration
	Config            json.RawMessage `json:"config"`
	
	// Status and testing
	Status            string          `json:"status"`
	IsAbTest          bool            `json:"is_ab_test"`
	AbTestPercentage  pgtype.Float8   `json:"ab_test_percentage"`
	
	// Metadata
	CreatedBy         pgtype.Text     `json:"created_by"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	UpdatedAt         pgtype.Timestamptz `json:"updated_at"`
}

// ToEntity converts the model to a domain entity
func (m *DunningConfiguration) ToEntity() dunning.DunningConfiguration {
	entity := dunning.DunningConfiguration{
		OrgId:             m.OrgId,
		Id:                m.Id,
		Name:              m.Name,
		Priority:          m.Priority,
		AppliesTo:         dunning.DunningConfigScope(m.AppliesTo),
		Status:            dunning.ConfigStatus(m.Status),
		IsAbTest:          m.IsAbTest,
	}
	
	// Handle nullable fields
	if m.Description.Valid {
		entity.Description = m.Description.String
	}
	
	if m.AbTestPercentage.Valid {
		entity.AbTestPercentage = m.AbTestPercentage.Float64
	}
	
	if m.CreatedBy.Valid {
		entity.CreatedBy = m.CreatedBy.String
	}
	
	if m.CreatedAt.Valid {
		entity.CreatedAt = m.CreatedAt.Time
	}
	
	if m.UpdatedAt.Valid {
		entity.UpdatedAt = m.UpdatedAt.Time
	}
	
	// Handle JSON fields
	if len(m.TargetRules) > 0 {
		var targetRules map[string]interface{}
		_ = json.Unmarshal(m.TargetRules, &targetRules)
		entity.TargetRules = targetRules
	}
	
	if len(m.Config) > 0 {
		var config map[string]interface{}
		_ = json.Unmarshal(m.Config, &config)
		entity.Config = config
	}
	
	return entity
}