//go:build integration

package postgres

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from the current working directory until it finds go.mod,
// returning the module root. Integration tests run from the package dir, so the
// migrations live at <root>/schemas/app/migrations.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "reached filesystem root without finding go.mod")
		dir = parent
	}
}

// applyBaseline runs the operational Goose migrations against db.
// It applies both the app schema (schemas/app/migrations) and the usage schema
// (schemas/usage/migrations), using separate goose version-tracking tables so
// the two migration sequences do not collide.
func applyBaseline(t *testing.T, db *sql.DB) {
	t.Helper()
	require.NoError(t, goose.SetDialect("postgres"))
	root := repoRoot(t)

	// Apply the main app schema.
	goose.SetTableName("goose_db_version_app")
	appDir := filepath.Join(root, "schemas", "app", "migrations")
	require.NoError(t, goose.Up(db, appDir))

	// Apply the usage schema (meter_events lives here).
	goose.SetTableName("goose_db_version_usage")
	usageDir := filepath.Join(root, "schemas", "usage", "migrations")
	require.NoError(t, goose.Up(db, usageDir))

	// Restore the default table name so any other goose usage is unaffected.
	goose.SetTableName("goose_db_version")
}
