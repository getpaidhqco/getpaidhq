package postgres

import (
	"payloop/internal/core/port"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type UserRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewUserRepository(primaryDb lib.Database, logger port.Logger) UserRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return UserRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}
