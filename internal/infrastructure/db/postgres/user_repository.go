package postgres

import (
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"

	_ "github.com/jackc/pgx/v5"
)

type UserRepository struct {
	*lib.PgDatabase
	logger logger.Logger
}

func NewUserRepository(database lib.Database, logger logger.Logger) UserRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return UserRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}
