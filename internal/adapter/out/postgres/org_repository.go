package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"

	"payloop/internal/lib"
)

type OrgRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewOrgRepository(primaryDb lib.Database, logger port.Logger) port.OrgRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrgRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r OrgRepository) Create(ctx context.Context, entity domain.Org) (domain.Org, error) {
	tx := r.getTransactionFromContext(ctx)

	var Org domain.Org
	query := `INSERT INTO orgs (id, name, country, timezone, status, metadata, created_at, updated_at)
			  VALUES (@id, @name, @country, @timezone, @status, @metadata, NOW(), NOW())
			  RETURNING (id, name, country, timezone, status, metadata, created_at, updated_at)`

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		r.logger.Error(`failed to marshal metadata`, err)
		return domain.Org{}, err
	}

	err = tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":       entity.Id,
		"name":     entity.Name,
		"country":  entity.Country,
		"timezone": pgtype.Text{String: entity.Timezone, Valid: entity.Timezone != ""},
		"status":   entity.Status,
		"metadata": metadata,
	}).Scan(&Org)

	if err != nil {
		r.logger.Error(`failed to insert Org`, err)
		return domain.Org{}, err
	}

	return Org, nil
}
