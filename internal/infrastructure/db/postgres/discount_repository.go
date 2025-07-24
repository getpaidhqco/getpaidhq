package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
	"strings"
	"time"
)

type DiscountRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewDiscountRepository(primaryDb lib.Database, logger logger.Logger) repositories.DiscountRepository {
	logger.Debug("Creating new Discount Repository")
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return &DiscountRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *DiscountRepository) FindById(ctx context.Context, orgId string, id string) (entities.Discount, error) {
	tx := r.getTransactionFromContext(ctx)

	var discountModel models.Discount

	query := `SELECT id, org_id, name, type, value, code, starts_at, ends_at, 
			  max_redemptions, recurring, cycles, currency, active, 
			  created_at, updated_at, metadata
			  FROM discounts
			  WHERE org_id = @org_id AND id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&discountModel.Id,
		&discountModel.OrgId,
		&discountModel.Name,
		&discountModel.Type,
		&discountModel.Value,
		&discountModel.Code,
		&discountModel.StartsAt,
		&discountModel.EndsAt,
		&discountModel.MaxRedemptions,
		&discountModel.Recurring,
		&discountModel.Cycles,
		&discountModel.Currency,
		&discountModel.Active,
		&discountModel.CreatedAt,
		&discountModel.UpdatedAt,
		&discountModel.Metadata,
	)

	if err != nil {
		r.logger.Error("failed to find discount", err)
		if err.Error() == "no rows in result set" {
			return entities.Discount{}, lib.NewCustomError(lib.NotFoundError, "Discount not found", err)
		}
		return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Error finding discount", err)
	}

	// Convert model to entity
	discount := discountModel.ToEntity()

	return discount, nil
}

func (r *DiscountRepository) FindByCode(ctx context.Context, orgId string, code string) (entities.Discount, error) {
	tx := r.getTransactionFromContext(ctx)

	var discountModel models.Discount

	query := `SELECT id, org_id, name, type, value, code, starts_at, ends_at, 
			  max_redemptions, recurring, cycles, currency, active, 
			  created_at, updated_at, metadata
			  FROM discounts
			  WHERE org_id = @org_id AND UPPER(code) = UPPER(@code)`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"code":   code,
	}).Scan(
		&discountModel.Id,
		&discountModel.OrgId,
		&discountModel.Name,
		&discountModel.Type,
		&discountModel.Value,
		&discountModel.Code,
		&discountModel.StartsAt,
		&discountModel.EndsAt,
		&discountModel.MaxRedemptions,
		&discountModel.Recurring,
		&discountModel.Cycles,
		&discountModel.Currency,
		&discountModel.Active,
		&discountModel.CreatedAt,
		&discountModel.UpdatedAt,
		&discountModel.Metadata,
	)

	if err != nil {
		r.logger.Error("failed to find discount by code", err)
		if err.Error() == "no rows in result set" {
			return entities.Discount{}, lib.NewCustomError(lib.NotFoundError, "Discount code not found", err)
		}
		return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Error finding discount by code", err)
	}

	// Convert model to entity
	discount := discountModel.ToEntity()

	return discount, nil
}

func (r *DiscountRepository) Create(ctx context.Context, discount entities.Discount) (entities.Discount, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO discounts (
			  id, org_id, name, type, value, code, starts_at, ends_at, max_redemptions, 
			  recurring, cycles, currency, active, created_at, updated_at, metadata)
			  VALUES (@id, @org_id, @name, @type, @value, @code, @starts_at, @ends_at, @max_redemptions, 
			  @recurring, @cycles, @currency, @active, NOW(), NOW(), @metadata)`

	_, err := tx.Exec(ctx, query, discountEntityToNamedArgs(discount))

	if err != nil {
		r.logger.Error("failed to create discount", err)
		if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "code") {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Discount code already exists", err)
		}
		return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Error creating discount", err)
	}

	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepository) Update(ctx context.Context, discount entities.Discount) (entities.Discount, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE discounts
			  SET name = @name, type = @type, value = @value, code = @code, starts_at = @starts_at, ends_at = @ends_at, 
			  max_redemptions = @max_redemptions, recurring = @recurring, cycles = @cycles, currency = @currency, 
			  active = @active, updated_at = NOW(), metadata = @metadata
			  WHERE org_id = @org_id AND id = @id`

	commandTag, err := tx.Exec(ctx, query, discountEntityToNamedArgs(discount))

	if err != nil {
		r.logger.Error("failed to update discount", err)
		if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "code") {
			return entities.Discount{}, lib.NewCustomError(lib.ValidationError, "Discount code already exists", err)
		}
		return entities.Discount{}, lib.NewCustomError(lib.InternalError, "Error updating discount", err)
	}

	if commandTag.RowsAffected() == 0 {
		return entities.Discount{}, lib.NewCustomError(lib.NotFoundError, "Discount not found", nil)
	}

	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM discounts WHERE org_id = @org_id AND id = @id`

	commandTag, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})

	if err != nil {
		r.logger.Error("failed to delete discount", err)
		return lib.NewCustomError(lib.InternalError, "Error deleting discount", err)
	}

	if commandTag.RowsAffected() == 0 {
		return lib.NewCustomError(lib.NotFoundError, "Discount not found", nil)
	}

	return nil
}

