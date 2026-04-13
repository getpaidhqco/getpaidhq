package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"log/slog"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type PgDatabase struct {
	*pgxpool.Pool
	pgx.Tx
	logger port.Logger
}

func (r PgDatabase) getTransactionFromContext(ctx context.Context) QueryRower {
	var p QueryRower = r.Pool
	tx := ctx.Value(lib.DBTransaction)
	if tx != nil {
		p = tx.(QueryRower)
	}

	return p
}

func NewDatabase(url string, logger port.Logger) lib.Database {
	logger.Info("Connecting to database", "url", url)

	dbConfig, err := pgxpool.ParseConfig(url)
	//dbConfig.ConnConfig.Tracer = &myQueryTracer{
	//	logger: logger,
	//}
	pool, err := pgxpool.NewWithConfig(context.TODO(), dbConfig)
	if err != nil {
		log.Fatalf("could not connect to database %v", err)
		return nil
	}

	return &PgDatabase{
		pool,
		nil,
		logger,
	}
}

type myQueryTracer struct {
	logger port.Logger
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

func (d *PgDatabase) Begin(ctx context.Context) (lib.Committer, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return PgCommitter{
		tx,
	}, nil
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
