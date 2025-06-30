package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
)

// DunningAttempt represents a payment retry attempt in a dunning campaign
type DunningAttempt struct {
	OrgId             string          `json:"org_id"`
	Id                string          `json:"id"`
	
	// Relationships
	DunningCampaignId string          `json:"dunning_campaign_id"`
	SubscriptionId    string          `json:"subscription_id"`
	
	// Attempt details
	AttemptNumber     int             `json:"attempt_number"`
	AttemptType       string          `json:"attempt_type"`
	
	// Payment details
	Amount            int             `json:"amount"`
	Currency          string          `json:"currency"`
	PaymentMethodId   pgtype.Text     `json:"payment_method_id"`
	
	// Results
	Status            string          `json:"status"`
	FailureReason     pgtype.Text     `json:"failure_reason"`
	FailureCode       pgtype.Text     `json:"failure_code"`
	ProcessorResponse json.RawMessage `json:"processor_response"`
	
	// Performance metrics
	ProcessingTimeMs  pgtype.Int4     `json:"processing_time_ms"`
	AttemptedAt       pgtype.Timestamptz `json:"attempted_at"`
	CompletedAt       pgtype.Timestamptz `json:"completed_at"`
	
	// Context
	TriggeredBy       pgtype.Text     `json:"triggered_by"`
	Metadata          json.RawMessage `json:"metadata"`
	
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
}

// ToEntity converts the model to a domain entity
func (m *DunningAttempt) ToEntity() dunning.DunningAttempt {
	entity := dunning.DunningAttempt{
		OrgId:             m.OrgId,
		Id:                m.Id,
		DunningCampaignId: m.DunningCampaignId,
		SubscriptionId:    m.SubscriptionId,
		AttemptNumber:     m.AttemptNumber,
		AttemptType:       dunning.DunningAttemptType(m.AttemptType),
		Amount:            m.Amount,
		Currency:          m.Currency,
		Status:            payments.PaymentStatus(m.Status),
	}
	
	// Handle nullable fields
	if m.PaymentMethodId.Valid {
		entity.PaymentMethodId = m.PaymentMethodId.String
	}
	
	if m.FailureReason.Valid {
		entity.FailureReason = m.FailureReason.String
	}
	
	if m.FailureCode.Valid {
		entity.FailureCode = m.FailureCode.String
	}
	
	if m.ProcessingTimeMs.Valid {
		entity.ProcessingTimeMs = int(m.ProcessingTimeMs.Int32)
	}
	
	if m.AttemptedAt.Valid {
		entity.AttemptedAt = m.AttemptedAt.Time
	}
	
	if m.CompletedAt.Valid {
		entity.CompletedAt = m.CompletedAt.Time
	}
	
	if m.TriggeredBy.Valid {
		entity.TriggeredBy = m.TriggeredBy.String
	}
	
	if m.CreatedAt.Valid {
		entity.CreatedAt = m.CreatedAt.Time
	}
	
	// Handle JSON fields
	if len(m.ProcessorResponse) > 0 {
		var processorResponse map[string]interface{}
		_ = json.Unmarshal(m.ProcessorResponse, &processorResponse)
		entity.ProcessorResponse = processorResponse
	}
	
	if len(m.Metadata) > 0 {
		var metadata map[string]string
		_ = json.Unmarshal(m.Metadata, &metadata)
		entity.Metadata = metadata
	}
	
	return entity
}