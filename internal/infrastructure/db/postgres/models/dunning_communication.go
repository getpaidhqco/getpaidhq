package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
)

// DunningCommunication represents a communication sent to a customer during a dunning campaign
type DunningCommunication struct {
	OrgId             string          `json:"org_id"`
	Id                string          `json:"id"`
	
	// Relationships
	DunningCampaignId string          `json:"dunning_campaign_id"`
	CustomerId        string          `json:"customer_id"`
	
	// Communication details
	Channel           string          `json:"channel"`
	TemplateId        string          `json:"template_id"`
	AttemptNumber     int             `json:"attempt_number"`
	
	// Content
	Subject           pgtype.Text     `json:"subject"`
	ContentPreview    pgtype.Text     `json:"content_preview"`
	PersonalizationData json.RawMessage `json:"personalization_data"`
	
	// Delivery tracking
	SentAt            pgtype.Timestamptz `json:"sent_at"`
	DeliveredAt       pgtype.Timestamptz `json:"delivered_at"`
	OpenedAt          pgtype.Timestamptz `json:"opened_at"`
	ClickedAt         pgtype.Timestamptz `json:"clicked_at"`
	BouncedAt         pgtype.Timestamptz `json:"bounced_at"`
	
	// Provider details
	Provider          string          `json:"provider"`
	ProviderMessageId pgtype.Text     `json:"provider_message_id"`
	ProviderResponse  json.RawMessage `json:"provider_response"`
	
	// Status
	Status            string          `json:"status"`
	FailureReason     pgtype.Text     `json:"failure_reason"`
	
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	UpdatedAt         pgtype.Timestamptz `json:"updated_at"`
}

// ToEntity converts the model to a domain entity
func (m *DunningCommunication) ToEntity() dunning.DunningCommunication {
	entity := dunning.DunningCommunication{
		OrgId:             m.OrgId,
		Id:                m.Id,
		DunningCampaignId: m.DunningCampaignId,
		CustomerId:        m.CustomerId,
		Channel:           dunning.CommunicationChannel(m.Channel),
		TemplateId:        m.TemplateId,
		AttemptNumber:     m.AttemptNumber,
		Provider:          m.Provider,
		Status:            dunning.CommunicationStatus(m.Status),
	}
	
	// Handle nullable fields
	if m.Subject.Valid {
		entity.Subject = m.Subject.String
	}
	
	if m.ContentPreview.Valid {
		entity.ContentPreview = m.ContentPreview.String
	}
	
	if m.SentAt.Valid {
		entity.SentAt = m.SentAt.Time
	}
	
	if m.DeliveredAt.Valid {
		entity.DeliveredAt = m.DeliveredAt.Time
	}
	
	if m.OpenedAt.Valid {
		entity.OpenedAt = m.OpenedAt.Time
	}
	
	if m.ClickedAt.Valid {
		entity.ClickedAt = m.ClickedAt.Time
	}
	
	if m.BouncedAt.Valid {
		entity.BouncedAt = m.BouncedAt.Time
	}
	
	if m.ProviderMessageId.Valid {
		entity.ProviderMessageId = m.ProviderMessageId.String
	}
	
	if m.FailureReason.Valid {
		entity.FailureReason = m.FailureReason.String
	}
	
	if m.CreatedAt.Valid {
		entity.CreatedAt = m.CreatedAt.Time
	}
	
	if m.UpdatedAt.Valid {
		entity.UpdatedAt = m.UpdatedAt.Time
	}
	
	// Handle JSON fields
	if len(m.PersonalizationData) > 0 {
		var personalizationData map[string]interface{}
		_ = json.Unmarshal(m.PersonalizationData, &personalizationData)
		entity.PersonalizationData = personalizationData
	}
	
	if len(m.ProviderResponse) > 0 {
		var providerResponse map[string]interface{}
		_ = json.Unmarshal(m.ProviderResponse, &providerResponse)
		entity.ProviderResponse = providerResponse
	}
	
	return entity
}