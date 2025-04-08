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

func TestGetMRR(t *testing.T) {
	// Mock dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	reportingDb := NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)

	repo := NewReportRepository(reportingDb, db, logger)

	// Define test data
	orgId := "mollie"
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now()

	// Mock the transaction and queries

	// Call the method
	_, err := repo.GetMRR(context.Background(), orgId, startDate, endDate)

	// Assert results
	assert.NoError(t, err)
}

func TestGetARR(t *testing.T) {
	// Mock dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	reportingDb := NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)

	repo := NewReportRepository(reportingDb, db, logger)

	// Define test data
	orgId := "mollie"
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now()

	// Mock the transaction and queries

	// Call the method
	_, err := repo.GetARR(context.Background(), orgId, startDate, endDate)

	// Assert results
	assert.NoError(t, err)
}
