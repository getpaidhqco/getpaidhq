package postgres

import (
	"context"
	"github.com/stretchr/testify/assert"
	"payloop/internal/lib"
	"testing"
	"time"
)

func TestStoreDailyMetrics(t *testing.T) {
	// Mock dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	reportingDb := NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)

	repo := NewReportRepository(reportingDb, db, logger)

	// Define test data
	orgId := "mollie"
	date := time.Now()

	// Mock the transaction and queries

	// Call the method
	err := repo.StoreDailyMetrics(context.Background(), orgId, date)

	// Assert results
	assert.NoError(t, err)
}
