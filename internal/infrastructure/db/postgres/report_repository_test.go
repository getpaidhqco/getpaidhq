package postgres

import (
	"context"
	"github.com/stretchr/testify/assert"
	"payloop/internal/application/dto"
	"payloop/internal/lib"
	"testing"
	"time"
)

func TestStoreDailyMetricsForRange(t *testing.T) {
	// Mock dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	reportingDb := NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)

	repo := NewReportRepository(reportingDb, db, logger)

	// Define test data
	orgId := "org_2vl8HFwFQMPaZ1Y7fh2Foi0THpU"
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Now().UTC()

	for d := startDate; !d.After(endDate); d = d.Add(24 * time.Hour) {
		err := repo.StoreDailyMetrics(context.Background(), dto.ProcessDailyMetricsInput{
			OrgId:    orgId,
			Date:     d,
			Timezone: "Africa/Johannesburg",
		})
		assert.NoError(t, err)
	}

}

func TestStoreDailyMetrics(t *testing.T) {
	// Mock dependencies
	env := lib.NewEnv()
	logger := lib.GetLogger()
	db := NewDatabase(env.Get("DATABASE_URL"), logger)
	reportingDb := NewDatabase(env.Get("REPORTING_DATABASE_URL"), logger)

	repo := NewReportRepository(reportingDb, db, logger)

	// Define test data
	orgId := "mollie"
	date := time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC)

	// Mock the transaction and queries

	// Call the method
	err := repo.StoreDailyMetrics(context.Background(), dto.ProcessDailyMetricsInput{
		OrgId:    orgId,
		Date:     date,
		Timezone: "Africa/Johannesburg",
	})

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
