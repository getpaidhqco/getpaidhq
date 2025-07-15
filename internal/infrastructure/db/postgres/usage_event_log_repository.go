package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type UsageEventLogRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewUsageEventLogRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageEventLogRepository {
	pgDatabase, ok := usageDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *PgDatabase")
	}
	return &UsageEventLogRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *UsageEventLogRepository) Create(ctx context.Context, log entities.UsageEventLog) error {
	tx := r.getTransactionFromContext(ctx)

	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO usage_event_log (
			id, timestamp, org_id, event_type, 
			subscription_id, subscription_item_id, customer_id, invoice_id,
			amount, quantity, event_count, billing_period,
			triggered_by, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = tx.Exec(ctx, query,
		log.ID,
		log.Timestamp,
		log.OrgID,
		log.EventType,
		log.SubscriptionID,
		log.SubscriptionItemID,
		log.CustomerID,
		log.InvoiceID,
		log.Amount,
		log.Quantity,
		log.EventCount,
		log.BillingPeriod,
		log.TriggeredBy,
		log.Reason,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert usage event log: %w", err)
	}

	return nil
}

func (r *UsageEventLogRepository) FindByOrg(ctx context.Context, orgID string, limit, offset int) ([]entities.UsageEventLog, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT id, timestamp, org_id, event_type, 
			   subscription_id, subscription_item_id, customer_id, invoice_id,
			   amount, quantity, event_count, billing_period,
			   triggered_by, reason, metadata
		FROM usage_event_log
		WHERE org_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	return r.queryLogs(ctx, tx, query, orgID, limit, offset)
}

func (r *UsageEventLogRepository) FindByEventType(ctx context.Context, orgID, eventType string, limit, offset int) ([]entities.UsageEventLog, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT id, timestamp, org_id, event_type, 
			   subscription_id, subscription_item_id, customer_id, invoice_id,
			   amount, quantity, event_count, billing_period,
			   triggered_by, reason, metadata
		FROM usage_event_log
		WHERE org_id = $1 AND event_type = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`

	return r.queryLogs(ctx, tx, query, orgID, eventType, limit, offset)
}

func (r *UsageEventLogRepository) FindBySubscription(ctx context.Context, orgID, subscriptionID string, limit, offset int) ([]entities.UsageEventLog, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT id, timestamp, org_id, event_type, 
			   subscription_id, subscription_item_id, customer_id, invoice_id,
			   amount, quantity, event_count, billing_period,
			   triggered_by, reason, metadata
		FROM usage_event_log
		WHERE org_id = $1 AND subscription_id = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`

	return r.queryLogs(ctx, tx, query, orgID, subscriptionID, limit, offset)
}

func (r *UsageEventLogRepository) FindByInvoice(ctx context.Context, orgID, invoiceID string) ([]entities.UsageEventLog, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT id, timestamp, org_id, event_type, 
			   subscription_id, subscription_item_id, customer_id, invoice_id,
			   amount, quantity, event_count, billing_period,
			   triggered_by, reason, metadata
		FROM usage_event_log
		WHERE org_id = $1 AND invoice_id = $2
		ORDER BY timestamp DESC
	`

	return r.queryLogs(ctx, tx, query, orgID, invoiceID)
}

// Helper method to avoid code duplication in query methods
func (r *UsageEventLogRepository) queryLogs(ctx context.Context, tx QueryRower, query string, args ...interface{}) ([]entities.UsageEventLog, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event logs: %w", err)
	}
	defer rows.Close()

	var logs []entities.UsageEventLog
	for rows.Next() {
		var log entities.UsageEventLog
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.OrgID,
			&log.EventType,
			&log.SubscriptionID,
			&log.SubscriptionItemID,
			&log.CustomerID,
			&log.InvoiceID,
			&log.Amount,
			&log.Quantity,
			&log.EventCount,
			&log.BillingPeriod,
			&log.TriggeredBy,
			&log.Reason,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage event log: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
				r.logger.Warn("Failed to unmarshal metadata", "error", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}
