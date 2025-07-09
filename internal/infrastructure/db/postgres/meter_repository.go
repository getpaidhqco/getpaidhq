package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type MeterRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewMeterRepository(primaryDb lib.Database, logger logger.Logger) repositories.MeterRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return MeterRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r MeterRepository) Create(ctx context.Context, meter entities.Meter) (entities.Meter, error) {
	tx := r.getTransactionFromContext(ctx)

	// Convert event filter to JSON
	eventFilterJson, err := json.Marshal(meter.EventFilter)
	if err != nil {
		r.logger.Error("failed to marshal event filter", err.Error())
		return entities.Meter{}, err
	}

	// Convert metadata to JSON
	metadataJson, err := json.Marshal(meter.Metadata)
	if err != nil {
		r.logger.Error("failed to marshal metadata", err.Error())
		return entities.Meter{}, err
	}

	query := `INSERT INTO meters (
		org_id, id, slug, name, description, event_name, event_filter, 
		aggregation_type, value_property, unit_type, display_name, 
		window_size, reset_interval, metadata, created_at, updated_at)
		VALUES (
		@org_id, @id, @slug, @name, @description, @event_name, @event_filter, 
		@aggregation_type, @value_property, @unit_type, @display_name, 
		@window_size, @reset_interval, @metadata, NOW(), NOW())`

	args := pgx.NamedArgs{
		"org_id":           meter.OrgId,
		"id":               meter.Id,
		"slug":             meter.Slug,
		"name":             meter.Name,
		"description":      meter.Description,
		"event_name":       meter.EventName,
		"event_filter":     eventFilterJson,
		"aggregation_type": string(meter.AggregationType),
		"value_property":   meter.ValueProperty,
		"unit_type":        string(meter.UnitType),
		"display_name":     meter.DisplayName,
		"window_size":      meter.WindowSize,
		"reset_interval":   meter.ResetInterval,
		"metadata":         metadataJson,
	}

	_, err = tx.Exec(ctx, query, args)

	if err != nil {
		r.logger.Error("failed to create meter", "err", err.Error())
		return entities.Meter{}, err
	}

	return r.FindById(ctx, meter.OrgId, meter.Id)
}

func (r MeterRepository) Update(ctx context.Context, meter entities.Meter) (entities.Meter, error) {
	tx := r.getTransactionFromContext(ctx)

	// Convert event filter to JSON
	eventFilterJson, err := json.Marshal(meter.EventFilter)
	if err != nil {
		r.logger.Error("failed to marshal event filter", err.Error())
		return entities.Meter{}, err
	}

	// Convert metadata to JSON
	metadataJson, err := json.Marshal(meter.Metadata)
	if err != nil {
		r.logger.Error("failed to marshal metadata", err.Error())
		return entities.Meter{}, err
	}

	query := `UPDATE meters SET
		name = @name,
		description = @description,
		event_name = @event_name,
		event_filter = @event_filter,
		aggregation_type = @aggregation_type,
		value_property = @value_property,
		unit_type = @unit_type,
		display_name = @display_name,
		window_size = @window_size,
		reset_interval = @reset_interval,
		metadata = @metadata,
		updated_at = NOW()
		WHERE org_id = @org_id AND id = @id`

	args := pgx.NamedArgs{
		"name":             meter.Name,
		"description":      meter.Description,
		"event_name":       meter.EventName,
		"event_filter":     eventFilterJson,
		"aggregation_type": string(meter.AggregationType),
		"value_property":   meter.ValueProperty,
		"unit_type":        string(meter.UnitType),
		"display_name":     meter.DisplayName,
		"window_size":      meter.WindowSize,
		"reset_interval":   meter.ResetInterval,
		"metadata":         metadataJson,
		"org_id":           meter.OrgId,
		"id":               meter.Id,
	}

	_, err = tx.Exec(ctx, query, args)

	if err != nil {
		r.logger.Error("failed to update meter", err.Error())
		return entities.Meter{}, err
	}

	return r.FindById(ctx, meter.OrgId, meter.Id)
}

