package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"

	"payloop/internal/models"
)

type SessionRepositoryIf interface {
	FindById(ctx context.Context, accountId string, id string) (models.Session, error)
	Create(ctx context.Context, input sessions.CreateSessionInput) (models.Session, error)
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

func (r *SessionRepository) FindById(ctx context.Context, accountId string, id string) (models.Session, error) {
	var session models.Session

	query := `INSERT INTO sessions (acct_id,id,cart_id, created_at, updated_at) 
			  VALUES (@acct_id,@id,@cart_id, NOW(), NOW())`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acct_id": accountId,
		"id":      id,
	}).Scan(&session)

	if err != nil {
		r.logger.Error(`failed to find session`, err)
		return models.Session{}, err
	}

	return session, nil
}

func (r *SessionRepository) Create(ctx context.Context, input sessions.CreateSessionInput) (models.Session, error) {
	var session models.Session

	query := `INSERT INTO sessions (acct_id,id,cart_id, created_at, updated_at) 
			  VALUES (@acct_id,@id,@cart_id, NOW(), NOW())
			  RETURNING (acct_id,id,cart_id)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acct_id": input.AccountId,
		"id":      input.Id,
		"cart_id": input.CartId,
	}).Scan(&session)

	if err != nil {
		r.logger.Error(`failed to insert Session`, err)
		return models.Session{}, err
	}

	return session, nil
}
