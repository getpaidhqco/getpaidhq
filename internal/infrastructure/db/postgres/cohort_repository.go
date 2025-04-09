package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type CohortRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewCohortRepository(primaryDb lib.Database, logger logger.Logger) repositories.CohortRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CohortRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CohortRepository) FindById(ctx context.Context, orgId string, id string) (entities.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	var cohort models.Cohort
	query := `SELECT org_id, id, name, type, metadata, created_at, updated_at 
				FROM cohorts 
				WHERE org_id=@org_id AND id=@id`
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		cohort.OrgId,
		&cohort.Id,
		&cohort.Name,
		&cohort.Type,
		&cohort.Metadata,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Cohort`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.Cohort{}, err
	}
	return cohort.ToEntity(), nil
}

func (r CohortRepository) Create(ctx context.Context, input entities.Cohort) (entities.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO cohorts (org_id, id, name, type, metadata, created_at, updated_at)
			  VALUES (@org_id, @id, @name, @type, @metadata, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       input.Id,
		"name":     input.Name,
		"type":     input.Type,
		"metadata": input.Metadata,
	})
	if err != nil {
		r.logger.Error(`failed to create Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return entities.Cohort{}, err
	}

	return input, nil
}

func (r CohortRepository) Update(ctx context.Context, input entities.Cohort) (entities.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE cohorts SET name=@name, type=@type, metadata=@metadata, updated_at=NOW()
			  WHERE org_id=@org_id AND id=@id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       input.Id,
		"name":     input.Name,
		"type":     input.Type,
		"metadata": input.Metadata,
	})
	if err != nil {
		r.logger.Error(`failed to update Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return entities.Cohort{}, err
	}

	return input, nil
}

func (r CohortRepository) Delete(ctx context.Context, input entities.Cohort) (entities.Cohort, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM cohorts WHERE org_id=@org_id AND id=@id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": input.OrgId,
		"id":     input.Id,
	})
	if err != nil {
		r.logger.Error(`failed to delete Cohort`, "orgId", input.OrgId, "id", input.Id, "err", err.Error())
		return entities.Cohort{}, err
	}

	return input, nil
}
