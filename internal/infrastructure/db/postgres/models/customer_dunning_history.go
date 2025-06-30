package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities/dunning"
)

// CustomerDunningHistory represents a summary of a customer's dunning history
type CustomerDunningHistory struct {
	OrgId                    string          `json:"org_id"`
	CustomerId               string          `json:"customer_id"`
	
	// Lifetime stats
	TotalDunningCampaigns    int             `json:"total_dunning_campaigns"`
	SuccessfulRecoveries     int             `json:"successful_recoveries"`
	FailedCampaigns          int             `json:"failed_campaigns"`
	
	// Financial impact
	TotalAmountAtRisk        int             `json:"total_amount_at_risk"`
	TotalAmountRecovered     int             `json:"total_amount_recovered"`
	TotalAmountLost          int             `json:"total_amount_lost"`
	
	// Behavior patterns
	AvgRecoveryTimeHours     pgtype.Float8   `json:"avg_recovery_time_hours"`
	PreferredRecoveryMethod  pgtype.Text     `json:"preferred_recovery_method"`
	MostResponsiveChannel    pgtype.Text     `json:"most_responsive_channel"`
	
	// Risk scoring
	PaymentReliabilityScore  pgtype.Float8   `json:"payment_reliability_score"`
	DunningRiskTier          pgtype.Text     `json:"dunning_risk_tier"`
	
	// Dates
	FirstDunningAt           pgtype.Timestamptz `json:"first_dunning_at"`
	LastDunningAt            pgtype.Timestamptz `json:"last_dunning_at"`
	LastRecoveryAt           pgtype.Timestamptz `json:"last_recovery_at"`
	
	UpdatedAt                pgtype.Timestamptz `json:"updated_at"`
}

// ToEntity converts the model to a domain entity
func (m *CustomerDunningHistory) ToEntity() dunning.CustomerDunningHistory {
	entity := dunning.CustomerDunningHistory{
		OrgId:                 m.OrgId,
		CustomerId:            m.CustomerId,
		TotalDunningCampaigns: m.TotalDunningCampaigns,
		SuccessfulRecoveries:  m.SuccessfulRecoveries,
		FailedCampaigns:       m.FailedCampaigns,
		TotalAmountAtRisk:     m.TotalAmountAtRisk,
		TotalAmountRecovered:  m.TotalAmountRecovered,
		TotalAmountLost:       m.TotalAmountLost,
	}
	
	// Handle nullable fields
	if m.AvgRecoveryTimeHours.Valid {
		entity.AvgRecoveryTimeHours = m.AvgRecoveryTimeHours.Float64
	}
	
	if m.PreferredRecoveryMethod.Valid {
		entity.PreferredRecoveryMethod = m.PreferredRecoveryMethod.String
	}
	
	if m.MostResponsiveChannel.Valid {
		entity.MostResponsiveChannel = dunning.CommunicationChannel(m.MostResponsiveChannel.String)
	}
	
	if m.PaymentReliabilityScore.Valid {
		entity.PaymentReliabilityScore = m.PaymentReliabilityScore.Float64
	}
	
	if m.DunningRiskTier.Valid {
		entity.DunningRiskTier = m.DunningRiskTier.String
	}
	
	if m.FirstDunningAt.Valid {
		entity.FirstDunningAt = m.FirstDunningAt.Time
	}
	
	if m.LastDunningAt.Valid {
		entity.LastDunningAt = m.LastDunningAt.Time
	}
	
	if m.LastRecoveryAt.Valid {
		entity.LastRecoveryAt = m.LastRecoveryAt.Time
	}
	
	if m.UpdatedAt.Valid {
		entity.UpdatedAt = m.UpdatedAt.Time
	}
	
	return entity
}