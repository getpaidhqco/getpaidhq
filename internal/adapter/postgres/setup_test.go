//go:build integration

// Integration tests for the postgres repositories. These run against a REAL
// PostgreSQL database and are gated behind the `integration` build tag so the
// default `go test ./...` (which has no DB) stays green.
//
// Run with:
//
//	go test -tags=integration ./internal/adapter/postgres/...
//
// DB selection:
//   - Default DSN points at the local docker stack
//     (`docker compose -f docker/docker-compose.yml up -d postgresql`),
//     host port 6432, db getpaidhq, user/pass postgres.
//   - Override with the TEST_DATABASE_URL env var.
//   - If no DB is reachable, every test t.Skip()s.
//
// Schema: the Prisma schema is the source of truth, but Prisma can't be run
// from a Go test. Instead we GORM-AutoMigrate the domain structs each repo
// persists. AutoMigrate is driven off the same gorm tags + TableName() the
// production code uses, so the table/column shapes match what the repos query.
// Caveat: AutoMigrate does NOT reproduce Prisma-specific constraints,
// defaults, enum types, or indexes — it only guarantees the columns the repos
// read/write exist. That is sufficient for round-trip repo tests.
//
// Isolation: every test uses a freshly generated unique org id, so rows from
// one test are invisible to another even though they share tables. Created
// rows are cleaned up via t.Cleanup. The schema is migrated once per package
// run (sync.Once).
package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

const defaultTestDSN = "host=localhost port=6432 user=postgres password=postgres dbname=getpaidhq sslmode=disable"

var (
	migrateOnce sync.Once
	migrateErr  error
)

// allModels lists every domain struct the tested repos persist. AutoMigrate
// needs the full set so multi-column foreign keys resolve.
func allModels() []any {
	return []any{
		&domain.Customer{},
		&domain.Cohort{},
		&domain.CustomerCohort{},
		&domain.Price{},
		&domain.Order{},
		&domain.OrderItem{},
		&domain.Subscription{},
		&domain.Payment{},
		&domain.Refund{},
		&domain.PaymentMethod{},
		&domain.DunningCampaign{},
		&domain.DunningAttempt{},
		&domain.DunningCommunication{},
		&domain.PaymentUpdateToken{},
		&domain.DunningConfiguration{},
		&domain.CustomerDunningHistory{},
	}
}

// testDB connects to the test database and migrates the schema once. If the
// DB is unreachable the test is skipped (not failed) so the suite degrades
// gracefully on machines with no DB.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := NewDatabase(dsn, nil)
	if err != nil {
		t.Skipf("skipping postgres integration test: cannot open db: %v", err)
	}
	// Quiet the GORM query log so test output stays readable. Production code
	// keeps logger.Info; this only affects the test session's handle.
	db.Logger = db.Logger.LogMode(gormlogger.Silent)

	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("skipping postgres integration test: cannot get sql.DB: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("skipping postgres integration test: db not reachable at %q: %v", dsn, err)
	}

	migrateOnce.Do(func() {
		migrateErr = db.AutoMigrate(allModels()...)
	})
	if migrateErr != nil {
		t.Fatalf("auto-migrate failed: %v", migrateErr)
	}

	return db
}

// uniqueOrg returns a fresh org id unique to the calling test, giving each
// test its own isolated slice of the shared tables.
func uniqueOrg(t *testing.T) string {
	t.Helper()
	return lib.GenerateId("org_test")
}

// cleanupOrg deletes every row this package's models own for the given org.
// Registered via t.Cleanup so a test leaves no residue even if it fails
// mid-way. Children are deleted before parents to respect FK constraints.
func cleanupOrg(t *testing.T, db *gorm.DB, orgId string) {
	t.Helper()
	t.Cleanup(func() {
		// Order matters: delete FK children before their parents.
		ordered := []any{
			&domain.DunningCommunication{},
			&domain.PaymentUpdateToken{},
			&domain.DunningAttempt{},
			&domain.DunningCampaign{},
			&domain.DunningConfiguration{},
			&domain.CustomerDunningHistory{},
			&domain.Refund{},
			&domain.Payment{},
			&domain.Subscription{},
			&domain.OrderItem{},
			&domain.Order{},
			&domain.Price{},
			&domain.PaymentMethod{},
			&domain.CustomerCohort{},
			&domain.Cohort{},
			&domain.Customer{},
		}
		for _, m := range ordered {
			db.Where("org_id = ?", orgId).Delete(m)
		}
	})
}

// --- fixture builders -------------------------------------------------------

func seedCustomer(t *testing.T, db *gorm.DB, orgId string) domain.Customer {
	t.Helper()
	c := domain.Customer{
		OrgId:     orgId,
		Id:        lib.GenerateId("cus"),
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     fmt.Sprintf("%s@example.com", lib.GenerateId("ada")),
		Phone:     "+15550001111",
		BillingAddress: domain.Address{
			Line1:   "1 Analytical Engine Way",
			City:    "London",
			Country: "GB",
		},
		Metadata:  map[string]string{"tier": "gold"},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, db.Create(&c).Error)
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
	require.NoError(t, db.Create(&p).Error)
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
	require.NoError(t, db.Create(&item).Error)
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
	require.NoError(t, db.Create(&o).Error)
	return o
}
