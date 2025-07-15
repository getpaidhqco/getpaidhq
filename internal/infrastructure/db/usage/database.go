package usage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
	"payloop/internal/application/lib/logger"
)

var Module = fx.Module("usage-database",
	fx.Provide(
		fx.Annotate(
			NewUsageDatabase,
			fx.ResultTags(`name:"usageDB"`),
		),
	),
)

// NewUsageDatabase creates a new connection to the usage database
func NewUsageDatabase(logger logger.Logger) (*sql.DB, error) {
	// Get connection string from environment
	connectionString := os.Getenv("USAGE_DATABASE_URL")
	if connectionString == "" {
		// Default for development
		connectionString = "postgres://postgres:postgres@localhost:5433/payloop_usage"
		logger.Warn("USAGE_DATABASE_URL not set, using default development connection")
	}

	// Open database connection
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open usage database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // Connections are reused forever

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping usage database: %w", err)
	}

	logger.Info("Connected to usage database", "url", maskConnectionString(connectionString))
	
	return db, nil
}

// maskConnectionString masks the password in the connection string for logging
func maskConnectionString(connStr string) string {
	// Simple masking - in production use a proper URL parser
	if len(connStr) > 20 {
		return connStr[:20] + "..."
	}
	return "***"
}