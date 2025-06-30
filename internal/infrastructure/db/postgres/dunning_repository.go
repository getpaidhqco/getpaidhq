package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

// DunningRepository implements the repositories.DunningRepository interface
type DunningRepository struct {
	*PgDatabase
	logger logger.Logger
}

// NewDunningRepository creates a new DunningRepository
func NewDunningRepository(primaryDb lib.Database, logger logger.Logger) repositories.DunningRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return DunningRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// Campaign operations

// CreateCampaign creates a new dunning campaign
func (r DunningRepository) CreateCampaign(ctx context.Context, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	tx := r.getTransactionFromContext(ctx)

	metadataJson, _ := json.Marshal(campaign.Metadata)
	configSnapshotJson, _ := json.Marshal(campaign.ConfigSnapshot)

	query := `INSERT INTO dunning_campaigns (
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at
	) VALUES (
		@org_id, @id, @subscription_id, @customer_id,
		@temporal_workflow_id, @temporal_run_id, @parent_workflow_id,
		@status, @failed_amount, @currency, @initial_failure_reason,
		@total_attempts, @immediate_attempts, @progressive_attempts,
		@started_at, @last_attempt_at, @next_attempt_at, @completed_at,
		@recovery_method, @recovered_amount, @recovered_at, @final_failure_reason,
		@config_snapshot, @metadata, NOW(), NOW()
	) RETURNING *`

	var campaignModel models.DunningCampaign
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":                campaign.OrgId,
		"id":                    campaign.Id,
		"subscription_id":       campaign.SubscriptionId,
		"customer_id":           campaign.CustomerId,
		"temporal_workflow_id":  campaign.TemporalWorkflowId,
		"temporal_run_id":       campaign.TemporalRunId,
		"parent_workflow_id":    pgtype.Text{String: campaign.ParentWorkflowId, Valid: campaign.ParentWorkflowId != ""},
		"status":                string(campaign.Status),
		"failed_amount":         campaign.FailedAmount,
		"currency":              campaign.Currency,
		"initial_failure_reason": pgtype.Text{String: campaign.InitialFailureReason, Valid: campaign.InitialFailureReason != ""},
		"total_attempts":        campaign.TotalAttempts,
		"immediate_attempts":    campaign.ImmediateAttempts,
		"progressive_attempts":  campaign.ProgressiveAttempts,
		"started_at":            pgtype.Timestamptz{Time: campaign.StartedAt, Valid: !campaign.StartedAt.IsZero()},
		"last_attempt_at":       pgtype.Timestamptz{Time: campaign.LastAttemptAt, Valid: !campaign.LastAttemptAt.IsZero()},
		"next_attempt_at":       pgtype.Timestamptz{Time: campaign.NextAttemptAt, Valid: !campaign.NextAttemptAt.IsZero()},
		"completed_at":          pgtype.Timestamptz{Time: campaign.CompletedAt, Valid: !campaign.CompletedAt.IsZero()},
		"recovery_method":       pgtype.Text{String: campaign.RecoveryMethod, Valid: campaign.RecoveryMethod != ""},
		"recovered_amount":      pgtype.Int4{Int32: int32(campaign.RecoveredAmount), Valid: campaign.RecoveredAmount != 0},
		"recovered_at":          pgtype.Timestamptz{Time: campaign.RecoveredAt, Valid: !campaign.RecoveredAt.IsZero()},
		"final_failure_reason":  pgtype.Text{String: campaign.FinalFailureReason, Valid: campaign.FinalFailureReason != ""},
		"config_snapshot":       configSnapshotJson,
		"metadata":              metadataJson,
	}).Scan(
		&campaignModel.OrgId,
		&campaignModel.Id,
		&campaignModel.SubscriptionId,
		&campaignModel.CustomerId,
		&campaignModel.TemporalWorkflowId,
		&campaignModel.TemporalRunId,
		&campaignModel.ParentWorkflowId,
		&campaignModel.Status,
		&campaignModel.FailedAmount,
		&campaignModel.Currency,
		&campaignModel.InitialFailureReason,
		&campaignModel.TotalAttempts,
		&campaignModel.ImmediateAttempts,
		&campaignModel.ProgressiveAttempts,
		&campaignModel.StartedAt,
		&campaignModel.LastAttemptAt,
		&campaignModel.NextAttemptAt,
		&campaignModel.CompletedAt,
		&campaignModel.RecoveryMethod,
		&campaignModel.RecoveredAmount,
		&campaignModel.RecoveredAt,
		&campaignModel.FinalFailureReason,
		&campaignModel.ConfigSnapshot,
		&campaignModel.Metadata,
		&campaignModel.CreatedAt,
		&campaignModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to create DunningCampaign`, err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaignModel.ToEntity(), nil
}

// FindCampaignById finds a dunning campaign by ID
func (r DunningRepository) FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at
	FROM dunning_campaigns
	WHERE org_id = $1 AND id = $2`

	var campaignModel models.DunningCampaign
	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&campaignModel.OrgId,
		&campaignModel.Id,
		&campaignModel.SubscriptionId,
		&campaignModel.CustomerId,
		&campaignModel.TemporalWorkflowId,
		&campaignModel.TemporalRunId,
		&campaignModel.ParentWorkflowId,
		&campaignModel.Status,
		&campaignModel.FailedAmount,
		&campaignModel.Currency,
		&campaignModel.InitialFailureReason,
		&campaignModel.TotalAttempts,
		&campaignModel.ImmediateAttempts,
		&campaignModel.ProgressiveAttempts,
		&campaignModel.StartedAt,
		&campaignModel.LastAttemptAt,
		&campaignModel.NextAttemptAt,
		&campaignModel.CompletedAt,
		&campaignModel.RecoveryMethod,
		&campaignModel.RecoveredAmount,
		&campaignModel.RecoveredAt,
		&campaignModel.FinalFailureReason,
		&campaignModel.ConfigSnapshot,
		&campaignModel.Metadata,
		&campaignModel.CreatedAt,
		&campaignModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find DunningCampaign by id`, err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaignModel.ToEntity(), nil
}

// FindCampaigns finds all dunning campaigns for an organization with pagination
func (r DunningRepository) FindCampaigns(ctx context.Context, orgId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", pagination.SortDirection, pagination.SortBy)

	var campaigns = make([]dunning.DunningCampaign, 0)
	var count int

	query := `SELECT 
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at,
		count(*) OVER()
	FROM dunning_campaigns
	WHERE org_id = @org_id
	ORDER BY
		CASE
			WHEN @sort_dir = 'asc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					WHEN 'status' THEN status
					ELSE created_at
				END
			ELSE NULL
		END ASC,
		CASE
			WHEN @sort_dir = 'desc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					WHEN 'status' THEN status
					ELSE created_at
				END
			ELSE NULL
		END DESC
	LIMIT @limit OFFSET @offset`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"limit":    pagination.Limit,
		"offset":   pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find DunningCampaigns`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var campaignModel models.DunningCampaign
		err := rows.Scan(
			&campaignModel.OrgId,
			&campaignModel.Id,
			&campaignModel.SubscriptionId,
			&campaignModel.CustomerId,
			&campaignModel.TemporalWorkflowId,
			&campaignModel.TemporalRunId,
			&campaignModel.ParentWorkflowId,
			&campaignModel.Status,
			&campaignModel.FailedAmount,
			&campaignModel.Currency,
			&campaignModel.InitialFailureReason,
			&campaignModel.TotalAttempts,
			&campaignModel.ImmediateAttempts,
			&campaignModel.ProgressiveAttempts,
			&campaignModel.StartedAt,
			&campaignModel.LastAttemptAt,
			&campaignModel.NextAttemptAt,
			&campaignModel.CompletedAt,
			&campaignModel.RecoveryMethod,
			&campaignModel.RecoveredAmount,
			&campaignModel.RecoveredAt,
			&campaignModel.FinalFailureReason,
			&campaignModel.ConfigSnapshot,
			&campaignModel.Metadata,
			&campaignModel.CreatedAt,
			&campaignModel.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningCampaign`, err.Error())
			return nil, 0, err
		}
		campaigns = append(campaigns, campaignModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return campaigns, count, nil
}

// FindCampaignsBySubscriptionId finds all dunning campaigns for a subscription with pagination
func (r DunningRepository) FindCampaignsBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", pagination.SortDirection, pagination.SortBy)

	var campaigns = make([]dunning.DunningCampaign, 0)
	var count int

	query := `SELECT 
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at,
		count(*) OVER()
	FROM dunning_campaigns
	WHERE org_id = $1 AND subscription_id = $2
	ORDER BY created_at DESC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, subscriptionId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find DunningCampaigns by subscription id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var campaignModel models.DunningCampaign
		err := rows.Scan(
			&campaignModel.OrgId,
			&campaignModel.Id,
			&campaignModel.SubscriptionId,
			&campaignModel.CustomerId,
			&campaignModel.TemporalWorkflowId,
			&campaignModel.TemporalRunId,
			&campaignModel.ParentWorkflowId,
			&campaignModel.Status,
			&campaignModel.FailedAmount,
			&campaignModel.Currency,
			&campaignModel.InitialFailureReason,
			&campaignModel.TotalAttempts,
			&campaignModel.ImmediateAttempts,
			&campaignModel.ProgressiveAttempts,
			&campaignModel.StartedAt,
			&campaignModel.LastAttemptAt,
			&campaignModel.NextAttemptAt,
			&campaignModel.CompletedAt,
			&campaignModel.RecoveryMethod,
			&campaignModel.RecoveredAmount,
			&campaignModel.RecoveredAt,
			&campaignModel.FinalFailureReason,
			&campaignModel.ConfigSnapshot,
			&campaignModel.Metadata,
			&campaignModel.CreatedAt,
			&campaignModel.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningCampaign`, err.Error())
			return nil, 0, err
		}
		campaigns = append(campaigns, campaignModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return campaigns, count, nil
}

// FindCampaignsByCustomerId finds all dunning campaigns for a customer with pagination
func (r DunningRepository) FindCampaignsByCustomerId(ctx context.Context, orgId string, customerId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", pagination.SortDirection, pagination.SortBy)

	var campaigns = make([]dunning.DunningCampaign, 0)
	var count int

	query := `SELECT 
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at,
		count(*) OVER()
	FROM dunning_campaigns
	WHERE org_id = $1 AND customer_id = $2
	ORDER BY created_at DESC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, customerId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find DunningCampaigns by customer id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var campaignModel models.DunningCampaign
		err := rows.Scan(
			&campaignModel.OrgId,
			&campaignModel.Id,
			&campaignModel.SubscriptionId,
			&campaignModel.CustomerId,
			&campaignModel.TemporalWorkflowId,
			&campaignModel.TemporalRunId,
			&campaignModel.ParentWorkflowId,
			&campaignModel.Status,
			&campaignModel.FailedAmount,
			&campaignModel.Currency,
			&campaignModel.InitialFailureReason,
			&campaignModel.TotalAttempts,
			&campaignModel.ImmediateAttempts,
			&campaignModel.ProgressiveAttempts,
			&campaignModel.StartedAt,
			&campaignModel.LastAttemptAt,
			&campaignModel.NextAttemptAt,
			&campaignModel.CompletedAt,
			&campaignModel.RecoveryMethod,
			&campaignModel.RecoveredAmount,
			&campaignModel.RecoveredAt,
			&campaignModel.FinalFailureReason,
			&campaignModel.ConfigSnapshot,
			&campaignModel.Metadata,
			&campaignModel.CreatedAt,
			&campaignModel.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningCampaign`, err.Error())
			return nil, 0, err
		}
		campaigns = append(campaigns, campaignModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return campaigns, count, nil
}

// UpdateCampaign updates a dunning campaign
func (r DunningRepository) UpdateCampaign(ctx context.Context, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	tx := r.getTransactionFromContext(ctx)

	metadataJson, _ := json.Marshal(campaign.Metadata)
	configSnapshotJson, _ := json.Marshal(campaign.ConfigSnapshot)

	query := `UPDATE dunning_campaigns
	SET 
		status = $3,
		failed_amount = $4,
		currency = $5,
		initial_failure_reason = $6,
		total_attempts = $7,
		immediate_attempts = $8,
		progressive_attempts = $9,
		started_at = $10,
		last_attempt_at = $11,
		next_attempt_at = $12,
		completed_at = $13,
		recovery_method = $14,
		recovered_amount = $15,
		recovered_at = $16,
		final_failure_reason = $17,
		config_snapshot = $18,
		metadata = $19,
		updated_at = NOW()
	WHERE org_id = $1 AND id = $2
	RETURNING 
		org_id, id, subscription_id, customer_id, 
		temporal_workflow_id, temporal_run_id, parent_workflow_id,
		status, failed_amount, currency, initial_failure_reason,
		total_attempts, immediate_attempts, progressive_attempts,
		started_at, last_attempt_at, next_attempt_at, completed_at,
		recovery_method, recovered_amount, recovered_at, final_failure_reason,
		config_snapshot, metadata, created_at, updated_at`

	var campaignModel models.DunningCampaign
	err := tx.QueryRow(ctx, query,
		campaign.OrgId,
		campaign.Id,
		string(campaign.Status),
		campaign.FailedAmount,
		campaign.Currency,
		campaign.InitialFailureReason,
		campaign.TotalAttempts,
		campaign.ImmediateAttempts,
		campaign.ProgressiveAttempts,
		campaign.StartedAt,
		campaign.LastAttemptAt,
		campaign.NextAttemptAt,
		campaign.CompletedAt,
		campaign.RecoveryMethod,
		campaign.RecoveredAmount,
		campaign.RecoveredAt,
		campaign.FinalFailureReason,
		configSnapshotJson,
		metadataJson,
	).Scan(
		&campaignModel.OrgId,
		&campaignModel.Id,
		&campaignModel.SubscriptionId,
		&campaignModel.CustomerId,
		&campaignModel.TemporalWorkflowId,
		&campaignModel.TemporalRunId,
		&campaignModel.ParentWorkflowId,
		&campaignModel.Status,
		&campaignModel.FailedAmount,
		&campaignModel.Currency,
		&campaignModel.InitialFailureReason,
		&campaignModel.TotalAttempts,
		&campaignModel.ImmediateAttempts,
		&campaignModel.ProgressiveAttempts,
		&campaignModel.StartedAt,
		&campaignModel.LastAttemptAt,
		&campaignModel.NextAttemptAt,
		&campaignModel.CompletedAt,
		&campaignModel.RecoveryMethod,
		&campaignModel.RecoveredAmount,
		&campaignModel.RecoveredAt,
		&campaignModel.FinalFailureReason,
		&campaignModel.ConfigSnapshot,
		&campaignModel.Metadata,
		&campaignModel.CreatedAt,
		&campaignModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to update DunningCampaign`, err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaignModel.ToEntity(), nil
}

// Attempt operations

// CreateAttempt creates a new dunning attempt
func (r DunningRepository) CreateAttempt(ctx context.Context, attempt dunning.DunningAttempt) (dunning.DunningAttempt, error) {
	tx := r.getTransactionFromContext(ctx)

	metadataJson, _ := json.Marshal(attempt.Metadata)
	processorResponseJson, _ := json.Marshal(attempt.ProcessorResponse)

	query := `INSERT INTO dunning_attempts (
		org_id, id, dunning_campaign_id, subscription_id,
		attempt_number, attempt_type, 
		amount, currency, payment_method_id,
		status, failure_reason, failure_code, processor_response,
		processing_time_ms, attempted_at, completed_at,
		triggered_by, metadata, created_at
	) VALUES (
		$1, $2, $3, $4, 
		$5, $6, 
		$7, $8, $9, 
		$10, $11, $12, $13, 
		$14, $15, $16, 
		$17, $18, NOW()
	) RETURNING 
		org_id, id, dunning_campaign_id, subscription_id,
		attempt_number, attempt_type, 
		amount, currency, payment_method_id,
		status, failure_reason, failure_code, processor_response,
		processing_time_ms, attempted_at, completed_at,
		triggered_by, metadata, created_at`

	var attemptModel models.DunningAttempt
	err := tx.QueryRow(ctx, query,
		attempt.OrgId,
		attempt.Id,
		attempt.DunningCampaignId,
		attempt.SubscriptionId,
		attempt.AttemptNumber,
		string(attempt.AttemptType),
		attempt.Amount,
		attempt.Currency,
		attempt.PaymentMethodId,
		string(attempt.Status),
		attempt.FailureReason,
		attempt.FailureCode,
		processorResponseJson,
		attempt.ProcessingTimeMs,
		attempt.AttemptedAt,
		attempt.CompletedAt,
		attempt.TriggeredBy,
		metadataJson,
	).Scan(
		&attemptModel.OrgId,
		&attemptModel.Id,
		&attemptModel.DunningCampaignId,
		&attemptModel.SubscriptionId,
		&attemptModel.AttemptNumber,
		&attemptModel.AttemptType,
		&attemptModel.Amount,
		&attemptModel.Currency,
		&attemptModel.PaymentMethodId,
		&attemptModel.Status,
		&attemptModel.FailureReason,
		&attemptModel.FailureCode,
		&attemptModel.ProcessorResponse,
		&attemptModel.ProcessingTimeMs,
		&attemptModel.AttemptedAt,
		&attemptModel.CompletedAt,
		&attemptModel.TriggeredBy,
		&attemptModel.Metadata,
		&attemptModel.CreatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to create DunningAttempt`, err.Error())
		return dunning.DunningAttempt{}, err
	}

	return attemptModel.ToEntity(), nil
}

// FindAttemptById finds a dunning attempt by ID
func (r DunningRepository) FindAttemptById(ctx context.Context, orgId string, id string) (dunning.DunningAttempt, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, dunning_campaign_id, subscription_id,
		attempt_number, attempt_type, 
		amount, currency, payment_method_id,
		status, failure_reason, failure_code, processor_response,
		processing_time_ms, attempted_at, completed_at,
		triggered_by, metadata, created_at
	FROM dunning_attempts
	WHERE org_id = $1 AND id = $2`

	var attemptModel models.DunningAttempt
	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&attemptModel.OrgId,
		&attemptModel.Id,
		&attemptModel.DunningCampaignId,
		&attemptModel.SubscriptionId,
		&attemptModel.AttemptNumber,
		&attemptModel.AttemptType,
		&attemptModel.Amount,
		&attemptModel.Currency,
		&attemptModel.PaymentMethodId,
		&attemptModel.Status,
		&attemptModel.FailureReason,
		&attemptModel.FailureCode,
		&attemptModel.ProcessorResponse,
		&attemptModel.ProcessingTimeMs,
		&attemptModel.AttemptedAt,
		&attemptModel.CompletedAt,
		&attemptModel.TriggeredBy,
		&attemptModel.Metadata,
		&attemptModel.CreatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find DunningAttempt by id`, err.Error())
		return dunning.DunningAttempt{}, err
	}

	return attemptModel.ToEntity(), nil
}

// FindAttemptsByCampaignId finds all dunning attempts for a campaign with pagination
func (r DunningRepository) FindAttemptsByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.DunningAttempt, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var attempts = make([]dunning.DunningAttempt, 0)
	var count int

	query := `SELECT 
		org_id, id, dunning_campaign_id, subscription_id,
		attempt_number, attempt_type, 
		amount, currency, payment_method_id,
		status, failure_reason, failure_code, processor_response,
		processing_time_ms, attempted_at, completed_at,
		triggered_by, metadata, created_at,
		count(*) OVER()
	FROM dunning_attempts
	WHERE org_id = $1 AND dunning_campaign_id = $2
	ORDER BY attempt_number ASC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, campaignId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find DunningAttempts by campaign id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var attemptModel models.DunningAttempt
		err := rows.Scan(
			&attemptModel.OrgId,
			&attemptModel.Id,
			&attemptModel.DunningCampaignId,
			&attemptModel.SubscriptionId,
			&attemptModel.AttemptNumber,
			&attemptModel.AttemptType,
			&attemptModel.Amount,
			&attemptModel.Currency,
			&attemptModel.PaymentMethodId,
			&attemptModel.Status,
			&attemptModel.FailureReason,
			&attemptModel.FailureCode,
			&attemptModel.ProcessorResponse,
			&attemptModel.ProcessingTimeMs,
			&attemptModel.AttemptedAt,
			&attemptModel.CompletedAt,
			&attemptModel.TriggeredBy,
			&attemptModel.Metadata,
			&attemptModel.CreatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningAttempt`, err.Error())
			return nil, 0, err
		}
		attempts = append(attempts, attemptModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return attempts, count, nil
}

// Communication operations

// CreateCommunication creates a new dunning communication
func (r DunningRepository) CreateCommunication(ctx context.Context, communication dunning.DunningCommunication) (dunning.DunningCommunication, error) {
	tx := r.getTransactionFromContext(ctx)

	metadataJson, _ := json.Marshal(communication.PersonalizationData)
	providerResponseJson, _ := json.Marshal(communication.ProviderResponse)

	query := `INSERT INTO dunning_communications (
		org_id, id, dunning_campaign_id, customer_id,
		channel, template_id, attempt_number,
		subject, content_preview, personalization_data,
		sent_at, delivered_at, opened_at, clicked_at, bounced_at,
		provider, provider_message_id, provider_response,
		status, failure_reason,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7,
		$8, $9, $10,
		$11, $12, $13, $14, $15,
		$16, $17, $18,
		$19, $20,
		NOW(), NOW()
	) RETURNING 
		org_id, id, dunning_campaign_id, customer_id,
		channel, template_id, attempt_number,
		subject, content_preview, personalization_data,
		sent_at, delivered_at, opened_at, clicked_at, bounced_at,
		provider, provider_message_id, provider_response,
		status, failure_reason,
		created_at, updated_at`

	var communicationModel models.DunningCommunication
	err := tx.QueryRow(ctx, query,
		communication.OrgId,
		communication.Id,
		communication.DunningCampaignId,
		communication.CustomerId,
		string(communication.Channel),
		communication.TemplateId,
		communication.AttemptNumber,
		communication.Subject,
		communication.ContentPreview,
		metadataJson,
		communication.SentAt,
		communication.DeliveredAt,
		communication.OpenedAt,
		communication.ClickedAt,
		communication.BouncedAt,
		communication.Provider,
		communication.ProviderMessageId,
		providerResponseJson,
		string(communication.Status),
		communication.FailureReason,
	).Scan(
		&communicationModel.OrgId,
		&communicationModel.Id,
		&communicationModel.DunningCampaignId,
		&communicationModel.CustomerId,
		&communicationModel.Channel,
		&communicationModel.TemplateId,
		&communicationModel.AttemptNumber,
		&communicationModel.Subject,
		&communicationModel.ContentPreview,
		&communicationModel.PersonalizationData,
		&communicationModel.SentAt,
		&communicationModel.DeliveredAt,
		&communicationModel.OpenedAt,
		&communicationModel.ClickedAt,
		&communicationModel.BouncedAt,
		&communicationModel.Provider,
		&communicationModel.ProviderMessageId,
		&communicationModel.ProviderResponse,
		&communicationModel.Status,
		&communicationModel.FailureReason,
		&communicationModel.CreatedAt,
		&communicationModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to create DunningCommunication`, err.Error())
		return dunning.DunningCommunication{}, err
	}

	return communicationModel.ToEntity(), nil
}

// FindCommunicationById finds a dunning communication by ID
func (r DunningRepository) FindCommunicationById(ctx context.Context, orgId string, id string) (dunning.DunningCommunication, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, dunning_campaign_id, customer_id,
		channel, template_id, attempt_number,
		subject, content_preview, personalization_data,
		sent_at, delivered_at, opened_at, clicked_at, bounced_at,
		provider, provider_message_id, provider_response,
		status, failure_reason,
		created_at, updated_at
	FROM dunning_communications
	WHERE org_id = $1 AND id = $2`

	var communicationModel models.DunningCommunication
	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&communicationModel.OrgId,
		&communicationModel.Id,
		&communicationModel.DunningCampaignId,
		&communicationModel.CustomerId,
		&communicationModel.Channel,
		&communicationModel.TemplateId,
		&communicationModel.AttemptNumber,
		&communicationModel.Subject,
		&communicationModel.ContentPreview,
		&communicationModel.PersonalizationData,
		&communicationModel.SentAt,
		&communicationModel.DeliveredAt,
		&communicationModel.OpenedAt,
		&communicationModel.ClickedAt,
		&communicationModel.BouncedAt,
		&communicationModel.Provider,
		&communicationModel.ProviderMessageId,
		&communicationModel.ProviderResponse,
		&communicationModel.Status,
		&communicationModel.FailureReason,
		&communicationModel.CreatedAt,
		&communicationModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find DunningCommunication by id`, err.Error())
		return dunning.DunningCommunication{}, err
	}

	return communicationModel.ToEntity(), nil
}

// FindCommunicationsByCampaignId finds all dunning communications for a campaign with pagination
func (r DunningRepository) FindCommunicationsByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.DunningCommunication, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var communications = make([]dunning.DunningCommunication, 0)
	var count int

	query := `SELECT 
		org_id, id, dunning_campaign_id, customer_id,
		channel, template_id, attempt_number,
		subject, content_preview, personalization_data,
		sent_at, delivered_at, opened_at, clicked_at, bounced_at,
		provider, provider_message_id, provider_response,
		status, failure_reason,
		created_at, updated_at,
		count(*) OVER()
	FROM dunning_communications
	WHERE org_id = $1 AND dunning_campaign_id = $2
	ORDER BY attempt_number ASC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, campaignId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find DunningCommunications by campaign id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var communicationModel models.DunningCommunication
		err := rows.Scan(
			&communicationModel.OrgId,
			&communicationModel.Id,
			&communicationModel.DunningCampaignId,
			&communicationModel.CustomerId,
			&communicationModel.Channel,
			&communicationModel.TemplateId,
			&communicationModel.AttemptNumber,
			&communicationModel.Subject,
			&communicationModel.ContentPreview,
			&communicationModel.PersonalizationData,
			&communicationModel.SentAt,
			&communicationModel.DeliveredAt,
			&communicationModel.OpenedAt,
			&communicationModel.ClickedAt,
			&communicationModel.BouncedAt,
			&communicationModel.Provider,
			&communicationModel.ProviderMessageId,
			&communicationModel.ProviderResponse,
			&communicationModel.Status,
			&communicationModel.FailureReason,
			&communicationModel.CreatedAt,
			&communicationModel.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningCommunication`, err.Error())
			return nil, 0, err
		}
		communications = append(communications, communicationModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return communications, count, nil
}