func (r *DiscountRepository) List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Discount, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var discounts []entities.Discount
	var count int

	query := `SELECT id, org_id, name, type, value, code, starts_at, ends_at, 
			  max_redemptions, recurring, cycles, currency, active, 
			  created_at, updated_at, metadata, count(*) OVER()
			  FROM discounts
			  WHERE org_id = @org_id
			  ORDER BY
			    -- Handle timestamp columns
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'asc' THEN created_at
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'desc' THEN created_at
			        ELSE NULL
			    END DESC,
			    CASE
			        WHEN @sort_col = 'updated_at' AND @sort_dir = 'asc' THEN updated_at
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'updated_at' AND @sort_dir = 'desc' THEN updated_at
			        ELSE NULL
			    END DESC,

			    -- Handle text columns
			    CASE
			        WHEN @sort_col = 'name' AND @sort_dir = 'asc' THEN name
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'name' AND @sort_dir = 'desc' THEN name
			        ELSE NULL
			    END DESC,

			    -- Default to created_at desc if no valid sort column
			    created_at DESC
			  LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      pagination.Limit,
		"off":      pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})
	if err != nil {
		r.logger.Error("failed to list discounts", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error listing discounts", err)
	}
	defer rows.Close()

	for rows.Next() {
		var discountModel models.Discount

		err := rows.Scan(
			&discountModel.Id,
			&discountModel.OrgId,
			&discountModel.Name,
			&discountModel.Type,
			&discountModel.Value,
			&discountModel.Code,
			&discountModel.StartsAt,
			&discountModel.EndsAt,
			&discountModel.MaxRedemptions,
			&discountModel.Recurring,
			&discountModel.Cycles,
			&discountModel.Currency,
			&discountModel.Active,
			&discountModel.CreatedAt,
			&discountModel.UpdatedAt,
			&discountModel.Metadata,
			&count,
		)

		if err != nil {
			r.logger.Error("failed to scan discount", err)
			return nil, 0, lib.NewCustomError(lib.InternalError, "Error scanning discount", err)
		}

		// Convert model to entity
		discount := discountModel.ToEntity()
		discounts = append(discounts, discount)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating discounts", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error iterating discounts", err)
	}

	return discounts, count, nil
}

func (r *DiscountRepository) ListActive(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Discount, int, error) {
	tx := r.getTransactionFromContext(ctx)
	now := time.Now()

	var discounts []entities.Discount
	var count int

	query := `SELECT id, org_id, name, type, value, code, starts_at, ends_at, 
			  max_redemptions, recurring, cycles, currency, active, 
			  created_at, updated_at, metadata, count(*) OVER()
			  FROM discounts
			  WHERE org_id = @org_id
			    AND active = true
			    AND (starts_at IS NULL OR starts_at <= @now)
			    AND (ends_at IS NULL OR ends_at >= @now)
			  ORDER BY
			    -- Handle timestamp columns
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'asc' THEN created_at
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'desc' THEN created_at
			        ELSE NULL
			    END DESC,
			    CASE
			        WHEN @sort_col = 'updated_at' AND @sort_dir = 'asc' THEN updated_at
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'updated_at' AND @sort_dir = 'desc' THEN updated_at
			        ELSE NULL
			    END DESC,

			    -- Handle text columns
			    CASE
			        WHEN @sort_col = 'name' AND @sort_dir = 'asc' THEN name
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'name' AND @sort_dir = 'desc' THEN name
			        ELSE NULL
			    END DESC,

			    -- Default to created_at desc if no valid sort column
			    created_at DESC
			  LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"now":      now,
		"lim":      pagination.Limit,
		"off":      pagination.Offset,
		"sort_col": pagination.SortBy,
		"sort_dir": pagination.SortDirection,
	})
	if err != nil {
		r.logger.Error("failed to list active discounts", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error listing active discounts", err)
	}
	defer rows.Close()

	for rows.Next() {
		var discountModel models.Discount

		err := rows.Scan(
			&discountModel.Id,
			&discountModel.OrgId,
			&discountModel.Name,
			&discountModel.Type,
			&discountModel.Value,
			&discountModel.Code,
			&discountModel.StartsAt,
			&discountModel.EndsAt,
			&discountModel.MaxRedemptions,
			&discountModel.Recurring,
			&discountModel.Cycles,
			&discountModel.Currency,
			&discountModel.Active,
			&discountModel.CreatedAt,
			&discountModel.UpdatedAt,
			&discountModel.Metadata,
			&count,
		)

		if err != nil {
			r.logger.Error("failed to scan active discount", err)
			return nil, 0, lib.NewCustomError(lib.InternalError, "Error scanning active discount", err)
		}

		// Convert model to entity
		discount := discountModel.ToEntity()
		discounts = append(discounts, discount)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating active discounts", err)
		return nil, 0, lib.NewCustomError(lib.InternalError, "Error iterating active discounts", err)
	}

	return discounts, count, nil
}

