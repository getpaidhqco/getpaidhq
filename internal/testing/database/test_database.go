package database

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDatabase provides a test database instance
type TestDatabase struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	DSN       string
}

// SetupTestDatabase creates a new test database using testcontainers (legacy - only applies first migration)
func SetupTestDatabase(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Create postgres container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts("../../../prisma/migrations/20250326072241_init/migration.sql"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	return &TestDatabase{
		Container: postgresContainer,
		Pool:      pool,
		DSN:       dsn,
	}
}

// SetupTestDatabaseWithPrisma creates a new test database and syncs schema using Prisma
func SetupTestDatabaseWithPrisma(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Create postgres container without init scripts
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Apply Prisma schema using db push
	err = applyPrismaSchema(dsn)
	require.NoError(t, err, "Failed to apply Prisma schema")

	// Create connection pool
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Create usage_events table if needed (normally in separate DB)
	err = createUsageEventsTable(ctx, pool)
	if err != nil {
		t.Logf("Warning: Failed to create usage_events table: %v", err)
	}

	return &TestDatabase{
		Container: postgresContainer,
		Pool:      pool,
		DSN:       dsn,
	}
}

// applyPrismaSchema uses Prisma CLI to push the schema to the test database
func applyPrismaSchema(dsn string) error {
	// Create a clean environment without loading .env file
	env := os.Environ()
	
	// Replace or add GPHQ_DATABASE_URL (as configured in schema.prisma)
	found := false
	for i, e := range env {
		if strings.HasPrefix(e, "GPHQ_DATABASE_URL=") {
			env[i] = fmt.Sprintf("GPHQ_DATABASE_URL=%s", dsn)
			found = true
			break
		}
	}
	if !found {
		env = append(env, fmt.Sprintf("GPHQ_DATABASE_URL=%s", dsn))
	}
	
	// Also set DATABASE_URL just in case
	env = append(env, fmt.Sprintf("DATABASE_URL=%s", dsn))
	
	// Tell Prisma to ignore .env file
	env = append(env, "DOTENV_CONFIG_PATH=/dev/null")
	
	// Run prisma db push with DATABASE_URL set
	cmd := exec.Command("pnpm", "dlx", "prisma", "db", "push", "--skip-generate", "--accept-data-loss")
	cmd.Dir = "../../.." // Navigate to project root
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// createUsageEventsTable creates a simplified usage_events table for testing
func createUsageEventsTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS usage_events (
			org_id TEXT NOT NULL,
			id TEXT NOT NULL,
			subscription_id TEXT NOT NULL,
			subscription_item_id TEXT NOT NULL,
			meter_id TEXT NOT NULL,
			spec_version TEXT NOT NULL,
			type TEXT NOT NULL,
			event_id TEXT NOT NULL,
			time TIMESTAMPTZ NOT NULL,
			source TEXT NOT NULL,
			subject TEXT NOT NULL,
			data JSONB NOT NULL,
			received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			quantity DECIMAL(15,4),
			transaction_value BIGINT,
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (org_id, id)
		)
	`
	
	_, err := pool.Exec(ctx, query)
	return err
}

// Cleanup terminates the test database
func (db *TestDatabase) Cleanup(t *testing.T) {
	ctx := context.Background()
	
	if db.Pool != nil {
		db.Pool.Close()
	}
	
	if db.Container != nil {
		err := db.Container.Terminate(ctx)
		require.NoError(t, err)
	}
}

// TruncateTables clears all test data from specified tables
func (db *TestDatabase) TruncateTables(t *testing.T, tables ...string) {
	ctx := context.Background()
	
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		_, err := db.Pool.Exec(ctx, query)
		require.NoError(t, err)
	}
}

// ExecuteInTransaction runs a function within a database transaction
func (db *TestDatabase) ExecuteInTransaction(t *testing.T, fn func(context.Context) error) {
	ctx := context.Background()
	
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()
	
	// Add transaction to context
	ctx = context.WithValue(ctx, "tx", tx)
	
	err = fn(ctx)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	
	err = tx.Commit(ctx)
	require.NoError(t, err)
}