// UpdateCommunication updates a dunning communication
func (r DunningRepository) UpdateCommunication(ctx context.Context, communication dunning.DunningCommunication) (dunning.DunningCommunication, error) {
	tx := r.getTransactionFromContext(ctx)

	personalizationDataJson, _ := json.Marshal(communication.PersonalizationData)
	providerResponseJson, _ := json.Marshal(communication.ProviderResponse)

	query := `UPDATE dunning_communications
	SET 
		channel = $3,
		template_id = $4,
		attempt_number = $5,
		subject = $6,
		content_preview = $7,
		personalization_data = $8,
		sent_at = $9,
		delivered_at = $10,
		opened_at = $11,
		clicked_at = $12,
		bounced_at = $13,
		provider = $14,
		provider_message_id = $15,
		provider_response = $16,
		status = $17,
		failure_reason = $18,
		updated_at = NOW()
	WHERE org_id = $1 AND id = $2
	RETURNING 
		org_id, id, dunning_campaign_id, customer_id,
		channel, template_id, attempt_number,
		subject, content_preview, personalization_data,
		sent_at, delivered_at, opened_at, clicked_at, bounced_at,
		provider, provider_message_id, provider_response,
		status, failure_reason,
		created_at, updated_at`

	var communicationModel models.DunningCommunication
	err := tx.QueryRow(ctx, query,
		communication.OrgId,
		communication.Id,
		string(communication.Channel),
		communication.TemplateId,
		communication.AttemptNumber,
		communication.Subject,
		communication.ContentPreview,
		personalizationDataJson,
		communication.SentAt,
		communication.DeliveredAt,
		communication.OpenedAt,
		communication.ClickedAt,
		communication.BouncedAt,
		communication.Provider,
		communication.ProviderMessageId,
		providerResponseJson,
		string(communication.Status),
		communication.FailureReason,
	).Scan(
		&communicationModel.OrgId,
		&communicationModel.Id,
		&communicationModel.DunningCampaignId,
		&communicationModel.CustomerId,
		&communicationModel.Channel,
		&communicationModel.TemplateId,
		&communicationModel.AttemptNumber,
		&communicationModel.Subject,
		&communicationModel.ContentPreview,
		&communicationModel.PersonalizationData,
		&communicationModel.SentAt,
		&communicationModel.DeliveredAt,
		&communicationModel.OpenedAt,
		&communicationModel.ClickedAt,
		&communicationModel.BouncedAt,
		&communicationModel.Provider,
		&communicationModel.ProviderMessageId,
		&communicationModel.ProviderResponse,
		&communicationModel.Status,
		&communicationModel.FailureReason,
		&communicationModel.CreatedAt,
		&communicationModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to update DunningCommunication`, err.Error())
		return dunning.DunningCommunication{}, err
	}

	return communicationModel.ToEntity(), nil
}

// Token operations

// CreateToken creates a new payment update token
func (r DunningRepository) CreateToken(ctx context.Context, token dunning.PaymentUpdateToken) (dunning.PaymentUpdateToken, error) {
	tx := r.getTransactionFromContext(ctx)

	tokenDataJson, _ := json.Marshal(token.TokenData)
	allowedActionsJson, _ := json.Marshal(token.AllowedActions)

	query := `INSERT INTO payment_update_tokens (
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7,
		$8, $9, $10, $11,
		$12,
		$13, $14, $15, $16,
		$17, NOW(), $18, $19
	) RETURNING 
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip`

	var tokenModel models.PaymentUpdateToken
	err := tx.QueryRow(ctx, query,
		token.OrgId,
		token.TokenId,
		token.SubscriptionId,
		token.CustomerId,
		token.DunningCampaignId,
		tokenDataJson,
		token.Signature,
		token.ExpiresAt,
		token.MaxUses,
		token.UsedCount,
		string(token.Status),
		allowedActionsJson,
		token.AdminGenerated,
		token.AdminUserId,
		token.AdminReason,
		token.AdminNotes,
		token.CreatedBy,
		token.LastUsedAt,
		token.LastUsedIp,
	).Scan(
		&tokenModel.OrgId,
		&tokenModel.TokenId,
		&tokenModel.SubscriptionId,
		&tokenModel.CustomerId,
		&tokenModel.DunningCampaignId,
		&tokenModel.TokenData,
		&tokenModel.Signature,
		&tokenModel.ExpiresAt,
		&tokenModel.MaxUses,
		&tokenModel.UsedCount,
		&tokenModel.Status,
		&tokenModel.AllowedActions,
		&tokenModel.AdminGenerated,
		&tokenModel.AdminUserId,
		&tokenModel.AdminReason,
		&tokenModel.AdminNotes,
		&tokenModel.CreatedBy,
		&tokenModel.CreatedAt,
		&tokenModel.LastUsedAt,
		&tokenModel.LastUsedIp,
	)

	if err != nil {
		r.logger.Error(`failed to create PaymentUpdateToken`, err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	return tokenModel.ToEntity(), nil
}

// FindTokenById finds a payment update token by ID
func (r DunningRepository) FindTokenById(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip
	FROM payment_update_tokens
	WHERE org_id = $1 AND token_id = $2`

	var tokenModel models.PaymentUpdateToken
	err := tx.QueryRow(ctx, query, orgId, tokenId).Scan(
		&tokenModel.OrgId,
		&tokenModel.TokenId,
		&tokenModel.SubscriptionId,
		&tokenModel.CustomerId,
		&tokenModel.DunningCampaignId,
		&tokenModel.TokenData,
		&tokenModel.Signature,
		&tokenModel.ExpiresAt,
		&tokenModel.MaxUses,
		&tokenModel.UsedCount,
		&tokenModel.Status,
		&tokenModel.AllowedActions,
		&tokenModel.AdminGenerated,
		&tokenModel.AdminUserId,
		&tokenModel.AdminReason,
		&tokenModel.AdminNotes,
		&tokenModel.CreatedBy,
		&tokenModel.CreatedAt,
		&tokenModel.LastUsedAt,
		&tokenModel.LastUsedIp,
	)

	if err != nil {
		r.logger.Error(`failed to find PaymentUpdateToken by id`, err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	return tokenModel.ToEntity(), nil
}

// FindTokensBySubscriptionId finds all payment update tokens for a subscription with pagination
func (r DunningRepository) FindTokensBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination entities.Pagination) ([]dunning.PaymentUpdateToken, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var tokens = make([]dunning.PaymentUpdateToken, 0)
	var count int

	query := `SELECT 
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip,
		count(*) OVER()
	FROM payment_update_tokens
	WHERE org_id = $1 AND subscription_id = $2
	ORDER BY created_at DESC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, subscriptionId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find PaymentUpdateTokens by subscription id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var tokenModel models.PaymentUpdateToken
		err := rows.Scan(
			&tokenModel.OrgId,
			&tokenModel.TokenId,
			&tokenModel.SubscriptionId,
			&tokenModel.CustomerId,
			&tokenModel.DunningCampaignId,
			&tokenModel.TokenData,
			&tokenModel.Signature,
			&tokenModel.ExpiresAt,
			&tokenModel.MaxUses,
			&tokenModel.UsedCount,
			&tokenModel.Status,
			&tokenModel.AllowedActions,
			&tokenModel.AdminGenerated,
			&tokenModel.AdminUserId,
			&tokenModel.AdminReason,
			&tokenModel.AdminNotes,
			&tokenModel.CreatedBy,
			&tokenModel.CreatedAt,
			&tokenModel.LastUsedAt,
			&tokenModel.LastUsedIp,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan PaymentUpdateToken`, err.Error())
			return nil, 0, err
		}
		tokens = append(tokens, tokenModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return tokens, count, nil
}

// FindTokensByCampaignId finds all payment update tokens for a campaign with pagination
func (r DunningRepository) FindTokensByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.PaymentUpdateToken, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var tokens = make([]dunning.PaymentUpdateToken, 0)
	var count int

	query := `SELECT 
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip,
		count(*) OVER()
	FROM payment_update_tokens
	WHERE org_id = $1 AND dunning_campaign_id = $2
	ORDER BY created_at DESC
	LIMIT $3 OFFSET $4`

	rows, err := tx.Query(ctx, query, orgId, campaignId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find PaymentUpdateTokens by campaign id`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var tokenModel models.PaymentUpdateToken
		err := rows.Scan(
			&tokenModel.OrgId,
			&tokenModel.TokenId,
			&tokenModel.SubscriptionId,
			&tokenModel.CustomerId,
			&tokenModel.DunningCampaignId,
			&tokenModel.TokenData,
			&tokenModel.Signature,
			&tokenModel.ExpiresAt,
			&tokenModel.MaxUses,
			&tokenModel.UsedCount,
			&tokenModel.Status,
			&tokenModel.AllowedActions,
			&tokenModel.AdminGenerated,
			&tokenModel.AdminUserId,
			&tokenModel.AdminReason,
			&tokenModel.AdminNotes,
			&tokenModel.CreatedBy,
			&tokenModel.CreatedAt,
			&tokenModel.LastUsedAt,
			&tokenModel.LastUsedIp,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan PaymentUpdateToken`, err.Error())
			return nil, 0, err
		}
		tokens = append(tokens, tokenModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return tokens, count, nil
}

