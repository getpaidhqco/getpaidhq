package postgres

import (
	"context"
	"fmt"
	"time"
	
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type UsageProcessingStatusRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewUsageProcessingStatusRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageProcessingStatusRepository {
	pgDatabase, ok := usageDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *PgDatabase")
	}
	return &UsageProcessingStatusRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *UsageProcessingStatusRepository) UpsertProcessingStatus(ctx context.Context, status entities.UsageProcessingStatus) error {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		INSERT INTO usage_processing_status (
			org_id, subscription_item_id, billing_period, 
			total_quantity, total_amount, event_count,
			processed, processed_at, invoice_id,
			first_event_time, last_event_time, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (org_id, subscription_item_id, billing_period) DO UPDATE SET
			total_quantity = EXCLUDED.total_quantity,
			total_amount = EXCLUDED.total_amount,
			event_count = EXCLUDED.event_count,
			processed = EXCLUDED.processed,
			processed_at = EXCLUDED.processed_at,
			invoice_id = EXCLUDED.invoice_id,
			first_event_time = EXCLUDED.first_event_time,
			last_event_time = EXCLUDED.last_event_time,
			last_updated = EXCLUDED.last_updated
	`
	
	_, err := tx.Exec(ctx, query,
		status.OrgID,
		status.SubscriptionItemID,
		status.BillingPeriod,
		status.TotalQuantity,
		status.TotalAmount,
		status.EventCount,
		status.Processed,
		status.ProcessedAt,
		status.InvoiceID,
		status.FirstEventTime,
		status.LastEventTime,
		status.LastUpdated,
	)
	
	if err != nil {
		return fmt.Errorf("failed to upsert usage processing status: %w", err)
	}
	
	return nil
}

func (r *UsageProcessingStatusRepository) GetProcessingStatus(ctx context.Context, orgID, subscriptionItemID, billingPeriod string) (entities.UsageProcessingStatus, error) {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		SELECT org_id, subscription_item_id, billing_period, 
			   total_quantity, total_amount, event_count,
			   processed, processed_at, invoice_id,
			   first_event_time, last_event_time, last_updated
		FROM usage_processing_status
		WHERE org_id = $1 AND subscription_item_id = $2 AND billing_period = $3
	`
	
	var status entities.UsageProcessingStatus
	
	err := tx.QueryRow(ctx, query, orgID, subscriptionItemID, billingPeriod).Scan(
		&status.OrgID,
		&status.SubscriptionItemID,
		&status.BillingPeriod,
		&status.TotalQuantity,
		&status.TotalAmount,
		&status.EventCount,
		&status.Processed,
		&status.ProcessedAt,
		&status.InvoiceID,
		&status.FirstEventTime,
		&status.LastEventTime,
		&status.LastUpdated,
	)
	
	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.UsageProcessingStatus{}, fmt.Errorf("usage processing status not found")
		}
		return entities.UsageProcessingStatus{}, fmt.Errorf("failed to get usage processing status: %w", err)
	}
	
	return status, nil
}

func (r *UsageProcessingStatusRepository) GetUnprocessedUsage(ctx context.Context, orgID, billingPeriod string) ([]entities.UsageProcessingStatus, error) {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		SELECT org_id, subscription_item_id, billing_period, 
			   total_quantity, total_amount, event_count,
			   processed, processed_at, invoice_id,
			   first_event_time, last_event_time, last_updated
		FROM usage_processing_status
		WHERE org_id = $1 AND billing_period = $2 AND processed = false
		ORDER BY subscription_item_id
	`
	
	rows, err := tx.Query(ctx, query, orgID, billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed usage: %w", err)
	}
	defer rows.Close()
	
	var statuses []entities.UsageProcessingStatus
	for rows.Next() {
		var status entities.UsageProcessingStatus
		
		err := rows.Scan(
			&status.OrgID,
			&status.SubscriptionItemID,
			&status.BillingPeriod,
			&status.TotalQuantity,
			&status.TotalAmount,
			&status.EventCount,
			&status.Processed,
			&status.ProcessedAt,
			&status.InvoiceID,
			&status.FirstEventTime,
			&status.LastEventTime,
			&status.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage processing status: %w", err)
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}

func (r *UsageProcessingStatusRepository) MarkAsProcessed(ctx context.Context, orgID, subscriptionItemID, billingPeriod, invoiceID string) error {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		UPDATE usage_processing_status
		SET processed = true, processed_at = $4, invoice_id = $5, last_updated = $6
		WHERE org_id = $1 AND subscription_item_id = $2 AND billing_period = $3
	`
	
	now := time.Now()
	result, err := tx.Exec(ctx, query, orgID, subscriptionItemID, billingPeriod, now, invoiceID, now)
	if err != nil {
		return fmt.Errorf("failed to mark usage as processed: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("usage processing status not found")
	}
	
	return nil
}

func (r *UsageProcessingStatusRepository) GetByInvoiceID(ctx context.Context, invoiceID string) ([]entities.UsageProcessingStatus, error) {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		SELECT org_id, subscription_item_id, billing_period, 
			   total_quantity, total_amount, event_count,
			   processed, processed_at, invoice_id,
			   first_event_time, last_event_time, last_updated
		FROM usage_processing_status
		WHERE invoice_id = $1
		ORDER BY org_id, subscription_item_id
	`
	
	rows, err := tx.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage by invoice: %w", err)
	}
	defer rows.Close()
	
	var statuses []entities.UsageProcessingStatus
	for rows.Next() {
		var status entities.UsageProcessingStatus
		
		err := rows.Scan(
			&status.OrgID,
			&status.SubscriptionItemID,
			&status.BillingPeriod,
			&status.TotalQuantity,
			&status.TotalAmount,
			&status.EventCount,
			&status.Processed,
			&status.ProcessedAt,
			&status.InvoiceID,
			&status.FirstEventTime,
			&status.LastEventTime,
			&status.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage processing status: %w", err)
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}