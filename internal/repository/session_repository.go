package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"
)

type SessionRepositoryIf interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Session, error)
	Create(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error)
}

type SessionRepository struct {
	*lib.PgDatabase
	logger             lib.Logger
	customerRepository CustomerRepository
}

func NewSessionRepository(database lib.Database, customerRepository CustomerRepository, logger lib.Logger) SessionRepository {
	logger.Debug("Creating new Session Repository")
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SessionRepository{
		PgDatabase:         pgDatabase,
		logger:             logger,
		customerRepository: customerRepository,
	}
}

// WithTrx enables repository with transaction
func (r *SessionRepository) WithTrx(trxHandle interface{}) *SessionRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *SessionRepository) FindById(ctx context.Context, orgId string, id string) (entities.Session, error) {
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

func (r *SessionRepository) Create(ctx context.Context, input sessions.CreateSessionInput) (entities.Session, error) {
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
