package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
)

type SubscriptionItemRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewSubscriptionItemRepository(primaryDb *PgDatabase, logger logger.Logger) repositories.SubscriptionItemRepository {
	return SubscriptionItemRepository{
		PgDatabase: primaryDb,
		logger:     logger,
	}
}

func (r SubscriptionItemRepository) FindById(ctx context.Context, orgId string, id string) (entities.SubscriptionItem, error) {
	tx := r.getTransactionFromContext(ctx)

	var item models.SubscriptionItem
	query := `SELECT org_id, id, subscription_id, price_id, product_id, variant_id, 
              name, description, status, quantity, amount, currency, has_usage, usage_type, aggregation_type, 
              metadata, created_at, updated_at
              FROM subscription_items
              WHERE org_id = @org_id AND id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&item.OrgId,
		&item.Id,
		&item.SubscriptionId,
		&item.PriceId,
		&item.ProductId,
		&item.VariantId,
		&item.Name,
		&item.Description,
		&item.Status,
		&item.Quantity,
		&item.Amount,
		&item.Currency,
		&item.HasUsage,
		&item.UsageType,
		&item.AggregationType,
		&item.Metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find SubscriptionItem by id`, err.Error())
		return entities.SubscriptionItem{}, err
	}

	return item.ToEntity(), nil
}

func (r SubscriptionItemRepository) Create(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	tx := r.getTransactionFromContext(ctx)

	metaJson, _ := json.Marshal(entity.Metadata)
	query := `INSERT INTO subscription_items (
              org_id, id, subscription_id, price_id, product_id, variant_id, 
              name, description, status, quantity, amount, currency, has_usage, usage_type, aggregation_type, 
              metadata, created_at, updated_at)
              VALUES (
              @org_id, @id, @subscription_id, @price_id, @product_id, @variant_id, 
              @name, @description, @status, @quantity, @amount, @currency, @has_usage, @usage_type, @aggregation_type, 
              @metadata, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":           entity.OrgId,
		"id":               entity.Id,
		"subscription_id":  entity.SubscriptionId,
		"price_id":         entity.PriceId,
		"product_id":       pgtype.Text{String: entity.ProductId, Valid: entity.ProductId != ""},
		"variant_id":       pgtype.Text{String: entity.VariantId, Valid: entity.VariantId != ""},
		"name":             entity.Name,
		"description":      pgtype.Text{String: entity.Description, Valid: entity.Description != ""},
		"status":           entity.Status,
		"quantity":         entity.Quantity,
		"amount":           pgtype.Int8{Int64: entity.Amount, Valid: entity.Amount != 0},
		"currency":         entity.Currency,
		"has_usage":        entity.HasUsage,
		"usage_type":       pgtype.Text{String: entity.UsageType, Valid: entity.UsageType != ""},
		"aggregation_type": pgtype.Text{String: entity.AggregationType, Valid: entity.AggregationType != ""},
		"metadata":         metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to create SubscriptionItem`, err.Error())
		return entities.SubscriptionItem{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r SubscriptionItemRepository) Update(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	tx := r.getTransactionFromContext(ctx)

	metaJson, _ := json.Marshal(entity.Metadata)
	query := `UPDATE subscription_items SET
              name = @name,
              description = @description,
              status = @status,
              quantity = @quantity,
              amount = @amount,
              has_usage = @has_usage,
              usage_type = @usage_type,
              aggregation_type = @aggregation_type,
              metadata = @metadata,
              updated_at = NOW()
              WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":           entity.OrgId,
		"id":               entity.Id,
		"name":             entity.Name,
		"description":      pgtype.Text{String: entity.Description, Valid: entity.Description != ""},
		"status":           entity.Status,
		"quantity":         entity.Quantity,
		"amount":           pgtype.Int8{Int64: entity.Amount, Valid: entity.Amount != 0},
		"has_usage":        entity.HasUsage,
		"usage_type":       pgtype.Text{String: entity.UsageType, Valid: entity.UsageType != ""},
		"aggregation_type": pgtype.Text{String: entity.AggregationType, Valid: entity.AggregationType != ""},
		"metadata":         metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to update SubscriptionItem`, err.Error())
		return entities.SubscriptionItem{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r SubscriptionItemRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionItem, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, price_id, product_id, variant_id, 
              name, description, status, quantity, amount, currency, has_usage, usage_type, aggregation_type, 
              metadata, created_at, updated_at
              FROM subscription_items
              WHERE org_id = @org_id AND subscription_id = @subscription_id
              ORDER BY created_at ASC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":          orgId,
		"subscription_id": subscriptionId,
	})

	if err != nil {
		r.logger.Error(`failed to find SubscriptionItems by subscription_id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var items []entities.SubscriptionItem
	for rows.Next() {
		var item models.SubscriptionItem
		err := rows.Scan(
			&item.OrgId,
			&item.Id,
			&item.SubscriptionId,
			&item.PriceId,
			&item.ProductId,
			&item.VariantId,
			&item.Name,
			&item.Description,
			&item.Status,
			&item.Quantity,
			&item.Amount,
			&item.Currency,
			&item.HasUsage,
			&item.UsageType,
			&item.AggregationType,
			&item.Metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan SubscriptionItem`, err.Error())
			return nil, err
		}
		items = append(items, item.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return items, nil
}

func (r SubscriptionItemRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.SubscriptionItem, int, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, price_id, product_id, variant_id, 
              name, description, status, quantity, amount, currency, has_usage, usage_type, aggregation_type, 
              metadata, created_at, updated_at, count(*) OVER()
              FROM subscription_items
              WHERE org_id = @org_id
              ORDER BY created_at DESC
              LIMIT @limit OFFSET @offset`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"limit":  p.Limit,
		"offset": p.Offset,
	})

	if err != nil {
		r.logger.Error(`failed to find SubscriptionItems`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	var items []entities.SubscriptionItem
	var count int
	for rows.Next() {
		var item models.SubscriptionItem
		err := rows.Scan(
			&item.OrgId,
			&item.Id,
			&item.SubscriptionId,
			&item.PriceId,
			&item.ProductId,
			&item.VariantId,
			&item.Name,
			&item.Description,
			&item.Status,
			&item.Quantity,
			&item.Amount,
			&item.Currency,
			&item.HasUsage,
			&item.UsageType,
			&item.AggregationType,
			&item.Metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan SubscriptionItem`, err.Error())
			return nil, 0, err
		}
		items = append(items, item.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return items, count, nil
}

func (r SubscriptionItemRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM subscription_items WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})

	if err != nil {
		r.logger.Error(`failed to delete SubscriptionItem`, err.Error())
		return err
	}

	return nil
}