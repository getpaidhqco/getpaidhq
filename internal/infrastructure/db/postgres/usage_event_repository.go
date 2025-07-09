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
	"payloop/internal/infrastructure/db/postgres/models"
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

	// Convert entity to model
	model := models.UsageEventFromEntity(event)

	// Marshal JSON fields
	dataJSON, err := json.Marshal(model.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
		INSERT INTO usage_events (
			org_id, id, 
			subscription_id, subscription_item_id, meter_id,
			spec_version, type, event_id, time, source, subject, data,
			received_at
		) VALUES (
			@org_id, @id, 
			@subscription_id, @subscription_item_id, @meter_id,
			@spec_version, @type, @event_id, @time, @source, @subject, @data,
			@received_at
		)
		ON CONFLICT (org_id, id) DO NOTHING
	`

	args := pgx.NamedArgs{
		"org_id":               model.OrgId,
		"id":                   model.Id,
		"subscription_id":      model.SubscriptionId,
		"subscription_item_id": model.SubscriptionItemId,
		"meter_id":             model.MeterId,
		"spec_version":         model.SpecVersion,
		"type":                 model.Type,
		"event_id":             model.EventId,
		"time":                 model.Time,
		"source":               model.Source,
		"subject":              model.Subject,
		"data":                 dataJSON,
		"received_at":          model.ReceivedAt,
	}

	_, err = tx.Exec(ctx, query, args)

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
			org_id, id, 
			subscription_id, subscription_item_id, meter_id,
			spec_version, type, event_id, time, source, subject, data,
			received_at
		) VALUES (
			@org_id, @id, 
			@subscription_id, @subscription_item_id, @meter_id,
			@spec_version, @type, @event_id, @time, @source, @subject, @data,
			@received_at
		)
		ON CONFLICT (org_id, id) DO NOTHING
	`

	for i, event := range events {
		// Convert entity to model
		model := models.UsageEventFromEntity(event)

		// Marshal JSON fields
		dataJSON, err := json.Marshal(model.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal data for event %d: %w", i, err)
		}

		args := pgx.NamedArgs{
			"org_id":               model.OrgId,
			"id":                   model.Id,
			"subscription_id":      model.SubscriptionId,
			"subscription_item_id": model.SubscriptionItemId,
			"meter_id":             model.MeterId,
			"spec_version":         model.SpecVersion,
			"type":                 model.Type,
			"event_id":             model.EventId,
			"time":                 model.Time,
			"source":               model.Source,
			"subject":              model.Subject,
			"data":                 dataJSON,
			"received_at":          model.ReceivedAt,
		}

		_, err = tx.Exec(ctx, query, args)

		if err != nil {
			return fmt.Errorf("failed to insert batch event %d: %w", i, err)
		}
	}

	return nil
}

func (r *UsageEventRepository) FindByID(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) (entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	// Note: The repository interface expects to find by orgID, subscriptionItemID, and time
	// but the schema has a primary key of [orgId, id]. We'll query by the available fields.
	query := `
		SELECT 
			org_id, id, 
			subscription_id, subscription_item_id, meter_id,
			spec_version, type, event_id, time, source, subject, data,
			received_at
		FROM usage_events
		WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND time = @time
		LIMIT 1
	`

	args := pgx.NamedArgs{
		"org_id":               orgID,
		"subscription_item_id": subscriptionItemID,
		"time":                 eventTime,
	}

	var model models.UsageEvent
	var dataJSON []byte

	err := tx.QueryRow(ctx, query, args).Scan(
		&model.OrgId,
		&model.Id,
		&model.SubscriptionId,
		&model.SubscriptionItemId,
		&model.MeterId,
		&model.SpecVersion,
		&model.Type,
		&model.EventId,
		&model.Time,
		&model.Source,
		&model.Subject,
		&dataJSON,
		&model.ReceivedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.UsageEvent{}, fmt.Errorf("usage event not found")
		}
		return entities.UsageEvent{}, fmt.Errorf("failed to find usage event: %w", err)
	}

	// Parse JSON fields
	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &model.Data); err != nil {
			r.logger.Warn("Failed to unmarshal data", "error", err)
		}
	}

	return model.ToEntity(), nil
}

func (r *UsageEventRepository) FindBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, startTime, endTime time.Time) ([]entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `
		SELECT 
			org_id, id, 
			subscription_id, subscription_item_id, meter_id,
			spec_version, type, event_id, time, source, subject, data,
			received_at
		FROM usage_events
		WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id
		  AND time >= @start_time AND time < @end_time
		ORDER BY time DESC
	`

	args := pgx.NamedArgs{
		"org_id":               orgID,
		"subscription_item_id": subscriptionItemID,
		"start_time":           startTime,
		"end_time":             endTime,
	}

	rows, err := tx.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage events: %w", err)
	}
	defer rows.Close()

	var events []entities.UsageEvent
	for rows.Next() {
		var model models.UsageEvent
		var dataJSON []byte

		err := rows.Scan(
			&model.OrgId,
			&model.Id,
			&model.SubscriptionId,
			&model.SubscriptionItemId,
			&model.MeterId,
			&model.SpecVersion,
			&model.Type,
			&model.EventId,
			&model.Time,
			&model.Source,
			&model.Subject,
			&dataJSON,
			&model.ReceivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage event: %w", err)
		}

		// Parse JSON fields
		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &model.Data); err != nil {
				r.logger.Warn("Failed to unmarshal data", "error", err)
			}
		}

		events = append(events, model.ToEntity())
	}

	return events, nil
}

