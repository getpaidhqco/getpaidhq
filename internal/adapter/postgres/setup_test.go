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
// Schema: each container has the real operational baseline applied via Goose
// (schemas/app/migrations), so enums, FK constraints, defaults, and indexes
// match production exactly.
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

	"github.com/shopspring/decimal"
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

		db, err := NewDatabase(connStr, nil, "silent")
		if err != nil {
			sharedErr = fmt.Errorf("failed to open gorm connection: %w", err)
			return
		}
		db.Logger = db.Logger.LogMode(gormlogger.Silent)

		sqlDB, err := db.DB()
		if err != nil {
			sharedErr = fmt.Errorf("failed to get *sql.DB: %w", err)
			return
		}
		if err := applyBaseline(sqlDB); err != nil {
			sharedErr = fmt.Errorf("failed to apply baseline migrations: %w", err)
			return
		}
		sharedDB = db
	})

	if sharedErr != nil {
		t.Fatalf("test setup failed: %v", sharedErr)
	}

	return sharedDB
}

// seedVariantChain seeds a product + variant (required parent chain for prices)
// and returns the variant id. Callers must set price.VariantId to the returned value.
func seedVariantChain(t *testing.T, db *gorm.DB, orgId string) string {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	productId := lib.GenerateId("prod")
	require.NoError(t, db.Create(&productRow{
		OrgId:     orgId,
		Id:        productId,
		Name:      "Test Product",
		Status:    domain.ProductStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	variantId := lib.GenerateId("var")
	require.NoError(t, db.Create(&variantRow{
		OrgId:     orgId,
		Id:        variantId,
		ProductId: productId,
		Name:      "Default",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	return variantId
}

// seedOrg inserts a minimal org row so FK constraints on child tables are satisfied.
func seedOrg(t *testing.T, db *gorm.DB, orgId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	row := orgRow{
		Id:        orgId,
		Name:      "Test Org " + orgId,
		Country:   "US",
		Timezone:  "UTC",
		Status:    domain.OrgStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&row).Error)
}

// uniqueOrg generates a unique org ID and seeds the matching org row so that
// FK constraints on all child tables (customers, orders, api_keys, …) are satisfied.
func uniqueOrg(t *testing.T) string {
	t.Helper()
	db := testDB(t)
	orgId := lib.GenerateId("org_test")
	seedOrg(t, db, orgId)
	return orgId
}

func cleanupOrg(t *testing.T, db *gorm.DB, orgId string) {
	t.Helper()
	t.Cleanup(func() {
		ordered := []any{
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
			&cartRow{},
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
	// default_payment_method_id has a FK constraint: omit the column (→ NULL)
	// when no payment method is set, otherwise postgres rejects the empty string.
	require.NoError(t, db.Omit("DefaultPaymentMethodId").Create(&row).Error)
	return c
}

func seedPrice(t *testing.T, db *gorm.DB, orgId string) domain.Price {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	// prices.variant_id is NOT NULL with a FK to variants → products → orgs.
	variantId := seedVariantChain(t, db, orgId)
	p := domain.Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		VariantId:          variantId,
		Label:              "Monthly Pro",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          1999,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		CreatedAt:          now,
		UpdatedAt:          now,
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
		// metadata is NOT NULL in the DB; product_id is NOT NULL but has no FK.
		Metadata:  map[string]string{},
		ProductId: "test-product",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	row := orderItemRowFromDomain(item)
	// variant_id is nullable with a FK to variants; omit it (→ NULL) when not set
	// so the FK constraint is satisfied without needing a real variant parent.
	require.NoError(t, db.Omit("Price", "VariantId").Create(&row).Error)
	return item
}

func seedOrder(t *testing.T, db *gorm.DB, orgId, customerId string) domain.Order {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	// orders.cart_id is NOT NULL with a FK to carts(org_id, id). Seed a minimal cart.
	cartId := lib.GenerateId("cart")
	require.NoError(t, db.Create(&cartRow{
		OrgId:     orgId,
		Id:        cartId,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	o := domain.Order{
		OrgId:      orgId,
		Id:         lib.GenerateId("ord"),
		CustomerId: customerId,
		CartId:     cartId,
		Reference:  "REF-" + lib.GenerateId("r"),
		Status:     domain.OrderStatusPending,
		Currency:   "USD",
		Total:      1999,
		Metadata:   map[string]string{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	row := orderRowFromDomain(o)
	require.NoError(t, db.Omit("Customer", "Items").Create(&row).Error)
	return o
}

// seedCoupon inserts a valid percentage coupon for orgId and returns it.
func seedCoupon(t *testing.T, db *gorm.DB, orgId string) domain.Coupon {
	t.Helper()
	in, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:        orgId,
		Name:         "Seed Coupon",
		DiscountType: domain.DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     domain.DurationForever,
	})
	require.NoError(t, err)
	c, err := NewCouponRepo(db).Create(context.Background(), in)
	require.NoError(t, err)
	return c
}
