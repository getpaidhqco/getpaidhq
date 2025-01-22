package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"payloop/internal/domain/accounts"

	"payloop/internal/lib"

	"payloop/internal/models"
)

type AccountRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewAccountRepository(database lib.Database, logger lib.Logger) AccountRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return AccountRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *AccountRepository) Create(ctx context.Context, input accounts.CreateAccountInput) (models.Account, error) {
	AccountId := "t_" + ksuid.New().String()
	var Account models.Account
	query := `INSERT INTO accounts (id, name, description, created_at, updated_at) 
			  VALUES (@id, @name, @description, NOW(), NOW())
			  RETURNING (id,name,description,created_at,updated_at)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"id":          AccountId,
		"name":        input.Name,
		"description": input.Description,
	}).Scan(&Account)

	if err != nil {
		r.logger.Error(`failed to insert Account`, err)
		return models.Account{}, err
	}

	return Account, nil
}
