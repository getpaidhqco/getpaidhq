package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type UsageEventRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewUsageEventRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageEventRepository {
	pgDatabase, ok := usageDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *PgDatabase")
	}
	return &UsageEventRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *UsageEventRepository) Create(ctx context.Context, event entities.UsageEvent) error {
	tx := r.getTransactionFromContext(ctx)

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO usage_events (
			time, org_id, subscription_id, subscription_item_id, customer_id,
			usage_type, quantity, transaction_value, calculated_amount,
			reference_id, reference_type, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (time, org_id, subscription_item_id) DO NOTHING
	`

	_, err = tx.Exec(ctx, query,
		event.Time,
		event.OrgID,
		event.SubscriptionID,
		event.SubscriptionItemID,
		event.CustomerID,
		event.UsageType,
		event.Quantity,
		event.TransactionValue,
		event.CalculatedAmount,
		event.ReferenceID,
		event.ReferenceType,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert usage event: %w", err)
	}

	return nil
}

func (r *UsageEventRepository) BatchCreate(ctx context.Context, events []entities.UsageEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx := r.getTransactionFromContext(ctx)

	query := `
		INSERT INTO usage_events (
			time, org_id, subscription_id, subscription_item_id, customer_id,
			usage_type, quantity, transaction_value, calculated_amount,
			reference_id, reference_type, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (time, org_id, subscription_item_id) DO NOTHING
	`

	for i, event := range events {
		metadataJSON, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for event %d: %w", i, err)
		}

		_, err = tx.Exec(ctx, query,
			event.Time,
			event.OrgID,
			event.SubscriptionID,
			event.SubscriptionItemID,
			event.CustomerID,
			event.UsageType,
			event.Quantity,
			event.TransactionValue,
			event.CalculatedAmount,
			event.ReferenceID,
			event.ReferenceType,
			metadataJSON,
		)

		if err != nil {
			return fmt.Errorf("failed to insert batch event %d: %w", i, err)
		}
	}

	return nil
}

func (r *UsageEventRepository) FindByID(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) (entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
			   usage_type, quantity, transaction_value, calculated_amount,
			   reference_id, reference_type, metadata
		FROM usage_events
		WHERE org_id = $1 AND subscription_item_id = $2 AND time = $3
	`

	var event entities.UsageEvent
	var metadataJSON []byte

	err := tx.QueryRow(ctx, query, orgID, subscriptionItemID, eventTime).Scan(
		&event.Time,
		&event.OrgID,
		&event.SubscriptionID,
		&event.SubscriptionItemID,
		&event.CustomerID,
		&event.UsageType,
		&event.Quantity,
		&event.TransactionValue,
		&event.CalculatedAmount,
		&event.ReferenceID,
		&event.ReferenceType,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.UsageEvent{}, fmt.Errorf("usage event not found")
		}
		return entities.UsageEvent{}, fmt.Errorf("failed to find usage event: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			r.logger.Warn("Failed to unmarshal metadata", "error", err)
		}
	}

	return event, nil
}

func (r *UsageEventRepository) FindBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, startTime, endTime time.Time) ([]entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
			   usage_type, quantity, transaction_value, calculated_amount,
			   reference_id, reference_type, metadata
		FROM usage_events
		WHERE org_id = $1 AND subscription_item_id = $2
		  AND time >= $3 AND time < $4
		ORDER BY time DESC
	`

	rows, err := tx.Query(ctx, query, orgID, subscriptionItemID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage events: %w", err)
	}
	defer rows.Close()

	var events []entities.UsageEvent
	for rows.Next() {
		var event entities.UsageEvent
		var metadataJSON []byte

		err := rows.Scan(
			&event.Time,
			&event.OrgID,
			&event.SubscriptionID,
			&event.SubscriptionItemID,
			&event.CustomerID,
			&event.UsageType,
			&event.Quantity,
			&event.TransactionValue,
			&event.CalculatedAmount,
			&event.ReferenceID,
			&event.ReferenceType,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage event: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
				r.logger.Warn("Failed to unmarshal metadata", "error", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

func (r *UsageEventRepository) FindByReferenceID(ctx context.Context, referenceID, referenceType string) (entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
			   usage_type, quantity, transaction_value, calculated_amount,
			   reference_id, reference_type, metadata
		FROM usage_events
		WHERE reference_id = $1 AND reference_type = $2
		LIMIT 1
	`

	var event entities.UsageEvent
	var metadataJSON []byte

	err := tx.QueryRow(ctx, query, referenceID, referenceType).Scan(
		&event.Time,
		&event.OrgID,
		&event.SubscriptionID,
		&event.SubscriptionItemID,
		&event.CustomerID,
		&event.UsageType,
		&event.Quantity,
		&event.TransactionValue,
		&event.CalculatedAmount,
		&event.ReferenceID,
		&event.ReferenceType,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.UsageEvent{}, fmt.Errorf("usage event not found")
		}
		return entities.UsageEvent{}, fmt.Errorf("failed to find usage event by reference: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			r.logger.Warn("Failed to unmarshal metadata", "error", err)
		}
	}

	return event, nil
}

func (r *UsageEventRepository) Delete(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) error {
	tx := r.getTransactionFromContext(ctx)

	query := `
		DELETE FROM usage_events
		WHERE org_id = $1 AND subscription_item_id = $2 AND time = $3
	`

	result, err := tx.Exec(ctx, query, orgID, subscriptionItemID, eventTime)
	if err != nil {
		return fmt.Errorf("failed to delete usage event: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("usage event not found for deletion")
	}

	return nil
}
