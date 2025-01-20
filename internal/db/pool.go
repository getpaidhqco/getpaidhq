package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"log/slog"
	"os"
	"payloop/internal/env"
	"sync"
)

type PgDatabase struct {
	*pgxpool.Pool
}

var (
	pgInstance *PgDatabase
	pgOnce     sync.Once
)

func NewDatabase(ctx context.Context) *PgDatabase {
	env.Load()
	slog.Info("Connecting to database", "url", os.Getenv("DATABASE_URL"))

	pgOnce.Do(func() {
		pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			fmt.Println(err)
			return
		}

		pgInstance = &PgDatabase{pool}
	})

	if pgInstance == nil {
		log.Fatalf("could not connect to database")
	}
	return pgInstance
}

func (pg *PgDatabase) Ping(ctx context.Context) error {
	return pg.Ping(ctx)
}

func (pg *PgDatabase) Close() {
	pg.Close()
}

func (pg *PgDatabase) GetDb() any {
	return pg
}
