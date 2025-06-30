package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
)

// DunningCampaign represents a dunning campaign in the database
type DunningCampaign struct {
	OrgId             string          `json:"org_id"`
	Id                string          `json:"id"`

	// Relationships
	SubscriptionId    string          `json:"subscription_id"`
	CustomerId        string          `json:"customer_id"`

	// Workflow metadata
	TemporalWorkflowId string          `json:"temporal_workflow_id"`
	TemporalRunId      string          `json:"temporal_run_id"`
	ParentWorkflowId   pgtype.Text     `json:"parent_workflow_id"`

	// Campaign details
	Status             string          `json:"status"`
	FailedAmount       int             `json:"failed_amount"`
	Currency           string          `json:"currency"`
	InitialFailureReason pgtype.Text   `json:"initial_failure_reason"`

	// Attempt tracking
	TotalAttempts      int             `json:"total_attempts"`
	ImmediateAttempts  int             `json:"immediate_attempts"`
	ProgressiveAttempts int            `json:"progressive_attempts"`

	// Timeline
	StartedAt          pgtype.Timestamptz `json:"started_at"`
	LastAttemptAt      pgtype.Timestamptz `json:"last_attempt_at"`
	NextAttemptAt      pgtype.Timestamptz `json:"next_attempt_at"`
	CompletedAt        pgtype.Timestamptz `json:"completed_at"`

	// Outcomes
	RecoveryMethod     pgtype.Text     `json:"recovery_method"`
	RecoveredAmount    pgtype.Int4     `json:"recovered_amount"`
	RecoveredAt        pgtype.Timestamptz `json:"recovered_at"`
	FinalFailureReason pgtype.Text     `json:"final_failure_reason"`

	// Configuration snapshot
	ConfigSnapshot     json.RawMessage `json:"config_snapshot"`

	// Metadata
	Metadata           json.RawMessage `json:"metadata"`

	CreatedAt          pgtype.Timestamptz `json:"created_at"`
	UpdatedAt          pgtype.Timestamptz `json:"updated_at"`
}

// ToEntity converts the model to a domain entity
func (m *DunningCampaign) ToEntity() dunning.DunningCampaign {
	entity := dunning.DunningCampaign{
		OrgId:             m.OrgId,
		Id:                m.Id,
		SubscriptionId:    m.SubscriptionId,
		CustomerId:        m.CustomerId,
		TemporalWorkflowId: m.TemporalWorkflowId,
		TemporalRunId:     m.TemporalRunId,
		Status:            dunning.DunningStatus(m.Status),
		FailedAmount:      m.FailedAmount,
		Currency:          m.Currency,
		TotalAttempts:     m.TotalAttempts,
		ImmediateAttempts: m.ImmediateAttempts,
		ProgressiveAttempts: m.ProgressiveAttempts,
	}

	// Handle nullable fields
	if m.ParentWorkflowId.Valid {
		entity.ParentWorkflowId = m.ParentWorkflowId.String
	}

	if m.InitialFailureReason.Valid {
		entity.InitialFailureReason = m.InitialFailureReason.String
	}

	if m.StartedAt.Valid {
		entity.StartedAt = m.StartedAt.Time
	}

	if m.LastAttemptAt.Valid {
		entity.LastAttemptAt = m.LastAttemptAt.Time
	}

	if m.NextAttemptAt.Valid {
		entity.NextAttemptAt = m.NextAttemptAt.Time
	}

	if m.CompletedAt.Valid {
		entity.CompletedAt = m.CompletedAt.Time
	}

	if m.RecoveryMethod.Valid {
		entity.RecoveryMethod = m.RecoveryMethod.String
	}

	if m.RecoveredAmount.Valid {
		entity.RecoveredAmount = int(m.RecoveredAmount.Int32)
	}

	if m.RecoveredAt.Valid {
		entity.RecoveredAt = m.RecoveredAt.Time
	}

	if m.FinalFailureReason.Valid {
		entity.FinalFailureReason = m.FinalFailureReason.String
	}

	if m.CreatedAt.Valid {
		entity.CreatedAt = m.CreatedAt.Time
	}

	if m.UpdatedAt.Valid {
		entity.UpdatedAt = m.UpdatedAt.Time
	}

	// Handle JSON fields
	if len(m.ConfigSnapshot) > 0 {
		var configSnapshot map[string]interface{}
		_ = json.Unmarshal(m.ConfigSnapshot, &configSnapshot)
		entity.ConfigSnapshot = configSnapshot
	}

	if len(m.Metadata) > 0 {
		var metadata map[string]string
		_ = json.Unmarshal(m.Metadata, &metadata)
		entity.Metadata = metadata
	}

	return entity
}
