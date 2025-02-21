package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type SessionRepository struct {
	*PgDatabase
	logger             logger.Logger
	customerRepository repositories.CustomerRepository
}

func NewSessionRepository(database lib.Database, customerRepository repositories.CustomerRepository, logger logger.Logger) repositories.SessionRepository {
	logger.Debug("Creating new Session Repository")
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SessionRepository{
		PgDatabase:         pgDatabase,
		logger:             logger,
		customerRepository: customerRepository,
	}
}

func (r SessionRepository) FindById(ctx context.Context, orgId string, id string) (entities.Session, error) {
	var session entities.Session

	query := `INSERT INTO sessions (org_id,id,cart_id, created_at, updated_at) 
			  VALUES (@org_id,@id,@cart_id, NOW(), NOW())`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&session)

	if err != nil {
		r.logger.Error(`failed to find session`, err)
		return entities.Session{}, err
	}

	return session, nil
}

func (r SessionRepository) Create(ctx context.Context, input entities.Session) (entities.Session, error) {
	var session entities.Session

	query := `INSERT INTO sessions (org_id,id,cart_id, created_at, updated_at) 
			  VALUES (@org_id,@id,@cart_id, NOW(), NOW())
			  RETURNING (org_id,id,cart_id)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":  input.OrgId,
		"id":      input.Id,
		"cart_id": input.CartId,
	}).Scan(&session)

	if err != nil {
		r.logger.Error(`failed to insert Session`, err)
		return entities.Session{}, err
	}

	return session, nil
}