// UpdateToken updates a payment update token
func (r DunningRepository) UpdateToken(ctx context.Context, token dunning.PaymentUpdateToken) (dunning.PaymentUpdateToken, error) {
	tx := r.getTransactionFromContext(ctx)

	tokenDataJson, _ := json.Marshal(token.TokenData)
	allowedActionsJson, _ := json.Marshal(token.AllowedActions)

	query := `UPDATE payment_update_tokens
	SET 
		used_count = $3,
		status = $4,
		last_used_at = $5,
		last_used_ip = $6,
		token_data = $7,
		allowed_actions = $8
	WHERE org_id = $1 AND token_id = $2
	RETURNING 
		org_id, token_id, subscription_id, customer_id, dunning_campaign_id,
		token_data, signature,
		expires_at, max_uses, used_count, status,
		allowed_actions,
		admin_generated, admin_user_id, admin_reason, admin_notes,
		created_by, created_at, last_used_at, last_used_ip`

	var tokenModel models.PaymentUpdateToken
	err := tx.QueryRow(ctx, query,
		token.OrgId,
		token.TokenId,
		token.UsedCount,
		string(token.Status),
		token.LastUsedAt,
		token.LastUsedIp,
		tokenDataJson,
		allowedActionsJson,
	).Scan(
		&tokenModel.OrgId,
		&tokenModel.TokenId,
		&tokenModel.SubscriptionId,
		&tokenModel.CustomerId,
		&tokenModel.DunningCampaignId,
		&tokenModel.TokenData,
		&tokenModel.Signature,
		&tokenModel.ExpiresAt,
		&tokenModel.MaxUses,
		&tokenModel.UsedCount,
		&tokenModel.Status,
		&tokenModel.AllowedActions,
		&tokenModel.AdminGenerated,
		&tokenModel.AdminUserId,
		&tokenModel.AdminReason,
		&tokenModel.AdminNotes,
		&tokenModel.CreatedBy,
		&tokenModel.CreatedAt,
		&tokenModel.LastUsedAt,
		&tokenModel.LastUsedIp,
	)

	if err != nil {
		r.logger.Error(`failed to update PaymentUpdateToken`, err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	return tokenModel.ToEntity(), nil
}

// Configuration operations

// CreateConfiguration creates a new dunning configuration
func (r DunningRepository) CreateConfiguration(ctx context.Context, config dunning.DunningConfiguration) (dunning.DunningConfiguration, error) {
	tx := r.getTransactionFromContext(ctx)

	targetRulesJson, _ := json.Marshal(config.TargetRules)
	configJson, _ := json.Marshal(config.Config)

	query := `INSERT INTO dunning_configurations (
		org_id, id, name, description, priority,
		applies_to, target_rules, config,
		status, is_ab_test, ab_test_percentage,
		created_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8,
		$9, $10, $11,
		$12, NOW(), NOW()
	) RETURNING 
		org_id, id, name, description, priority,
		applies_to, target_rules, config,
		status, is_ab_test, ab_test_percentage,
		created_by, created_at, updated_at`

	var configModel models.DunningConfiguration
	err := tx.QueryRow(ctx, query,
		config.OrgId,
		config.Id,
		config.Name,
		config.Description,
		config.Priority,
		string(config.AppliesTo),
		targetRulesJson,
		configJson,
		string(config.Status),
		config.IsAbTest,
		config.AbTestPercentage,
		config.CreatedBy,
	).Scan(
		&configModel.OrgId,
		&configModel.Id,
		&configModel.Name,
		&configModel.Description,
		&configModel.Priority,
		&configModel.AppliesTo,
		&configModel.TargetRules,
		&configModel.Config,
		&configModel.Status,
		&configModel.IsAbTest,
		&configModel.AbTestPercentage,
		&configModel.CreatedBy,
		&configModel.CreatedAt,
		&configModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to create DunningConfiguration`, err.Error())
		return dunning.DunningConfiguration{}, err
	}

	return configModel.ToEntity(), nil
}

// FindConfigurationById finds a dunning configuration by ID
func (r DunningRepository) FindConfigurationById(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, name, description, priority,
		applies_to, target_rules, config,
		status, is_ab_test, ab_test_percentage,
		created_by, created_at, updated_at
	FROM dunning_configurations
	WHERE org_id = $1 AND id = $2`

	var configModel models.DunningConfiguration
	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&configModel.OrgId,
		&configModel.Id,
		&configModel.Name,
		&configModel.Description,
		&configModel.Priority,
		&configModel.AppliesTo,
		&configModel.TargetRules,
		&configModel.Config,
		&configModel.Status,
		&configModel.IsAbTest,
		&configModel.AbTestPercentage,
		&configModel.CreatedBy,
		&configModel.CreatedAt,
		&configModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find DunningConfiguration by id`, err.Error())
		return dunning.DunningConfiguration{}, err
	}

	return configModel.ToEntity(), nil
}

// FindConfigurations finds all dunning configurations for an organization with pagination
func (r DunningRepository) FindConfigurations(ctx context.Context, orgId string, pagination entities.Pagination) ([]dunning.DunningConfiguration, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", pagination.SortDirection, pagination.SortBy)

	var configs = make([]dunning.DunningConfiguration, 0)
	var count int

	query := `SELECT 
		org_id, id, name, description, priority,
		applies_to, target_rules, config,
		status, is_ab_test, ab_test_percentage,
		created_by, created_at, updated_at,
		count(*) OVER()
	FROM dunning_configurations
	WHERE org_id = $1
	ORDER BY priority DESC, created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := tx.Query(ctx, query, orgId, pagination.Limit, pagination.Offset)
	if err != nil {
		r.logger.Error(`failed to find DunningConfigurations`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var configModel models.DunningConfiguration
		err := rows.Scan(
			&configModel.OrgId,
			&configModel.Id,
			&configModel.Name,
			&configModel.Description,
			&configModel.Priority,
			&configModel.AppliesTo,
			&configModel.TargetRules,
			&configModel.Config,
			&configModel.Status,
			&configModel.IsAbTest,
			&configModel.AbTestPercentage,
			&configModel.CreatedBy,
			&configModel.CreatedAt,
			&configModel.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan DunningConfiguration`, err.Error())
			return nil, 0, err
		}
		configs = append(configs, configModel.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return configs, count, nil
}

// UpdateConfiguration updates a dunning configuration
func (r DunningRepository) UpdateConfiguration(ctx context.Context, config dunning.DunningConfiguration) (dunning.DunningConfiguration, error) {
	tx := r.getTransactionFromContext(ctx)

	targetRulesJson, _ := json.Marshal(config.TargetRules)
	configJson, _ := json.Marshal(config.Config)

	query := `UPDATE dunning_configurations
	SET 
		name = $3,
		description = $4,
		priority = $5,
		applies_to = $6,
		target_rules = $7,
		config = $8,
		status = $9,
		is_ab_test = $10,
		ab_test_percentage = $11,
		updated_at = NOW()
	WHERE org_id = $1 AND id = $2
	RETURNING 
		org_id, id, name, description, priority,
		applies_to, target_rules, config,
		status, is_ab_test, ab_test_percentage,
		created_by, created_at, updated_at`

	var configModel models.DunningConfiguration
	err := tx.QueryRow(ctx, query,
		config.OrgId,
		config.Id,
		config.Name,
		config.Description,
		config.Priority,
		string(config.AppliesTo),
		targetRulesJson,
		configJson,
		string(config.Status),
		config.IsAbTest,
		config.AbTestPercentage,
	).Scan(
		&configModel.OrgId,
		&configModel.Id,
		&configModel.Name,
		&configModel.Description,
		&configModel.Priority,
		&configModel.AppliesTo,
		&configModel.TargetRules,
		&configModel.Config,
		&configModel.Status,
		&configModel.IsAbTest,
		&configModel.AbTestPercentage,
		&configModel.CreatedBy,
		&configModel.CreatedAt,
		&configModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to update DunningConfiguration`, err.Error())
		return dunning.DunningConfiguration{}, err
	}

	return configModel.ToEntity(), nil
}

// Customer dunning history operations

// GetCustomerDunningHistory gets a customer's dunning history
func (r DunningRepository) GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, customer_id, 
		total_dunning_campaigns, successful_recoveries, failed_campaigns,
		total_amount_at_risk, total_amount_recovered, total_amount_lost,
		avg_recovery_time_hours, preferred_recovery_method, most_responsive_channel,
		payment_reliability_score, dunning_risk_tier,
		first_dunning_at, last_dunning_at, last_recovery_at,
		updated_at
	FROM customer_dunning_history
	WHERE org_id = $1 AND customer_id = $2`

	var historyModel models.CustomerDunningHistory
	err := tx.QueryRow(ctx, query, orgId, customerId).Scan(
		&historyModel.OrgId,
		&historyModel.CustomerId,
		&historyModel.TotalDunningCampaigns,
		&historyModel.SuccessfulRecoveries,
		&historyModel.FailedCampaigns,
		&historyModel.TotalAmountAtRisk,
		&historyModel.TotalAmountRecovered,
		&historyModel.TotalAmountLost,
		&historyModel.AvgRecoveryTimeHours,
		&historyModel.PreferredRecoveryMethod,
		&historyModel.MostResponsiveChannel,
		&historyModel.PaymentReliabilityScore,
		&historyModel.DunningRiskTier,
		&historyModel.FirstDunningAt,
		&historyModel.LastDunningAt,
		&historyModel.LastRecoveryAt,
		&historyModel.UpdatedAt,
	)

	if err != nil {
		// If no history exists yet, return an empty history object with the customer ID
		if errors.Is(err, pgx.ErrNoRows) {
			return dunning.CustomerDunningHistory{
				OrgId:      orgId,
				CustomerId: customerId,
			}, nil
		}
		r.logger.Error(`failed to get CustomerDunningHistory`, err.Error())
		return dunning.CustomerDunningHistory{}, err
	}

	return historyModel.ToEntity(), nil
}

// UpdateCustomerDunningHistory updates a customer's dunning history
func (r DunningRepository) UpdateCustomerDunningHistory(ctx context.Context, history dunning.CustomerDunningHistory) (dunning.CustomerDunningHistory, error) {
	tx := r.getTransactionFromContext(ctx)

	// Check if the history record exists
	existingHistory, err := r.GetCustomerDunningHistory(ctx, history.OrgId, history.CustomerId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		r.logger.Error(`failed to check if CustomerDunningHistory exists`, err.Error())
		return dunning.CustomerDunningHistory{}, err
	}

	// If the history doesn't exist, create it
	if existingHistory.CustomerId == "" || errors.Is(err, pgx.ErrNoRows) {
		query := `INSERT INTO customer_dunning_history (
			org_id, customer_id,
			total_dunning_campaigns, successful_recoveries, failed_campaigns,
			total_amount_at_risk, total_amount_recovered, total_amount_lost,
			avg_recovery_time_hours, preferred_recovery_method, most_responsive_channel,
			payment_reliability_score, dunning_risk_tier,
			first_dunning_at, last_dunning_at, last_recovery_at,
			updated_at
		) VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15, $16,
			NOW()
		) RETURNING 
			org_id, customer_id,
			total_dunning_campaigns, successful_recoveries, failed_campaigns,
			total_amount_at_risk, total_amount_recovered, total_amount_lost,
			avg_recovery_time_hours, preferred_recovery_method, most_responsive_channel,
			payment_reliability_score, dunning_risk_tier,
			first_dunning_at, last_dunning_at, last_recovery_at,
			updated_at`

		var historyModel models.CustomerDunningHistory
		err = tx.QueryRow(ctx, query,
			history.OrgId,
			history.CustomerId,
			history.TotalDunningCampaigns,
			history.SuccessfulRecoveries,
			history.FailedCampaigns,
			history.TotalAmountAtRisk,
			history.TotalAmountRecovered,
			history.TotalAmountLost,
			history.AvgRecoveryTimeHours,
			history.PreferredRecoveryMethod,
			string(history.MostResponsiveChannel),
			history.PaymentReliabilityScore,
			history.DunningRiskTier,
			history.FirstDunningAt,
			history.LastDunningAt,
			history.LastRecoveryAt,
		).Scan(
			&historyModel.OrgId,
			&historyModel.CustomerId,
			&historyModel.TotalDunningCampaigns,
			&historyModel.SuccessfulRecoveries,
			&historyModel.FailedCampaigns,
			&historyModel.TotalAmountAtRisk,
			&historyModel.TotalAmountRecovered,
			&historyModel.TotalAmountLost,
			&historyModel.AvgRecoveryTimeHours,
			&historyModel.PreferredRecoveryMethod,
			&historyModel.MostResponsiveChannel,
			&historyModel.PaymentReliabilityScore,
			&historyModel.DunningRiskTier,
			&historyModel.FirstDunningAt,
			&historyModel.LastDunningAt,
			&historyModel.LastRecoveryAt,
			&historyModel.UpdatedAt,
		)

		if err != nil {
			r.logger.Error(`failed to create CustomerDunningHistory`, err.Error())
			return dunning.CustomerDunningHistory{}, err
		}

		return historyModel.ToEntity(), nil
	}

	// Otherwise, update the existing record
	query := `UPDATE customer_dunning_history
	SET 
		total_dunning_campaigns = $3,
		successful_recoveries = $4,
		failed_campaigns = $5,
		total_amount_at_risk = $6,
		total_amount_recovered = $7,
		total_amount_lost = $8,
		avg_recovery_time_hours = $9,
		preferred_recovery_method = $10,
		most_responsive_channel = $11,
		payment_reliability_score = $12,
		dunning_risk_tier = $13,
		first_dunning_at = $14,
		last_dunning_at = $15,
		last_recovery_at = $16,
		updated_at = NOW()
	WHERE org_id = $1 AND customer_id = $2
	RETURNING 
		org_id, customer_id,
		total_dunning_campaigns, successful_recoveries, failed_campaigns,
		total_amount_at_risk, total_amount_recovered, total_amount_lost,
		avg_recovery_time_hours, preferred_recovery_method, most_responsive_channel,
		payment_reliability_score, dunning_risk_tier,
		first_dunning_at, last_dunning_at, last_recovery_at,
		updated_at`

	var historyModel models.CustomerDunningHistory
	err = tx.QueryRow(ctx, query,
		history.OrgId,
		history.CustomerId,
		history.TotalDunningCampaigns,
		history.SuccessfulRecoveries,
		history.FailedCampaigns,
		history.TotalAmountAtRisk,
		history.TotalAmountRecovered,
		history.TotalAmountLost,
		history.AvgRecoveryTimeHours,
		history.PreferredRecoveryMethod,
		string(history.MostResponsiveChannel),
		history.PaymentReliabilityScore,
		history.DunningRiskTier,
		history.FirstDunningAt,
		history.LastDunningAt,
		history.LastRecoveryAt,
	).Scan(
		&historyModel.OrgId,
		&historyModel.CustomerId,
		&historyModel.TotalDunningCampaigns,
		&historyModel.SuccessfulRecoveries,
		&historyModel.FailedCampaigns,
		&historyModel.TotalAmountAtRisk,
		&historyModel.TotalAmountRecovered,
		&historyModel.TotalAmountLost,
		&historyModel.AvgRecoveryTimeHours,
		&historyModel.PreferredRecoveryMethod,
		&historyModel.MostResponsiveChannel,
		&historyModel.PaymentReliabilityScore,
		&historyModel.DunningRiskTier,
		&historyModel.FirstDunningAt,
		&historyModel.LastDunningAt,
		&historyModel.LastRecoveryAt,
		&historyModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to update CustomerDunningHistory`, err.Error())
		return dunning.CustomerDunningHistory{}, err
	}

	return historyModel.ToEntity(), nil
}
