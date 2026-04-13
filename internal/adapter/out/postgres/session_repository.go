package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type SessionRepository struct {
	*PgDatabase
	logger             port.Logger
	customerRepository port.CustomerRepository
}

func NewSessionRepository(primaryDb lib.Database, customerRepository port.CustomerRepository, logger port.Logger) port.SessionRepository {
	logger.Debug("Creating new Session Repository")
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SessionRepository{
		PgDatabase:         pgDatabase,
		logger:             logger,
		customerRepository: customerRepository,
	}
}

func (r SessionRepository) FindById(ctx context.Context, orgId string, id string) (domain.Session, error) {
	tx := r.getTransactionFromContext(ctx)

	var session domain.Session

	query := `SELECT org_id, id, cart_id, created_at, updated_at
			  FROM sessions
			  WHERE org_id = $1 AND id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(&session.OrgId, &session.Id, &session.CartId, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to find session", err)
		return domain.Session{}, err
	}

	return session, nil
}

func (r SessionRepository) Create(ctx context.Context, input domain.Session) (domain.Session, error) {
	tx := r.getTransactionFromContext(ctx)

	var session domain.Session

	query := `INSERT INTO sessions (org_id,id,cart_id, created_at, updated_at)
			  VALUES (@org_id,@id,@cart_id, NOW(), NOW())
			  RETURNING (org_id,id,cart_id)`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":  input.OrgId,
		"id":      input.Id,
		"cart_id": input.CartId,
	}).Scan(&session)

	if err != nil {
		r.logger.Error(`failed to insert Session`, err)
		return domain.Session{}, err
	}

	return session, nil
}