func (r *DiscountRepository) CountRedemptions(ctx context.Context, orgId string, discountId string) (int, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT COUNT(*) 
			  FROM discount_redemptions 
			  WHERE org_id = @org_id AND discount_id = @discount_id`

	var count int
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      orgId,
		"discount_id": discountId,
	}).Scan(&count)

	if err != nil {
		r.logger.Error("failed to count redemptions", err)
		return 0, lib.NewCustomError(lib.InternalError, "Error counting redemptions", err)
	}

	return count, nil
}

func discountEntityToNamedArgs(entity entities.Discount) pgx.NamedArgs {
	metaJson, _ := json.Marshal(entity.Metadata)

	return pgx.NamedArgs{
		"id":              entity.Id,
		"org_id":          entity.OrgId,
		"name":            entity.Name,
		"type":            entity.Type,
		"value":           entity.Value,
		"code":            pgtype.Text{String: entity.Code, Valid: entity.Code != ""},
		"starts_at":       pgtype.Timestamptz{Time: entity.StartsAt, Valid: !entity.StartsAt.IsZero()},
		"ends_at":         pgtype.Timestamptz{Time: entity.EndsAt, Valid: !entity.EndsAt.IsZero()},
		"max_redemptions": entity.MaxRedemptions,
		"recurring":       entity.Recurring,
		"cycles":          entity.Cycles,
		"currency":        entity.Currency,
		"active":          entity.Active,
		"metadata":        metaJson,
	}
}