func (r MeterRepository) FindById(ctx context.Context, orgId, meterId string) (entities.Meter, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, slug, name, description, event_name, event_filter, 
		aggregation_type, value_property, unit_type, display_name, 
		window_size, reset_interval, metadata, created_at, updated_at
		FROM meters
		WHERE org_id = @org_id AND id = @id`

	var model models.Meter
	var eventFilterJson, metadataJson []byte

	args := pgx.NamedArgs{
		"org_id": orgId,
		"id":     meterId,
	}

	err := tx.QueryRow(ctx, query, args).Scan(
		&model.OrgId,
		&model.Id,
		&model.Slug,
		&model.Name,
		&model.Description,
		&model.EventName,
		&eventFilterJson,
		&model.AggregationType,
		&model.ValueProperty,
		&model.UnitType,
		&model.DisplayName,
		&model.WindowSize,
		&model.ResetInterval,
		&metadataJson,
		&model.CreatedAt,
		&model.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.Meter{}, fmt.Errorf("meter not found")
		}
		r.logger.Error("failed to find meter by id", "err", err.Error())
		return entities.Meter{}, err
	}

	// Parse JSON fields
	if len(eventFilterJson) > 0 {
		if err := json.Unmarshal(eventFilterJson, &model.EventFilter); err != nil {
			r.logger.Error("failed to unmarshal event filter", err.Error())
			return entities.Meter{}, err
		}
	}

	if len(metadataJson) > 0 {
		if err := json.Unmarshal(metadataJson, &model.Metadata); err != nil {
			r.logger.Error("failed to unmarshal metadata", err.Error())
			return entities.Meter{}, err
		}
	}

	return model.ToEntity(), nil
}

func (r MeterRepository) FindBySlug(ctx context.Context, orgId, slug string) (entities.Meter, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, slug, name, description, event_name, event_filter, 
		aggregation_type, value_property, unit_type, display_name, 
		window_size, reset_interval, metadata, created_at, updated_at
		FROM meters
		WHERE org_id = @org_id AND slug = @slug`

	var model models.Meter
	var eventFilterJson, metadataJson []byte

	args := pgx.NamedArgs{
		"org_id": orgId,
		"slug":   slug,
	}

	err := tx.QueryRow(ctx, query, args).Scan(
		&model.OrgId,
		&model.Id,
		&model.Slug,
		&model.Name,
		&model.Description,
		&model.EventName,
		&eventFilterJson,
		&model.AggregationType,
		&model.ValueProperty,
		&model.UnitType,
		&model.DisplayName,
		&model.WindowSize,
		&model.ResetInterval,
		&metadataJson,
		&model.CreatedAt,
		&model.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return entities.Meter{}, fmt.Errorf("meter not found")
		}
		r.logger.Error("failed to find meter by slug", err.Error())
		return entities.Meter{}, err
	}

	// Parse JSON fields
	if len(eventFilterJson) > 0 {
		if err := json.Unmarshal(eventFilterJson, &model.EventFilter); err != nil {
			r.logger.Error("failed to unmarshal event filter", err.Error())
			return entities.Meter{}, err
		}
	}

	if len(metadataJson) > 0 {
		if err := json.Unmarshal(metadataJson, &model.Metadata); err != nil {
			r.logger.Error("failed to unmarshal metadata", err.Error())
			return entities.Meter{}, err
		}
	}

	return model.ToEntity(), nil
}

