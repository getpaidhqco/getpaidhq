package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orgs"
	"payloop/internal/domain/repositories"

	"payloop/internal/lib"
)

type OrgRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewOrgRepository(database lib.Database, logger lib.Logger) repositories.OrgRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrgRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r OrgRepository) Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error) {
	OrgId := "t_" + ksuid.New().String()
	var Org entities.Org
	query := `INSERT INTO orgs (id, name, description, created_at, updated_at) 
			  VALUES (@id, @name, @description, NOW(), NOW())
			  RETURNING (id,name,description,created_at,updated_at)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"id":          OrgId,
		"name":        input.Name,
		"description": input.Description,
	}).Scan(&Org)

	if err != nil {
		r.logger.Error(`failed to insert Org`, err)
		return entities.Org{}, err
	}

	return Org, nil
}
