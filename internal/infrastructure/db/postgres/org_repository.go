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
)

type OrgRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewOrgRepository(primaryDb lib.Database, logger logger.Logger) repositories.OrgRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrgRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r OrgRepository) Create(ctx context.Context, entity entities.Org) (entities.Org, error) {
	tx := r.getTransactionFromContext(ctx)

	var orgModel models.Org
	query := `INSERT INTO orgs (id, name, country, timezone, status, metadata, created_at, updated_at) 
			  VALUES (@id, @name, @country, @timezone, @status, @metadata, NOW(), NOW())
			  RETURNING id, name, country, timezone, status, metadata, created_at, updated_at`

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		r.logger.Error(`failed to marshal metadata`, err)
		return entities.Org{}, err
	}

	err = tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":       entity.Id,
		"name":     entity.Name,
		"country":  entity.Country,
		"timezone": pgtype.Text{String: entity.Timezone, Valid: entity.Timezone != ""},
		"status":   entity.Status,
		"metadata": metadata,
	}).Scan(&orgModel.Id, &orgModel.Name, &orgModel.Country, &orgModel.Timezone, &orgModel.Status, &orgModel.Metadata, &orgModel.CreatedAt, &orgModel.UpdatedAt)

	if err != nil {
		r.logger.Error(`failed to insert Org`, err)
		return entities.Org{}, err
	}

	return orgModel.ToEntity(), nil
}

func (r OrgRepository) FindById(ctx context.Context, id string) (entities.Org, error) {
	tx := r.getTransactionFromContext(ctx)

	var orgModel models.Org

	query := `SELECT id, name, country, timezone, status, metadata, created_at, updated_at 
			  FROM orgs WHERE id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id": id,
	}).Scan(&orgModel.Id, &orgModel.Name, &orgModel.Country, &orgModel.Timezone, &orgModel.Status, &orgModel.Metadata, &orgModel.CreatedAt, &orgModel.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			r.logger.Warn("Organization not found", "id", id)
			return entities.Org{}, err
		}
		r.logger.Error("Failed to get organization", "error", err, "id", id)
		return entities.Org{}, err
	}

	return orgModel.ToEntity(), nil
}

func (r OrgRepository) Update(ctx context.Context, entity entities.Org) (entities.Org, error) {
	tx := r.getTransactionFromContext(ctx)

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		r.logger.Error("failed to marshal metadata", err)
		return entities.Org{}, err
	}

	query := `UPDATE orgs 
			  SET name = @name, country = @country, timezone = @timezone, status = @status, 
			      metadata = @metadata, updated_at = NOW()
			  WHERE id = @id
			  RETURNING id, name, country, timezone, status, metadata, created_at, updated_at`

	var orgModel models.Org

	err = tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":       entity.Id,
		"name":     entity.Name,
		"country":  entity.Country,
		"timezone": pgtype.Text{String: entity.Timezone, Valid: entity.Timezone != ""},
		"status":   entity.Status,
		"metadata": metadata,
	}).Scan(&orgModel.Id, &orgModel.Name, &orgModel.Country, &orgModel.Timezone, &orgModel.Status, &orgModel.Metadata, &orgModel.CreatedAt, &orgModel.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			r.logger.Warn("Organization not found for update", "id", entity.Id)
			return entities.Org{}, err
		}
		r.logger.Error("Failed to update organization", "error", err, "id", entity.Id)
		return entities.Org{}, err
	}

	return orgModel.ToEntity(), nil
}

func (r OrgRepository) List(ctx context.Context, pagination dto.Pagination) ([]entities.Org, int, error) {
	tx := r.getTransactionFromContext(ctx)

	// Count total records
	countQuery := `SELECT COUNT(*) FROM orgs`
	var total int
	err := tx.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		r.logger.Error("Failed to count organizations", "error", err)
		return nil, 0, err
	}

	// Build the main query with sorting and pagination
	selectPart := `SELECT id, name, country, timezone, status, metadata, created_at, updated_at FROM orgs`
	orderPart := ` ORDER BY ` + pagination.SortBy + ` ` + pagination.SortDirection
	limitPart := ` LIMIT @limit OFFSET @offset`
	query := selectPart + orderPart + limitPart

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"limit":  pagination.Limit,
		"offset": pagination.Offset,
	})
	if err != nil {
		r.logger.Error("Failed to list organizations", "error", err)
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []entities.Org
	for rows.Next() {
		var orgModel models.Org

		err := rows.Scan(&orgModel.Id, &orgModel.Name, &orgModel.Country, &orgModel.Timezone, &orgModel.Status, &orgModel.Metadata, &orgModel.CreatedAt, &orgModel.UpdatedAt)
		if err != nil {
			r.logger.Error("Failed to scan organization row", "error", err)
			return nil, 0, err
		}

		orgs = append(orgs, orgModel.ToEntity())
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating organization rows", "error", err)
		return nil, 0, err
	}

	return orgs, total, nil
}