func (r *UsageEventRepository) FindByReferenceID(ctx context.Context, referenceID, referenceType string) (entities.UsageEvent, error) {
	tx := r.getTransactionFromContext(ctx)

	// In the new schema, we'll use event_id field to match the referenceID
	query := `
		SELECT 
			org_id, id, 
			subscription_id, subscription_item_id, meter_id,
			spec_version, type, event_id, time, source, subject, data,
			received_at
		FROM usage_events
		WHERE event_id = @reference_id
		LIMIT 1
	`

	args := pgx.NamedArgs{
		"reference_id": referenceID,
	}

	var model models.UsageEvent
	var dataJSON []byte

	err := tx.QueryRow(ctx, query, args).Scan(
		&model.OrgId,
		&model.Id,
		&model.SubscriptionId,
		&model.SubscriptionItemId,
		&model.MeterId,
		&model.SpecVersion,
		&model.Type,
		&model.EventId,
		&model.Time,
		&model.Source,
		&model.Subject,
		&dataJSON,
		&model.ReceivedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.UsageEvent{}, fmt.Errorf("usage event not found")
		}
		return entities.UsageEvent{}, fmt.Errorf("failed to find usage event by reference: %w", err)
	}

	// Parse JSON fields
	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &model.Data); err != nil {
			r.logger.Warn("Failed to unmarshal data", "error", err)
		}
	}

	return model.ToEntity(), nil
}

func (r *UsageEventRepository) Delete(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) error {
	tx := r.getTransactionFromContext(ctx)

	// Note: The repository interface expects to delete by orgID, subscriptionItemID, and time
	// but the schema has a primary key of [orgId, id]. We'll delete by the available fields.
	query := `
		DELETE FROM usage_events
		WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND time = @time
	`

	args := pgx.NamedArgs{
		"org_id":               orgID,
		"subscription_item_id": subscriptionItemID,
		"time":                 eventTime,
	}

	result, err := tx.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to delete usage event: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("usage event not found for deletion")
	}

	return nil
}

// AggregateUsageBySubscriptionItem aggregates usage for a subscription item based on the specified aggregation type
func (r *UsageEventRepository) AggregateUsageBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, 
	startTime, endTime time.Time, aggregationType entities.AggregationType) (float64, error) {

	tx := r.getTransactionFromContext(ctx)

	var query string

	// Build query based on aggregation type
	switch aggregationType {
	case entities.AggregationTypeSum:
		query = `
			SELECT COALESCE(SUM(CAST(data->>'quantity' AS FLOAT)), 0)
			FROM usage_events
			WHERE org_id = @org_id 
			  AND subscription_item_id = @subscription_item_id
			  AND time >= @start_time 
			  AND time < @end_time
		`
	case entities.AggregationTypeMax:
		query = `
			SELECT COALESCE(MAX(CAST(data->>'quantity' AS FLOAT)), 0)
			FROM usage_events
			WHERE org_id = @org_id 
			  AND subscription_item_id = @subscription_item_id
			  AND time >= @start_time 
			  AND time < @end_time
		`
	case entities.AggregationTypeAverage:
		query = `
			SELECT COALESCE(AVG(CAST(data->>'quantity' AS FLOAT)), 0)
			FROM usage_events
			WHERE org_id = @org_id 
			  AND subscription_item_id = @subscription_item_id
			  AND time >= @start_time 
			  AND time < @end_time
		`
	case entities.AggregationTypeLastDuringPeriod:
		query = `
			SELECT COALESCE(CAST(data->>'quantity' AS FLOAT), 0)
			FROM usage_events
			WHERE org_id = @org_id 
			  AND subscription_item_id = @subscription_item_id
			  AND time >= @start_time 
			  AND time < @end_time
			ORDER BY time DESC
			LIMIT 1
		`
	default:
		// Default to sum
		query = `
			SELECT COALESCE(SUM(CAST(data->>'quantity' AS FLOAT)), 0)
			FROM usage_events
			WHERE org_id = @org_id 
			  AND subscription_item_id = @subscription_item_id
			  AND time >= @start_time 
			  AND time < @end_time
		`
	}

	args := pgx.NamedArgs{
		"org_id":               orgID,
		"subscription_item_id": subscriptionItemID,
		"start_time":           startTime,
		"end_time":             endTime,
	}

	var result float64
	err := tx.QueryRow(ctx, query, args).Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate usage: %w", err)
	}

	return result, nil
}
