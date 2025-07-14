package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payloop/internal/testing/database"
)

func TestSetupTestDatabaseWithPrisma(t *testing.T) {
	// Setup test database with Prisma
	testDB := database.SetupTestDatabaseWithPrisma(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Verify we can query tables that should exist from the full schema
	tables := []string{
		"orgs",
		"customers",
		"subscriptions",
		"subscription_items",
		"prices",
		"meters",
		"usage_events", // Created separately for testing
		"invoices",
		"payment_methods",
		"refunds",
	}

	for _, table := range tables {
		var exists bool
		err := testDB.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, table).Scan(&exists)
		
		require.NoError(t, err, "Failed to check if table %s exists", table)
		assert.True(t, exists, "Table %s should exist", table)
	}

	// Test that we can insert and query data
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO orgs (id, name, country, created_at, updated_at)
		VALUES ('test_org', 'Test Org', 'US', NOW(), NOW())
	`)
	require.NoError(t, err, "Should be able to insert into orgs table")

	var orgName string
	err = testDB.Pool.QueryRow(ctx, "SELECT name FROM orgs WHERE id = 'test_org'").Scan(&orgName)
	require.NoError(t, err, "Should be able to query orgs table")
	assert.Equal(t, "Test Org", orgName)
}

func TestCompareSchemas(t *testing.T) {
	// Setup both database versions
	oldDB := database.SetupTestDatabase(t)
	defer oldDB.Cleanup(t)

	newDB := database.SetupTestDatabaseWithPrisma(t)
	defer newDB.Cleanup(t)

	ctx := context.Background()

	// Get tables from old setup
	oldTables := getTableNames(t, ctx, oldDB)
	
	// Get tables from new setup
	newTables := getTableNames(t, ctx, newDB)

	// Log the differences
	t.Logf("Tables in old setup: %d", len(oldTables))
	t.Logf("Tables in new setup: %d", len(newTables))
	
	// Find tables only in new setup
	for table := range newTables {
		if _, exists := oldTables[table]; !exists {
			t.Logf("Table '%s' exists in Prisma setup but not in old setup", table)
		}
	}
}

func getTableNames(t *testing.T, ctx context.Context, db *database.TestDatabase) map[string]bool {
	rows, err := db.Pool.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	require.NoError(t, err)
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		require.NoError(t, err)
		tables[tableName] = true
	}

	return tables
}