//go:build integration

package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
)

// repoRoot walks up from the working directory to the module root (where go.mod lives).
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

// applyBaseline applies the operational (app) and usage Goose baselines to db,
// each tracked in its own goose version table so they don't collide.
func applyBaseline(db *sql.DB) error {
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
	// Restore the default table name so any other goose usage is unaffected.
	goose.SetTableName("goose_db_version")
	return nil
}
