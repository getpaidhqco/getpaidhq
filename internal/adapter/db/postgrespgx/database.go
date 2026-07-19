package postgrespgx

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/port"
)

// NewDatabase opens a pgx connection pool with explicit tuning that mirrors the
// gorm adapter's *sql.DB knobs, so swapping DB_DRIVER doesn't change the
// connection footprint:
//
//	gorm                         pgxpool
//	MaxOpenConns=25      ->       MaxConns=25
//	MaxIdleConns=10      ->       MinConns=0 (kept-warm floor; idle reaping below)
//	ConnMaxLifetime=5m   ->       MaxConnLifetime=5m
//	ConnMaxIdleTime=1m   ->       MaxConnIdleTime=1m
//
// logLevel is accepted for symmetry with the gorm constructor; pgx query
// tracing is left off unless explicitly enabled in a later change (the gorm
// SQL log is the chatty one in practice).
func NewDatabase(dsn string, log port.Logger, logLevel string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgrespgx: parse dsn: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("postgrespgx: open pool: %w", err)
	}

	// Fail fast on a bad DSN/unreachable DB, same as gorm.Open validating the
	// connection eagerly.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgrespgx: ping: %w", err)
	}

	return pool, nil
}
