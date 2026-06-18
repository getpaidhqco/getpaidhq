package postgrespgx

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type DunningRepo struct {
	pool *pgxpool.Pool
}

func NewDunningRepo(pool *pgxpool.Pool) port.DunningRepository {
	return &DunningRepo{pool: pool}
}

// ---- Campaigns ----

func (r *DunningRepo) CreateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	row := dunningCampaignRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO dunning_campaigns (`+dunningCampaignColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
		row.OrgId, row.Id, row.SubscriptionId, row.CustomerId, row.WorkflowId, row.WorkflowRunId, row.ParentWorkflowId,
		row.Status, row.FailedAmount, row.Currency, row.InitialFailureReason,
		row.TotalAttempts, row.ImmediateAttempts, row.ProgressiveAttempts,
		row.StartedAt, row.LastAttemptAt, row.NextAttemptAt, row.CompletedAt,
		row.RecoveryMethod, row.RecoveredAmount, row.RecoveredAt, row.FinalFailureReason,
		row.ConfigSnapshot, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCampaignById(ctx context.Context, orgId, id string) (domain.DunningCampaign, error) {
	q := dbFromCtx(ctx, r.pool)
	var row dunningCampaignRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+dunningCampaignColumns+` FROM dunning_campaigns WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.DunningCampaign{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM dunning_campaigns WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningCampaignColumns+` FROM dunning_campaigns WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectCampaigns(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) FindCampaignsBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM dunning_campaigns WHERE org_id = $1 AND subscription_id = $2`, orgId, subscriptionId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningCampaignColumns+` FROM dunning_campaigns WHERE org_id = $1 AND subscription_id = $2`+paginationClause(p),
		orgId, subscriptionId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectCampaigns(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) FindCampaignsByCustomerId(ctx context.Context, orgId, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM dunning_campaigns WHERE org_id = $1 AND customer_id = $2`, orgId, customerId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningCampaignColumns+` FROM dunning_campaigns WHERE org_id = $1 AND customer_id = $2`+paginationClause(p),
		orgId, customerId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectCampaigns(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) FindActiveCampaignForSubscription(ctx context.Context, orgId, subscriptionId string) (domain.DunningCampaign, error) {
	q := dbFromCtx(ctx, r.pool)
	var row dunningCampaignRow
	err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+dunningCampaignColumns+` FROM dunning_campaigns
		 WHERE org_id = $1 AND subscription_id = $2 AND status = ANY($3)
		 ORDER BY created_at DESC LIMIT 1`,
		orgId, subscriptionId, []string{string(domain.DunningStatusActive), string(domain.DunningStatusPaused)}))
	if err != nil {
		return domain.DunningCampaign{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) UpdateCampaign(ctx context.Context, c domain.DunningCampaign) (domain.DunningCampaign, error) {
	row := dunningCampaignRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE dunning_campaigns SET
		    subscription_id=$3, customer_id=$4, workflow_id=$5, workflow_run_id=$6, parent_workflow_id=$7,
		    status=$8, failed_amount=$9, currency=$10, initial_failure_reason=$11,
		    total_attempts=$12, immediate_attempts=$13, progressive_attempts=$14,
		    started_at=$15, last_attempt_at=$16, next_attempt_at=$17, completed_at=$18,
		    recovery_method=$19, recovered_amount=$20, recovered_at=$21, final_failure_reason=$22,
		    config_snapshot=$23, metadata=$24, updated_at=$25
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.SubscriptionId, row.CustomerId, row.WorkflowId, row.WorkflowRunId, row.ParentWorkflowId,
		row.Status, row.FailedAmount, row.Currency, row.InitialFailureReason,
		row.TotalAttempts, row.ImmediateAttempts, row.ProgressiveAttempts,
		row.StartedAt, row.LastAttemptAt, row.NextAttemptAt, row.CompletedAt,
		row.RecoveryMethod, row.RecoveredAmount, row.RecoveredAt, row.FinalFailureReason,
		row.ConfigSnapshot, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	return r.FindCampaignById(ctx, c.OrgId, c.Id)
}

// ---- Attempts ----

func (r *DunningRepo) CreateAttempt(ctx context.Context, a domain.DunningAttempt) (domain.DunningAttempt, error) {
	row := dunningAttemptRowFromDomain(a)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO dunning_attempts (`+dunningAttemptColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		row.OrgId, row.Id, row.DunningCampaignId, row.SubscriptionId, row.AttemptNumber, row.AttemptType,
		row.Amount, row.Currency, row.PaymentMethodId, row.Status, row.FailureReason, row.FailureCode,
		row.ProcessorResponse, row.ProcessingTimeMs, row.AttemptedAt, row.CompletedAt, row.TriggeredBy, row.Metadata, row.CreatedAt)
	if err != nil {
		return domain.DunningAttempt{}, err
	}
	return r.FindAttemptById(ctx, a.OrgId, a.Id)
}

func (r *DunningRepo) FindAttemptById(ctx context.Context, orgId, id string) (domain.DunningAttempt, error) {
	q := dbFromCtx(ctx, r.pool)
	var row dunningAttemptRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+dunningAttemptColumns+` FROM dunning_attempts WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.DunningAttempt{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindAttemptsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM dunning_attempts WHERE org_id = $1 AND dunning_campaign_id = $2`, orgId, campaignId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningAttemptColumns+` FROM dunning_attempts WHERE org_id = $1 AND dunning_campaign_id = $2`+paginationClause(p),
		orgId, campaignId)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.DunningAttempt
	for rows.Next() {
		var row dunningAttemptRow
		if err := row.scanInto(rows); err != nil {
			return nil, 0, err
		}
		out = append(out, row.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

// ---- Communications ----

func (r *DunningRepo) CreateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	row := dunningCommunicationRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO dunning_communications (`+dunningCommunicationColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		row.OrgId, row.Id, row.DunningCampaignId, row.CustomerId, row.Channel, row.TemplateId, row.AttemptNumber,
		row.Subject, row.ContentPreview, row.PersonalizationData,
		row.SentAt, row.DeliveredAt, row.OpenedAt, row.ClickedAt, row.BouncedAt,
		row.Provider, row.ProviderMessageId, row.ProviderResponse, row.Status, row.FailureReason, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindCommunicationById(ctx context.Context, orgId, id string) (domain.DunningCommunication, error) {
	q := dbFromCtx(ctx, r.pool)
	var row dunningCommunicationRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+dunningCommunicationColumns+` FROM dunning_communications WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.DunningCommunication{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindCommunicationsByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM dunning_communications WHERE org_id = $1 AND dunning_campaign_id = $2`, orgId, campaignId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningCommunicationColumns+` FROM dunning_communications WHERE org_id = $1 AND dunning_campaign_id = $2`+paginationClause(p),
		orgId, campaignId)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.DunningCommunication
	for rows.Next() {
		var row dunningCommunicationRow
		if err := row.scanInto(rows); err != nil {
			return nil, 0, err
		}
		out = append(out, row.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) UpdateCommunication(ctx context.Context, c domain.DunningCommunication) (domain.DunningCommunication, error) {
	row := dunningCommunicationRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE dunning_communications SET
		    dunning_campaign_id=$3, customer_id=$4, channel=$5, template_id=$6, attempt_number=$7,
		    subject=$8, content_preview=$9, personalization_data=$10,
		    sent_at=$11, delivered_at=$12, opened_at=$13, clicked_at=$14, bounced_at=$15,
		    provider=$16, provider_message_id=$17, provider_response=$18, status=$19, failure_reason=$20, updated_at=$21
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.DunningCampaignId, row.CustomerId, row.Channel, row.TemplateId, row.AttemptNumber,
		row.Subject, row.ContentPreview, row.PersonalizationData,
		row.SentAt, row.DeliveredAt, row.OpenedAt, row.ClickedAt, row.BouncedAt,
		row.Provider, row.ProviderMessageId, row.ProviderResponse, row.Status, row.FailureReason, row.UpdatedAt)
	if err != nil {
		return domain.DunningCommunication{}, err
	}
	return r.FindCommunicationById(ctx, c.OrgId, c.Id)
}

// ---- Tokens ----

func (r *DunningRepo) CreateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	row := paymentUpdateTokenRowFromDomain(t)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO payment_update_tokens (`+paymentUpdateTokenColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		row.OrgId, row.TokenId, row.SubscriptionId, row.CustomerId, row.DunningCampaignId,
		row.TokenData, row.Signature, row.ExpiresAt, row.MaxUses, row.UsedCount, row.Status, row.AllowedActions,
		row.AdminGenerated, row.AdminUserId, row.AdminReason, row.AdminNotes, row.CreatedBy, row.CreatedAt, row.LastUsedAt, row.LastUsedIp)
	if err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

func (r *DunningRepo) FindTokenById(ctx context.Context, orgId, tokenId string) (domain.PaymentUpdateToken, error) {
	q := dbFromCtx(ctx, r.pool)
	var row paymentUpdateTokenRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+paymentUpdateTokenColumns+` FROM payment_update_tokens WHERE org_id = $1 AND token_id = $2`, orgId, tokenId)); err != nil {
		return domain.PaymentUpdateToken{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindTokensBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM payment_update_tokens WHERE org_id = $1 AND subscription_id = $2`, orgId, subscriptionId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+paymentUpdateTokenColumns+` FROM payment_update_tokens WHERE org_id = $1 AND subscription_id = $2`+paginationClause(p),
		orgId, subscriptionId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectTokens(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) FindTokensByCampaignId(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM payment_update_tokens WHERE org_id = $1 AND dunning_campaign_id = $2`, orgId, campaignId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+paymentUpdateTokenColumns+` FROM payment_update_tokens WHERE org_id = $1 AND dunning_campaign_id = $2`+paginationClause(p),
		orgId, campaignId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectTokens(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) UpdateToken(ctx context.Context, t domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error) {
	row := paymentUpdateTokenRowFromDomain(t)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE payment_update_tokens SET
		    subscription_id=$3, customer_id=$4, dunning_campaign_id=$5,
		    token_data=$6, signature=$7, expires_at=$8, max_uses=$9, used_count=$10, status=$11, allowed_actions=$12,
		    admin_generated=$13, admin_user_id=$14, admin_reason=$15, admin_notes=$16, created_by=$17, last_used_at=$18, last_used_ip=$19
		 WHERE org_id=$1 AND token_id=$2`,
		row.OrgId, row.TokenId, row.SubscriptionId, row.CustomerId, row.DunningCampaignId,
		row.TokenData, row.Signature, row.ExpiresAt, row.MaxUses, row.UsedCount, row.Status, row.AllowedActions,
		row.AdminGenerated, row.AdminUserId, row.AdminReason, row.AdminNotes, row.CreatedBy, row.LastUsedAt, row.LastUsedIp)
	if err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	return r.FindTokenById(ctx, t.OrgId, t.TokenId)
}

// ---- Configurations ----

func (r *DunningRepo) CreateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	row := dunningConfigurationRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO dunning_configurations (`+dunningConfigurationColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		row.OrgId, row.Id, row.Name, row.Description, row.Priority, row.AppliesTo, row.TargetRules, row.Config,
		row.Status, row.IsAbTest, row.AbTestPercentage, row.CreatedBy, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

func (r *DunningRepo) FindConfigurationById(ctx context.Context, orgId, id string) (domain.DunningConfiguration, error) {
	q := dbFromCtx(ctx, r.pool)
	var row dunningConfigurationRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+dunningConfigurationColumns+` FROM dunning_configurations WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.DunningConfiguration{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) FindConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM dunning_configurations WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+dunningConfigurationColumns+` FROM dunning_configurations WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectConfigurations(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *DunningRepo) FindConfigurationsByPriority(ctx context.Context, orgId string) ([]domain.DunningConfiguration, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+dunningConfigurationColumns+` FROM dunning_configurations
		 WHERE org_id = $1 AND status = $2 ORDER BY priority DESC`,
		orgId, string(domain.ConfigStatusActive))
	if err != nil {
		return nil, err
	}
	return collectConfigurations(rows)
}

func (r *DunningRepo) UpdateConfiguration(ctx context.Context, c domain.DunningConfiguration) (domain.DunningConfiguration, error) {
	row := dunningConfigurationRowFromDomain(c)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE dunning_configurations SET
		    name=$3, description=$4, priority=$5, applies_to=$6, target_rules=$7, config=$8,
		    status=$9, is_ab_test=$10, ab_test_percentage=$11, created_by=$12, updated_at=$13
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Name, row.Description, row.Priority, row.AppliesTo, row.TargetRules, row.Config,
		row.Status, row.IsAbTest, row.AbTestPercentage, row.CreatedBy, row.UpdatedAt)
	if err != nil {
		return domain.DunningConfiguration{}, err
	}
	return r.FindConfigurationById(ctx, c.OrgId, c.Id)
}

// ---- Customer history ----

func (r *DunningRepo) GetCustomerDunningHistory(ctx context.Context, orgId, customerId string) (domain.CustomerDunningHistory, error) {
	q := dbFromCtx(ctx, r.pool)
	var row customerDunningHistoryRow
	err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+customerDunningHistoryColumns+` FROM customer_dunning_history WHERE org_id = $1 AND customer_id = $2`,
		orgId, customerId))
	if errors.Is(err, pgx.ErrNoRows) {
		// Synthesize a zero history rather than propagate "not found" —
		// callers treat the absence of any dunning row as "clean".
		return domain.CustomerDunningHistory{
			OrgId:      orgId,
			CustomerId: customerId,
		}, nil
	}
	if err != nil {
		return domain.CustomerDunningHistory{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DunningRepo) UpsertCustomerDunningHistory(ctx context.Context, h domain.CustomerDunningHistory) (domain.CustomerDunningHistory, error) {
	row := customerDunningHistoryRowFromDomain(h)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO customer_dunning_history (`+customerDunningHistoryColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		 ON CONFLICT (org_id, customer_id) DO UPDATE SET
		    total_dunning_campaigns=EXCLUDED.total_dunning_campaigns,
		    successful_recoveries=EXCLUDED.successful_recoveries,
		    failed_campaigns=EXCLUDED.failed_campaigns,
		    total_amount_at_risk=EXCLUDED.total_amount_at_risk,
		    total_amount_recovered=EXCLUDED.total_amount_recovered,
		    total_amount_lost=EXCLUDED.total_amount_lost,
		    avg_recovery_time_hours=EXCLUDED.avg_recovery_time_hours,
		    preferred_recovery_method=EXCLUDED.preferred_recovery_method,
		    most_responsive_channel=EXCLUDED.most_responsive_channel,
		    payment_reliability_score=EXCLUDED.payment_reliability_score,
		    dunning_risk_tier=EXCLUDED.dunning_risk_tier,
		    first_dunning_at=EXCLUDED.first_dunning_at,
		    last_dunning_at=EXCLUDED.last_dunning_at,
		    last_recovery_at=EXCLUDED.last_recovery_at,
		    updated_at=EXCLUDED.updated_at`,
		row.OrgId, row.CustomerId, row.TotalDunningCampaigns, row.SuccessfulRecoveries, row.FailedCampaigns,
		row.TotalAmountAtRisk, row.TotalAmountRecovered, row.TotalAmountLost,
		row.AvgRecoveryTimeHours, row.PreferredRecoveryMethod, row.MostResponsiveChannel, row.PaymentReliabilityScore, row.DunningRiskTier,
		row.FirstDunningAt, row.LastDunningAt, row.LastRecoveryAt, row.UpdatedAt)
	if err != nil {
		return domain.CustomerDunningHistory{}, err
	}
	return r.GetCustomerDunningHistory(ctx, h.OrgId, h.CustomerId)
}

// ---- collectors ----

func collectCampaigns(rows pgx.Rows) ([]domain.DunningCampaign, error) {
	defer rows.Close()
	var out []domain.DunningCampaign
	for rows.Next() {
		var row dunningCampaignRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}

func collectTokens(rows pgx.Rows) ([]domain.PaymentUpdateToken, error) {
	defer rows.Close()
	var out []domain.PaymentUpdateToken
	for rows.Next() {
		var row paymentUpdateTokenRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}

func collectConfigurations(rows pgx.Rows) ([]domain.DunningConfiguration, error) {
	defer rows.Close()
	var out []domain.DunningConfiguration
	for rows.Next() {
		var row dunningConfigurationRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