func (r MeterRepository) FindByEventName(ctx context.Context, orgId, eventName string) ([]entities.Meter, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT 
		org_id, id, slug, name, description, event_name, event_filter, 
		aggregation_type, value_property, unit_type, display_name, 
		window_size, reset_interval, metadata, created_at, updated_at
		FROM meters
		WHERE org_id = @org_id AND event_name = @event_name`

	args := pgx.NamedArgs{
		"org_id":     orgId,
		"event_name": eventName,
	}

	rows, err := tx.Query(ctx, query, args)
	if err != nil {
		r.logger.Error("failed to find meters by event name", err.Error())
		return nil, err
	}
	defer rows.Close()

	var meters []entities.Meter
	for rows.Next() {
		var model models.Meter
		var eventFilterJson, metadataJson []byte

		err := rows.Scan(
			&model.OrgId,
			&model.Id,
			&model.Slug,
			&model.Name,
			&model.Description,
			&model.EventName,
			&eventFilterJson,
			&model.AggregationType,
			&model.ValueProperty,
			&model.UnitType,
			&model.DisplayName,
			&model.WindowSize,
			&model.ResetInterval,
			&metadataJson,
			&model.CreatedAt,
			&model.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan meter", err.Error())
			return nil, err
		}

		// Parse JSON fields
		if len(eventFilterJson) > 0 {
			if err := json.Unmarshal(eventFilterJson, &model.EventFilter); err != nil {
				r.logger.Error("failed to unmarshal event filter", err.Error())
				return nil, err
			}
		}

		if len(metadataJson) > 0 {
			if err := json.Unmarshal(metadataJson, &model.Metadata); err != nil {
				r.logger.Error("failed to unmarshal metadata", err.Error())
				return nil, err
			}
		}

		meters = append(meters, model.ToEntity())
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating over meters", err.Error())
		return nil, err
	}

	return meters, nil
}

func (r MeterRepository) List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Meter], error) {
	tx := r.getTransactionFromContext(ctx)

	// Query for total count
	countQuery := `SELECT COUNT(*) FROM meters WHERE org_id = @org_id`
	var totalCount int
	countArgs := pgx.NamedArgs{
		"org_id": orgId,
	}
	err := tx.QueryRow(ctx, countQuery, countArgs).Scan(&totalCount)
	if err != nil {
		r.logger.Error("failed to count meters", err.Error())
		return dto.PaginatedResult[entities.Meter]{}, err
	}

	// Query for paginated results
	query := `SELECT 
		org_id, id, slug, name, description, event_name, event_filter, 
		aggregation_type, value_property, unit_type, display_name, 
		window_size, reset_interval, metadata, created_at, updated_at
		FROM meters
		WHERE org_id = @org_id
		ORDER BY created_at DESC
		LIMIT @limit OFFSET @offset`

	args := pgx.NamedArgs{
		"org_id": orgId,
		"limit":  pagination.Limit,
		"offset": pagination.Offset,
	}

	rows, err := tx.Query(ctx, query, args)
	if err != nil {
		r.logger.Error("failed to list meters", err.Error())
		return dto.PaginatedResult[entities.Meter]{}, err
	}
	defer rows.Close()

	var meters []entities.Meter
	for rows.Next() {
		var model models.Meter
		var eventFilterJson, metadataJson []byte

		err := rows.Scan(
			&model.OrgId,
			&model.Id,
			&model.Slug,
			&model.Name,
			&model.Description,
			&model.EventName,
			&eventFilterJson,
			&model.AggregationType,
			&model.ValueProperty,
			&model.UnitType,
			&model.DisplayName,
			&model.WindowSize,
			&model.ResetInterval,
			&metadataJson,
			&model.CreatedAt,
			&model.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan meter", err.Error())
			return dto.PaginatedResult[entities.Meter]{}, err
		}

		// Parse JSON fields
		if len(eventFilterJson) > 0 {
			if err := json.Unmarshal(eventFilterJson, &model.EventFilter); err != nil {
				r.logger.Error("failed to unmarshal event filter", err.Error())
				return dto.PaginatedResult[entities.Meter]{}, err
			}
		}

		if len(metadataJson) > 0 {
			if err := json.Unmarshal(metadataJson, &model.Metadata); err != nil {
				r.logger.Error("failed to unmarshal metadata", err.Error())
				return dto.PaginatedResult[entities.Meter]{}, err
			}
		}

		meters = append(meters, model.ToEntity())
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating over meters", err.Error())
		return dto.PaginatedResult[entities.Meter]{}, err
	}

	// Calculate pagination metadata
	hasMore := (pagination.Page+1)*pagination.Limit < totalCount

	return dto.PaginatedResult[entities.Meter]{
		Items:      meters,
		TotalCount: totalCount,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
		HasMore:    hasMore,
	}, nil
}

func (r MeterRepository) Delete(ctx context.Context, orgId, meterId string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM meters WHERE org_id = @org_id AND id = @id`
	args := pgx.NamedArgs{
		"org_id": orgId,
		"id":     meterId,
	}
	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		r.logger.Error("failed to delete meter", err.Error())
		return err
	}

	return nil
}
