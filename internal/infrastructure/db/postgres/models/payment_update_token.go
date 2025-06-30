package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
)

// PaymentUpdateToken represents a secure token for updating payment methods
type PaymentUpdateToken struct {
	OrgId             string          `json:"org_id"`
	TokenId           string          `json:"token_id"`
	
	// Relationships
	SubscriptionId    string          `json:"subscription_id"`
	CustomerId        string          `json:"customer_id"`
	DunningCampaignId pgtype.Text     `json:"dunning_campaign_id"`
	
	// Token data
	TokenData         json.RawMessage `json:"token_data"`
	Signature         string          `json:"signature"`
	
	// Security & usage
	ExpiresAt         pgtype.Timestamptz `json:"expires_at"`
	MaxUses           int             `json:"max_uses"`
	UsedCount         int             `json:"used_count"`
	Status            string          `json:"status"`
	
	// Allowed actions
	AllowedActions    json.RawMessage `json:"allowed_actions"`
	
	// Admin generation tracking
	AdminGenerated    bool            `json:"admin_generated"`
	AdminUserId       pgtype.Text     `json:"admin_user_id"`
	AdminReason       pgtype.Text     `json:"admin_reason"`
	AdminNotes        pgtype.Text     `json:"admin_notes"`
	
	// Audit trail
	CreatedBy         string          `json:"created_by"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	LastUsedAt        pgtype.Timestamptz `json:"last_used_at"`
	LastUsedIp        pgtype.Text     `json:"last_used_ip"`
}

// ToEntity converts the model to a domain entity
func (m *PaymentUpdateToken) ToEntity() dunning.PaymentUpdateToken {
	entity := dunning.PaymentUpdateToken{
		OrgId:           m.OrgId,
		TokenId:         m.TokenId,
		SubscriptionId:  m.SubscriptionId,
		CustomerId:      m.CustomerId,
		Signature:       m.Signature,
		MaxUses:         m.MaxUses,
		UsedCount:       m.UsedCount,
		Status:          dunning.TokenStatus(m.Status),
		AdminGenerated:  m.AdminGenerated,
		CreatedBy:       m.CreatedBy,
	}
	
	// Handle nullable fields
	if m.DunningCampaignId.Valid {
		entity.DunningCampaignId = m.DunningCampaignId.String
	}
	
	if m.ExpiresAt.Valid {
		entity.ExpiresAt = m.ExpiresAt.Time
	}
	
	if m.AdminUserId.Valid {
		entity.AdminUserId = m.AdminUserId.String
	}
	
	if m.AdminReason.Valid {
		entity.AdminReason = m.AdminReason.String
	}
	
	if m.AdminNotes.Valid {
		entity.AdminNotes = m.AdminNotes.String
	}
	
	if m.CreatedAt.Valid {
		entity.CreatedAt = m.CreatedAt.Time
	}
	
	if m.LastUsedAt.Valid {
		entity.LastUsedAt = m.LastUsedAt.Time
	}
	
	if m.LastUsedIp.Valid {
		entity.LastUsedIp = m.LastUsedIp.String
	}
	
	// Handle JSON fields
	if len(m.TokenData) > 0 {
		var tokenData map[string]interface{}
		_ = json.Unmarshal(m.TokenData, &tokenData)
		entity.TokenData = tokenData
	}
	
	if len(m.AllowedActions) > 0 {
		var allowedActions map[string]bool
		_ = json.Unmarshal(m.AllowedActions, &allowedActions)
		entity.AllowedActions = allowedActions
	}
	
	return entity
}