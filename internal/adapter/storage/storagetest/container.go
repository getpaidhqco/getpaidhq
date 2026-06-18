// Package storagetest provides the driver-agnostic integration harness and
// conformance suite that every storage adapter (postgresgorm, postgrespgx)
// runs against. It owns the testcontainer + Goose baseline so both adapters
// exercise the exact production schema, and (in the conformance suite) seeds
// through the repository ports so the assertions are identical across drivers.
//
// Adapters import this package only from their //go:build integration test
// files; nothing in the normal build depends on it.
package storagetest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	sharedDSN  string
	sharedOnce sync.Once
	sharedErr  error
	container  *tcpostgres.PostgresContainer
)

// StartPostgres boots a single fresh postgres:17-alpine testcontainer per test
// process, applies the operational + usage Goose baselines, and returns the
// connection DSN. Each adapter opens its own pool/handle from this DSN, so the
// harness stays driver-agnostic. The dev DB at localhost:10432 is never touched.
func StartPostgres(t *testing.T) string {
	t.Helper()
	sharedOnce.Do(func() {
		ctx := context.Background()
		c, err := tcpostgres.Run(ctx,
			"postgres:17-alpine",
			tcpostgres.WithDatabase("getpaidhq"),
			tcpostgres.WithUsername("postgres"),
			tcpostgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second)),
		)
		if err != nil {
			sharedErr = fmt.Errorf("start postgres container: %w", err)
			return
		}
		container = c

		dsn, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			sharedErr = fmt.Errorf("container connection string: %w", err)
			return
		}
		if err := applyBaseline(dsn); err != nil {
			sharedErr = fmt.Errorf("apply baseline: %w", err)
			return
		}
		sharedDSN = dsn
	})
	if sharedErr != nil {
		t.Fatalf("storagetest setup failed: %v", sharedErr)
	}
	return sharedDSN
}

// applyBaseline applies the operational (app) and usage Goose baselines, each
// tracked in its own goose version table so they don't collide.
func applyBaseline(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	for _, m := range []struct{ table, dir string }{
		{"goose_db_version_app", filepath.Join(root, "schemas", "app", "migrations")},
		{"goose_db_version_usage", filepath.Join(root, "schemas", "usage", "migrations")},
	} {
		goose.SetTableName(m.table)
		if err := goose.Up(db, m.dir); err != nil {
			return fmt.Errorf("goose up %s: %w", m.dir, err)
		}
	}
	goose.SetTableName("goose_db_version")
	return nil
}

// repoRoot walks up from the working directory to the module root (go.mod).
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("reached filesystem root without finding go.mod")
		}
		dir = parent
	}
}
