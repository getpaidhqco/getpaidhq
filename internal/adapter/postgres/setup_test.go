//go:build integration

// Integration tests for the postgres repositories. These run against a REAL
// PostgreSQL database managed by Testcontainers.
//
// Run with:
//
//	go test -tags=integration ./internal/adapter/postgres/...
//
// DB selection:
//   - Every run starts a fresh, isolated PostgreSQL container via Testcontainers.
//   - Cleanup is handled automatically by the container lifecycle.
//
// Schema: the Prisma schema is the source of truth, but Prisma can't be run
// from a Go test. Instead we GORM-AutoMigrate the *Row types each repo
// persists. AutoMigrate is driven off the same gorm tags + TableName() the
// production code uses, so the table/column shapes match what the repos query.
// Caveat: AutoMigrate does NOT reproduce Prisma-specific constraints,
// defaults, enum types, or indexes — it only guarantees the columns the repos
// read/write exist. That is sufficient for round-trip repo tests.
//
// Isolation: every test uses a freshly generated unique org id, so rows from
// one test are invisible to another even though they share tables. Created
// rows are cleaned up via t.Cleanup.
package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

var (
	sharedDB   *gorm.DB
	sharedOnce sync.Once
	sharedErr  error
	container  *postgres.PostgresContainer
)

func allModels() []any {
	return []any{
		&orgRow{},
		&customerRow{},
		&cohortRow{},
		&customerCohortRow{},
		&productRow{},
		&priceRow{},
		&orderRow{},
		&orderItemRow{},
		&subscriptionRow{},
		&paymentRow{},
		&refundRow{},
		&paymentMethodRow{},
		&pspConfigRow{},
		&settingRow{},
		&dunningCampaignRow{},
		&dunningAttemptRow{},
		&dunningCommunicationRow{},
		&paymentUpdateTokenRow{},
		&dunningConfigurationRow{},
		&customerDunningHistoryRow{},
		&apiKeyRow{},
		&billableMetricRow{},
		&meterEventRow{},
		&invoiceRow{},
		&invoiceLineItemRow{},
		&couponRow{},
		&couponCodeRow{},
		&discountRow{},
	}
}

// testDB starts a Postgres container once per package run and returns a GORM handle.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	sharedOnce.Do(func() {
		ctx := context.Background()
		dbName := "getpaidhq"
		dbUser := "postgres"
		dbPassword := "postgres"

		c, err := postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase(dbName),
			postgres.WithUsername(dbUser),
			postgres.WithPassword(dbPassword),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second)),
		)
		if err != nil {
			sharedErr = fmt.Errorf("failed to start postgres container: %w", err)
			return
		}
		container = c

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			sharedErr = fmt.Errorf("failed to get connection string: %w", err)
			return
		}

		db, err := NewDatabase(connStr, nil, "")
		if err != nil {
			sharedErr = fmt.Errorf("failed to open gorm connection: %w", err)
			return
		}
		db.Logger = db.Logger.LogMode(gormlogger.Silent)

		if err := db.AutoMigrate(allModels()...); err != nil {
			sharedErr = fmt.Errorf("auto-migrate failed: %w", err)
			return
		}
		sharedDB = db
	})

	if sharedErr != nil {
		t.Fatalf("test setup failed: %v", sharedErr)
	}

	return sharedDB
}

func uniqueOrg(t *testing.T) string {
	t.Helper()
	return lib.GenerateId("org_test")
}

func cleanupOrg(t *testing.T, db *gorm.DB, orgId string) {
	t.Helper()
	t.Cleanup(func() {
		ordered := []any{
			&discountRow{},
			&couponCodeRow{},
			&couponRow{},
			&dunningCommunicationRow{},
			&paymentUpdateTokenRow{},
			&dunningAttemptRow{},
			&dunningCampaignRow{},
			&dunningConfigurationRow{},
			&customerDunningHistoryRow{},
			&refundRow{},
			&paymentRow{},
			&subscriptionRow{},
			&orderItemRow{},
			&orderRow{},
			&priceRow{},
			&productRow{},
			&paymentMethodRow{},
			&customerCohortRow{},
			&cohortRow{},
			&customerRow{},
			&settingRow{},
			&pspConfigRow{},
			&apiKeyRow{},
		}
		for _, m := range ordered {
			db.Unscoped().Where("org_id = ?", orgId).Delete(m)
		}
		// The orgs table is keyed by `id`, not `org_id`.
		db.Unscoped().Where("id = ?", orgId).Delete(&orgRow{})
	})
}

func seedCustomer(t *testing.T, db *gorm.DB, orgId string) domain.Customer {
	t.Helper()
	c := domain.Customer{
		OrgId:     orgId,
		Id:        lib.GenerateId("cus"),
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     fmt.Sprintf("%s@example.com", lib.GenerateId("ada")),
		Phone:     "+155****1111",
		BillingAddress: domain.Address{
			Line1:   "1 Analytical Engine Way",
			City:    "London",
			Country: "GB",
		},
		Metadata:  map[string]string{"tier": "gold"},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	row := customerRowFromDomain(c)
	require.NoError(t, db.Create(&row).Error)
	return c
}

func seedPrice(t *testing.T, db *gorm.DB, orgId string) domain.Price {
	t.Helper()
	p := domain.Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              "Monthly Pro",
		Currency:           domain.USD,
		UnitPrice:          1999,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		CreatedAt:          time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:          time.Now().UTC().Truncate(time.Microsecond),
	}
	row := priceRowFromDomain(p)
	require.NoError(t, db.Create(&row).Error)
	return p
}

func seedOrderItem(t *testing.T, db *gorm.DB, orgId, orderId, priceId string) domain.OrderItem {
	t.Helper()
	item := domain.OrderItem{
		OrgId:       orgId,
		Id:          lib.GenerateId("oi"),
		OrderId:     orderId,
		PriceId:     priceId,
		Description: "Monthly Pro",
		Quantity:    1,
		Subtotal:    1999,
		Total:       1999,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	row := orderItemRowFromDomain(item)
	require.NoError(t, db.Omit("Price").Create(&row).Error)
	return item
}

func seedOrder(t *testing.T, db *gorm.DB, orgId, customerId string) domain.Order {
	t.Helper()
	o := domain.Order{
		OrgId:      orgId,
		Id:         lib.GenerateId("ord"),
		CustomerId: customerId,
		Reference:  "REF-" + lib.GenerateId("r"),
		Status:     domain.OrderStatusPending,
		Currency:   "USD",
		Total:      1999,
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	row := orderRowFromDomain(o)
	require.NoError(t, db.Omit("Customer", "Items").Create(&row).Error)
	return o
}
