package postgresgorm

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"getpaidhq/internal/core/port"
)

// NewDatabase opens a connection pool with explicit tuning.
//
// Defaults are intentionally conservative: GORM's bare defaults are
// MaxOpenConns=unlimited (will eat your PgBouncer or Postgres
// max_connections under burst), MaxIdleConns=2, and no lifetime limits
// (stale connections hang around through PgBouncer restarts). The
// values below assume a single-replica app behind PgBouncer with a
// per-server pool of ~25 — adjust if you're running more replicas or
// hitting the DB directly.
//
// All four knobs MUST be set together: without ConnMaxLifetime, idle
// connections accumulate; without ConnMaxIdleTime, the pool stays warm
// to MaxIdleConns long after demand drops.
//
// logLevel sets how chatty GORM is, independently of the app log level
// (GORM_LOG_LEVEL): "silent" | "error" | "warn" (slow queries + errors, the
// default) | "info"/"debug" (every query).
func NewDatabase(dsn string, log port.Logger, logLevel string) (*gorm.DB, error) {
	// Route GORM's SQL logs through the app logger so they share our slog
	// format/level. Tests construct the DB with a nil logger; fall back to a
	// silent GORM logger there rather than spamming test output.
	var gormLog logger.Interface
	if log != nil {
		gormLog = newGormLogger(log, parseGormLogLevel(logLevel))
	} else {
		gormLog = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLog,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: access *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}

// parseGormLogLevel maps the GORM_LOG_LEVEL string onto GORM's log levels.
// Empty or unknown values fall back to warn (slow queries + errors only).
func parseGormLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info", "debug":
		return logger.Info
	default:
		return logger.Warn
	}
}
