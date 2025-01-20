package lib

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"log"
	"sync"
)

type TransactionBeginner interface {
	Begin(ctx context.Context) (Committer, error)
}

type Committer interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Database interface {
	TransactionBeginner
	Ping(ctx context.Context) error
	Close()
}

type PgDatabase struct {
	*pgxpool.Pool
}

var (
	pgInstance *PgDatabase
	pgOnce     sync.Once
)

func NewDatabase(lc fx.Lifecycle, env Env, logger Logger) *PgDatabase {
	logger.Info("Connecting to database", "url", env.DBUrl)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pgInstance.Close()
			return nil
		},
	})

	pgOnce.Do(func() {
		pool, err := pgxpool.New(context.TODO(), env.DBUrl)
		if err != nil {
			logger.Error("could not connect to database", "error", err)
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

func (pg *PgDatabase) Begin(ctx context.Context) (Committer, error) {
	return pg.Pool.Begin(ctx)
}

func (pg *PgDatabase) Commit(ctx context.Context) (Committer, error) {
	return pg.Pool.Begin(ctx)
}
