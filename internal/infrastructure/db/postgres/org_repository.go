package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"

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

	var Org entities.Org
	query := `INSERT INTO orgs (id, name, country, status, description, metadata, created_at, updated_at) 
			  VALUES (@id, @name, @country, @status, @description, @metadata, NOW(), NOW())
			  RETURNING (id,name,country,status,description, metadata,created_at,updated_at)`

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		r.logger.Error(`failed to marshal metadata`, err)
		return entities.Org{}, err
	}

	err = tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":          entity.Id,
		"name":        entity.Name,
		"country":     entity.Country,
		"status":      entity.Status,
		"description": entity.Description,
		"metadata":    metadata,
	}).Scan(&Org)

	if err != nil {
		r.logger.Error(`failed to insert Org`, err)
		return entities.Org{}, err
	}

	return Org, nil
}
