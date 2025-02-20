package lib

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"log/slog"
	"payloop/internal/application/lib/logger"
	"sync"
)

const (
	// DBTransaction is database transaction handle set at router context
	DBTransaction = "db_trx"
)

type TransactionBeginner interface {
	Begin(ctx context.Context) (Committer, error)
}

type Committer interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	GetClient() interface{}
}

type Database interface {
	TransactionBeginner
	Ping(ctx context.Context) error
	Close()
}

type Tx interface {
	Begin(ctx context.Context) (Tx, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type PgCommitter struct {
	pgx.Tx
}

func (c PgCommitter) Commit(ctx context.Context) error {
	return c.Tx.Commit(ctx)
}
func (c PgCommitter) Rollback(ctx context.Context) error {
	return c.Tx.Rollback(ctx)
}
func (c PgCommitter) GetClient() interface{} {
	return c.Tx
}

type PgDatabase struct {
	*pgxpool.Pool
	pgx.Tx
	logger logger.Logger
}

var (
	pgInstance *PgDatabase
	pgOnce     sync.Once
)

func NewDatabase(env Env, logger logger.Logger) *PgDatabase {
	logger.Info("Connecting to database", "url", env.DBUrl)

	pgOnce.Do(func() {
		dbConfig, err := pgxpool.ParseConfig(env.DBUrl)
		dbConfig.ConnConfig.Tracer = &myQueryTracer{
			logger: logger,
		}
		pool, err := pgxpool.NewWithConfig(context.TODO(), dbConfig)
		if err != nil {
			logger.Error("could not connect to database", "error", err)
			return
		}

		pgInstance = &PgDatabase{
			pool,
			nil,
			logger,
		}
	})

	if pgInstance == nil {
		log.Fatalf("could not connect to database")
	}

	return pgInstance
}

type myQueryTracer struct {
	logger logger.Logger
}

func (l *myQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	// Failure
	if data.Err != nil {
		l.logger.
			Error("query end",
				slog.String("error", data.Err.Error()),
				slog.String("command_tag", data.CommandTag.String()),
			)
		return
	}

	// Success
	l.logger.
		Info("query end",
			slog.String("command_tag", data.CommandTag.String()),
		)
}

func (l *myQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	l.logger.Info("query start",
		slog.String("sql", data.SQL),
		slog.Any("args", data.Args),
	)
	return ctx
}

func (d *PgDatabase) Ping(ctx context.Context) error {
	return d.Ping(ctx)
}

func (d *PgDatabase) Close() {
	d.logger.Info("Closing database connection")
	d.Pool.Close()
}

func (d *PgDatabase) Begin(ctx context.Context) (Committer, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return PgCommitter{
		tx,
	}, nil
}
