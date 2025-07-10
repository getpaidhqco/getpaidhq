package services_test

import (
	"context"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/cart"
	"payloop/internal/lib"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payloop/internal/application/interfaces"

	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/testing/database"
	"payloop/internal/testing/fixtures"
)

type testContext struct {
	db                         *database.TestDatabase
	billingService             interfaces.BillingService
	usageEventRepository       repositories.UsageEventRepository
	subscriptionRepository     repositories.SubscriptionRepository
	subscriptionItemRepository repositories.SubscriptionItemRepository
	priceRepository            repositories.PriceRepository
	meterRepository            repositories.MeterRepository
	productRepository          repositories.ProductRepository
	customerRepository         repositories.CustomerRepository
	variantRepository          repositories.VariantRepository
	orderRepository            repositories.OrderRepository
	orderItemRepository        repositories.OrderItemRepository
	cartRepository             repositories.CartRepository
	cartFactory                factories.CartFactory
	orgId                      string
	customerId                 string
	subscriptionId             string
}

func setupTestContext(t *testing.T) *testContext {
	// Setup test database with Prisma schema sync
	testDB := database.SetupTestDatabaseWithPrisma(t)

	// Create repositories with test database
	pgDB := &postgres.PgDatabase{Pool: testDB.Pool}
	log := lib.GetLogger()

	usageDB := &postgres.PgDatabase{Pool: testDB.Pool} // Same pool for testing
	usageEventRepo := postgres.NewUsageEventRepository(usageDB, log)

	subscriptionItemRepo := postgres.NewSubscriptionItemRepository(pgDB, log)
	subscriptionRepo := postgres.NewSubscriptionRepository(pgDB, log, subscriptionItemRepo)
	priceRepo := postgres.NewPriceRepository(pgDB, log)
	meterRepo := postgres.NewMeterRepository(pgDB, log)
	customerRepo := postgres.NewCustomerRepository(pgDB, log)
	variantRepo := postgres.NewVariantRepository(pgDB, log)
	productRepository := postgres.NewProductRepository(pgDB, log, priceRepo)
	orderItemRepo := postgres.NewOrderItemRepository(pgDB, log)
	orderRepo := postgres.NewOrderRepository(pgDB, log)
	cartRepo := postgres.NewCartRepository(pgDB, log)

	// Create factories
	settingRepo := postgres.NewSettingRepository(pgDB, log)
	cartFactory := factories.NewCartFactory(settingRepo, priceRepo, productRepository, variantRepo, cartRepo, log)

	// Create tier calculation service
	tierCalcService := services.NewTierCalculationService(priceRepo)

	// Create billing service with real dependencies
	billingService := services.NewBillingService(
		usageEventRepo,
		subscriptionRepo,
		subscriptionItemRepo,
		priceRepo,
		meterRepo,
		tierCalcService,
	)

	return &testContext{
		db:                         testDB,
		billingService:             billingService,
		usageEventRepository:       usageEventRepo,
		subscriptionRepository:     subscriptionRepo,
		subscriptionItemRepository: subscriptionItemRepo,
		priceRepository:            priceRepo,
		productRepository:          productRepository,
		meterRepository:            meterRepo,
		customerRepository:         customerRepo,
		variantRepository:          variantRepo,
		orderRepository:            orderRepo,
		orderItemRepository:        orderItemRepo,
		cartRepository:             cartRepo,
		cartFactory:                cartFactory,
		orgId:                      "test_org_123",
		customerId:                 "cus_test_123",
		subscriptionId:             "sub_test_123",
	}
}

func (tc *testContext) cleanup(t *testing.T) {
	tc.db.TruncateTables(t,
		"usage_events",
		"subscription_items",
		"subscriptions",
		"order_items",
		"orders",
		"carts",
		"prices",
		"meters",
		"customers",
		"orgs",
	)
	tc.db.Cleanup(t)
}

func (tc *testContext) seedOrganization(t *testing.T, ctx context.Context) {
	// Create organization
	_, err := tc.db.Pool.Exec(ctx, `
		INSERT INTO orgs (id, name, country, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, tc.orgId, "Test Organization", "US", time.Now(), time.Now())
	require.NoError(t, err)
}

func (tc *testContext) seedCustomer(t *testing.T, ctx context.Context) {
	tc.seedOrganization(t, ctx)

	// Create customer
	customer := entities.Customer{
		Id:        tc.customerId,
		OrgId:     tc.orgId,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "Customer",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err := tc.customerRepository.Create(ctx, customer)
	require.NoError(t, err)
}

func TestBillingServiceIntegration_PureUsageBilling(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup(t)

	ctx := context.Background()
	tc.seedCustomer(t, ctx)

	// Create meter with sum aggregation
	meter := entities.Meter{
		Id:              "meter_api_calls",
		OrgId:           tc.orgId,
		Name:            "API Calls",
		AggregationType: entities.AggregationTypeSum,
		UnitType:        entities.UnitTypeCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, err := tc.meterRepository.Create(ctx, meter)
	require.NoError(t, err)

	product, err := tc.productRepository.Create(ctx,
		entities.Product{
			OrgId:       tc.orgId,
			Id:          lib.GenerateId("prod"),
			Name:        "api calls",
			Description: "api calls product",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	require.NoError(t, err)

	// Create variant
	variant := entities.Variant{
		Id:          "variant_api_calls",
		OrgId:       tc.orgId,
		ProductId:   product.Id,
		Name:        "API Calls Variant",
		Description: "Usage based API calls pricing",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err = tc.variantRepository.Create(ctx, variant)
	require.NoError(t, err)

	// Create price
	price := entities.Price{
		Id:                 "price_api_calls",
		OrgId:              tc.orgId,
		VariantId:          variant.Id,
		MeterId:            meter.Id,
		BillingIntervalQty: 1,
		Category:           prices.PriceCategoryUsage,
		Label:              "API Calls Usage",
		Currency:           "USD",
		HasUsage:           true,
		BillingInterval:    prices.BillingIntervalMonth,
		UnitPrice:          100, // $1.00 per call
		Scheme:             prices.Fixed,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_, err = tc.priceRepository.Create(ctx, price)
	require.NoError(t, err)

	// Create cart
	cartInstance := tc.cartFactory.NewCart(tc.orgId, "USD")
	_, err = cartInstance.AddItem(ctx, cart.AddItemInput{
		ProductId: product.Id,
		PriceId:   price.Id,
		Quantity:  1,
	})
	require.NoError(t, err)

	// Save cart to database
	_, err = tc.cartRepository.Create(ctx, entities.Cart{
		OrgId:    tc.orgId,
		Id:       cartInstance.Id,
		Data:     cartInstance.CartData,
		Status:   string(entities.CartStatusPending),
		Total:    100,
		Metadata: nil,
	})
	require.NoError(t, err)

	// Create order
	orderId := lib.GenerateId("order")
	ref := time.Now().Format("20060102150405")
	order := entities.Order{
		OrgId:      tc.orgId,
		Id:         orderId,
		CustomerId: tc.customerId,
		CartId:     cartInstance.Id,
		Reference:  ref,
		Status:     entities.OrderStatusPending,
		Currency:   "USD",
		Total:      100,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdOrder, err := tc.orderRepository.Create(ctx, order)
	require.NoError(t, err)

	// Create order item
	orderItem := entities.OrderItem{
		OrgId:         tc.orgId,
		Id:            lib.GenerateId("order_item"),
		OrderId:       createdOrder.Id,
		ProductId:     product.Id,
		VariantId:     variant.Id,
		PriceId:       price.Id,
		Description:   "API Calls",
		Quantity:      1,
		TaxTotal:      0,
		DiscountTotal: 0,
		Subtotal:      100,
		Total:         100,
		Metadata:      map[string]string{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_, err = tc.orderItemRepository.Create(ctx, orderItem)
	require.NoError(t, err)

	// Create subscription
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	subscription := entities.Subscription{
		Id:                 tc.subscriptionId,
		OrgId:              tc.orgId,
		CustomerId:         tc.customerId,
		OrderId:            createdOrder.Id,
		Status:             entities.SubscriptionStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		BillingInterval:    prices.BillingIntervalMonth,
		BillingIntervalQty: 1,
		Amount:             0,
		OrderItemId:        orderItem.Id,
		Currency:           "USD",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_, err = tc.subscriptionRepository.Create(ctx, subscription)
	require.NoError(t, err)

	// Create subscription item
	subscriptionItem := entities.SubscriptionItem{
		Id:             "si_api_calls",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		PriceId:        price.Id,
		MeterId:        meter.Id,
		Status:         entities.SubscriptionItemStatusActive,
		Description:    "API Calls",
		Currency:       "USD",
		UnitPrice:      100,
		HasUsage:       true,
		Metadata: map[string]string{
			"price_category": "usage",
			"pricing_scheme": "fixed",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
	require.NoError(t, err)

	// Create usage events
	events := []entities.UsageEvent{
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(24 * time.Hour)).
			WithQuantity(10).
			Build(),
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(48 * time.Hour)).
			WithQuantity(15).
			Build(),
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(72 * time.Hour)).
			WithQuantity(5).
			Build(),
	}

	for _, event := range events {
		err := tc.usageEventRepository.Create(ctx, event)
		require.NoError(t, err)
	}

	// Calculate billing amount
	result, err := tc.billingService.CalculateBillingAmount(ctx, subscription)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, "USD", result.Currency)
	assert.Equal(t, int64(0), result.BaseAmount, "Base amount should be zero for usage billing")
	assert.Equal(t, int64(300), result.UsageAmount, "Usage amount") // 30 calls * $1.00 = $30.00
	assert.Equal(t, int64(300), result.TotalAmount, "Total amount")

	// Verify usage breakdown
	assert.Len(t, result.UsageBreakdown, 1)
	assert.Equal(t, subscriptionItem.Id, result.UsageBreakdown[0].SubscriptionItemId)
	assert.Equal(t, float64(3), result.UsageBreakdown[0].Quantity)
	assert.Equal(t, "count", result.UsageBreakdown[0].UnitType)
	assert.Equal(t, int64(300), result.UsageBreakdown[0].Amount)
}

func TestBillingServiceIntegration_HybridBilling(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup(t)

	ctx := context.Background()
	tc.seedCustomer(t, ctx)

	// Create meter
	meter := entities.Meter{
		Id:              "meter_seats",
		OrgId:           tc.orgId,
		Name:            "Active Seats",
		AggregationType: entities.AggregationTypeMax,
		UnitType:        entities.UnitTypeCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, err := tc.meterRepository.Create(ctx, meter)
	require.NoError(t, err)

	// Create subscription
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	subscription := entities.Subscription{
		Id:                 tc.subscriptionId,
		OrgId:              tc.orgId,
		CustomerId:         tc.customerId,
		Status:             entities.SubscriptionStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Currency:           "USD",
		Amount:             5000, // $50 base
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_, err = tc.subscriptionRepository.Create(ctx, subscription)
	require.NoError(t, err)

	// Create subscription item with hybrid pricing
	subscriptionItem := entities.SubscriptionItem{
		Id:             "si_seats",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		MeterId:        meter.Id,
		Description:    "Team Plan - 5 seats included",
		Currency:       "USD",
		Amount:         5000, // $50 base includes 5 seats
		UnitPrice:      1000, // $10 per additional seat
		HasUsage:       true,
		Metadata: map[string]string{
			"price_category":     "hybrid",
			"pricing_scheme":     "fixed",
			"included_usage":     "5",
			"overage_unit_price": "1000",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
	require.NoError(t, err)

	// Create usage events showing max of 8 seats
	events := []entities.UsageEvent{
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(24 * time.Hour)).
			WithQuantity(3).
			Build(),
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(48 * time.Hour)).
			WithQuantity(8). // Peak usage
			Build(),
		fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
			WithMeterId(meter.Id).
			WithTime(periodStart.Add(72 * time.Hour)).
			WithQuantity(6).
			Build(),
	}

	for _, event := range events {
		err := tc.usageEventRepository.Create(ctx, event)
		require.NoError(t, err)
	}

	// Calculate billing amount
	result, err := tc.billingService.CalculateBillingAmount(ctx, subscription)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, int64(5000), result.BaseAmount)  // $50 base
	assert.Equal(t, int64(3000), result.UsageAmount) // 3 overage seats * $10 = $30
	assert.Equal(t, int64(8000), result.TotalAmount) // $80 total

	// Verify item breakdown
	assert.Len(t, result.ItemBreakdown, 1)
	assert.Equal(t, "hybrid", result.ItemBreakdown[0].PriceCategory)
	assert.Equal(t, int64(8000), result.ItemBreakdown[0].Amount)
}

func TestBillingServiceIntegration_TableDrivenScenarios(t *testing.T) {
	scenarios := []struct {
		name           string
		setupFunc      func(*testing.T, *testContext, context.Context)
		expectedResult interfaces.BillingCalculation
	}{
		{
			name: "No usage events",
			setupFunc: func(t *testing.T, tc *testContext, ctx context.Context) {
				// Create meter
				meter := entities.Meter{
					Id:              "meter_empty",
					OrgId:           tc.orgId,
					Name:            "Empty Meter",
					AggregationType: entities.AggregationTypeSum,
					UnitType:        entities.UnitTypeCount,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				_, err := tc.meterRepository.Create(ctx, meter)
				require.NoError(t, err)

				// Create subscription item
				subscriptionItem := entities.SubscriptionItem{
					Id:             "si_empty",
					OrgId:          tc.orgId,
					SubscriptionId: tc.subscriptionId,
					MeterId:        meter.Id,
					Description:    "No Usage",
					Currency:       "USD",
					UnitPrice:      100,
					HasUsage:       true,
					Metadata: map[string]string{
						"price_category": "usage",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
				require.NoError(t, err)
			},
			expectedResult: interfaces.BillingCalculation{
				Currency:    "USD",
				BaseAmount:  0,
				UsageAmount: 0,
				TotalAmount: 0,
			},
		},
		{
			name: "Average aggregation",
			setupFunc: func(t *testing.T, tc *testContext, ctx context.Context) {
				// Create meter with average aggregation
				meter := entities.Meter{
					Id:              "meter_avg",
					OrgId:           tc.orgId,
					Name:            "Average Usage",
					AggregationType: entities.AggregationTypeAverage,
					UnitType:        entities.UnitTypeCount,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				_, err := tc.meterRepository.Create(ctx, meter)
				require.NoError(t, err)

				// Create subscription item
				subscriptionItem := entities.SubscriptionItem{
					Id:             "si_avg",
					OrgId:          tc.orgId,
					SubscriptionId: tc.subscriptionId,
					MeterId:        meter.Id,
					Description:    "Average Usage",
					Currency:       "USD",
					UnitPrice:      100,
					HasUsage:       true,
					Metadata: map[string]string{
						"price_category": "usage",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
				require.NoError(t, err)

				// Create events with quantities: 10, 20, 30 (average = 20)
				now := time.Now()
				periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

				events := []entities.UsageEvent{
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(24 * time.Hour)).
						WithQuantity(10).
						Build(),
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(48 * time.Hour)).
						WithQuantity(20).
						Build(),
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(72 * time.Hour)).
						WithQuantity(30).
						Build(),
				}

				for _, event := range events {
					err := tc.usageEventRepository.Create(ctx, event)
					require.NoError(t, err)
				}
			},
			expectedResult: interfaces.BillingCalculation{
				Currency:    "USD",
				BaseAmount:  0,
				UsageAmount: 2000, // Average of 20 * $1.00 = $20.00
				TotalAmount: 2000,
			},
		},
		{
			name: "Last during period aggregation",
			setupFunc: func(t *testing.T, tc *testContext, ctx context.Context) {
				// Create meter with last_during_period aggregation
				meter := entities.Meter{
					Id:              "meter_last",
					OrgId:           tc.orgId,
					Name:            "Last Value",
					AggregationType: entities.AggregationTypeLastDuringPeriod,
					UnitType:        entities.UnitTypeCount,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				_, err := tc.meterRepository.Create(ctx, meter)
				require.NoError(t, err)

				// Create subscription item
				subscriptionItem := entities.SubscriptionItem{
					Id:             "si_last",
					OrgId:          tc.orgId,
					SubscriptionId: tc.subscriptionId,
					MeterId:        meter.Id,
					Description:    "Last Value Usage",
					Currency:       "USD",
					UnitPrice:      100,
					HasUsage:       true,
					Metadata: map[string]string{
						"price_category": "usage",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
				require.NoError(t, err)

				// Create events - last one has quantity 50
				now := time.Now()
				periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

				events := []entities.UsageEvent{
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(24 * time.Hour)).
						WithQuantity(10).
						Build(),
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(48 * time.Hour)).
						WithQuantity(20).
						Build(),
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(72 * time.Hour)).
						WithQuantity(50). // This should be the value used
						Build(),
				}

				for _, event := range events {
					err := tc.usageEventRepository.Create(ctx, event)
					require.NoError(t, err)
				}
			},
			expectedResult: interfaces.BillingCalculation{
				Currency:    "USD",
				BaseAmount:  0,
				UsageAmount: 5000, // Last value of 50 * $1.00 = $50.00
				TotalAmount: 5000,
			},
		},
		{
			name: "Events outside billing period",
			setupFunc: func(t *testing.T, tc *testContext, ctx context.Context) {
				// Create meter
				meter := entities.Meter{
					Id:              "meter_outside",
					OrgId:           tc.orgId,
					Name:            "Outside Period",
					AggregationType: entities.AggregationTypeSum,
					UnitType:        entities.UnitTypeCount,
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				_, err := tc.meterRepository.Create(ctx, meter)
				require.NoError(t, err)

				// Create subscription item
				subscriptionItem := entities.SubscriptionItem{
					Id:             "si_outside",
					OrgId:          tc.orgId,
					SubscriptionId: tc.subscriptionId,
					MeterId:        meter.Id,
					Description:    "Outside Period Usage",
					Currency:       "USD",
					UnitPrice:      100,
					HasUsage:       true,
					Metadata: map[string]string{
						"price_category": "usage",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
				require.NoError(t, err)

				// Create events outside the billing period
				now := time.Now()
				periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

				events := []entities.UsageEvent{
					// Before period
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(-48 * time.Hour)).
						WithQuantity(100).
						Build(),
					// After period
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.AddDate(0, 2, 0)).
						WithQuantity(200).
						Build(),
					// Within period
					fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, subscriptionItem.Id).
						WithMeterId(meter.Id).
						WithTime(periodStart.Add(24 * time.Hour)).
						WithQuantity(25).
						Build(),
				}

				for _, event := range events {
					err := tc.usageEventRepository.Create(ctx, event)
					require.NoError(t, err)
				}
			},
			expectedResult: interfaces.BillingCalculation{
				Currency:    "USD",
				BaseAmount:  0,
				UsageAmount: 2500, // Only the 25 within period * $1.00
				TotalAmount: 2500,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tc := setupTestContext(t)
			defer tc.cleanup(t)

			ctx := context.Background()
			tc.seedCustomer(t, ctx)

			// Create base subscription
			now := time.Now()
			periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			periodEnd := periodStart.AddDate(0, 1, 0)

			subscription := entities.Subscription{
				Id:                 tc.subscriptionId,
				OrgId:              tc.orgId,
				CustomerId:         tc.customerId,
				Status:             entities.SubscriptionStatusActive,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				Currency:           "USD",
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			_, err := tc.subscriptionRepository.Create(ctx, subscription)
			require.NoError(t, err)

			// Run scenario setup
			scenario.setupFunc(t, tc, ctx)

			// Calculate billing amount
			result, err := tc.billingService.CalculateBillingAmount(ctx, subscription)
			require.NoError(t, err)

			// Assertions
			assert.Equal(t, scenario.expectedResult.Currency, result.Currency)
			assert.Equal(t, scenario.expectedResult.BaseAmount, result.BaseAmount)
			assert.Equal(t, scenario.expectedResult.UsageAmount, result.UsageAmount)
			assert.Equal(t, scenario.expectedResult.TotalAmount, result.TotalAmount)
		})
	}
}

func TestBillingServiceIntegration_PercentageBilling(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup(t)

	ctx := context.Background()
	tc.seedCustomer(t, ctx)

	// Create meter for transaction events
	meter := entities.Meter{
		Id:              "meter_transactions",
		OrgId:           tc.orgId,
		Name:            "Transactions",
		AggregationType: entities.AggregationTypeSum,
		UnitType:        entities.UnitTypeCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, err := tc.meterRepository.Create(ctx, meter)
	require.NoError(t, err)

	// Create subscription
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	subscription := entities.Subscription{
		Id:                 tc.subscriptionId,
		OrgId:              tc.orgId,
		CustomerId:         tc.customerId,
		Status:             entities.SubscriptionStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Currency:           "USD",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_, err = tc.subscriptionRepository.Create(ctx, subscription)
	require.NoError(t, err)

	// Create subscription item with percentage pricing
	subscriptionItem := entities.SubscriptionItem{
		Id:             "si_percentage",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		MeterId:        meter.Id,
		Description:    "Transaction Fee - 2.5%",
		Currency:       "USD",
		PercentageRate: 2.5, // 2.5%
		HasUsage:       true,
		Metadata: map[string]string{
			"price_category": "usage",
			"pricing_scheme": "percentage",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, subscriptionItem)
	require.NoError(t, err)

	// Create transaction events
	transactions := []struct {
		Time  time.Time
		Value float64
	}{
		{periodStart.Add(24 * time.Hour), 1000.00}, // $1,000
		{periodStart.Add(48 * time.Hour), 2500.50}, // $2,500.50
		{periodStart.Add(72 * time.Hour), 500.00},  // $500
	}

	events := fixtures.CreateTransactionEvents(tc.orgId, tc.subscriptionId, subscriptionItem.Id, transactions)

	for _, event := range events {
		event.MeterId = meter.Id
		err := tc.usageEventRepository.Create(ctx, event)
		require.NoError(t, err)
	}

	// Calculate billing amount
	result, err := tc.billingService.CalculateBillingAmount(ctx, subscription)
	require.NoError(t, err)

	// Assertions
	// Total transaction value: $4,000.50
	// 2.5% of $4,000.50 = $100.01 (rounded to cents)
	assert.Equal(t, int64(0), result.BaseAmount)
	assert.Equal(t, int64(100), result.UsageAmount) // $100.01 rounds to $100
	assert.Equal(t, int64(100), result.TotalAmount)
}

func TestBillingServiceIntegration_MultipleSubscriptionItems(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup(t)

	ctx := context.Background()
	tc.seedCustomer(t, ctx)

	// Create meters
	meterAPI := entities.Meter{
		Id:              "meter_api",
		OrgId:           tc.orgId,
		Name:            "API Calls",
		AggregationType: entities.AggregationTypeSum,
		UnitType:        entities.UnitTypeCount,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, err := tc.meterRepository.Create(ctx, meterAPI)
	require.NoError(t, err)

	meterStorage := entities.Meter{
		Id:              "meter_storage",
		OrgId:           tc.orgId,
		Name:            "Storage GB",
		AggregationType: entities.AggregationTypeMax,
		UnitType:        entities.UnitTypeGB,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, err = tc.meterRepository.Create(ctx, meterStorage)
	require.NoError(t, err)

	// Create subscription
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	subscription := entities.Subscription{
		Id:                 tc.subscriptionId,
		OrgId:              tc.orgId,
		CustomerId:         tc.customerId,
		Status:             entities.SubscriptionStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Currency:           "USD",
		Amount:             2000, // $20 base subscription
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	_, err = tc.subscriptionRepository.Create(ctx, subscription)
	require.NoError(t, err)

	// Create multiple subscription items
	// 1. Base subscription (traditional)
	siBase := entities.SubscriptionItem{
		Id:             "si_base",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		Description:    "Base Plan",
		Currency:       "USD",
		Amount:         2000, // $20
		HasUsage:       false,
		Metadata: map[string]string{
			"price_category": "subscription",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, siBase)
	require.NoError(t, err)

	// 2. API usage (pure usage)
	siAPI := entities.SubscriptionItem{
		Id:             "si_api",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		MeterId:        meterAPI.Id,
		Description:    "API Calls",
		Currency:       "USD",
		UnitPrice:      10, // $0.10 per call
		HasUsage:       true,
		Metadata: map[string]string{
			"price_category": "usage",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, siAPI)
	require.NoError(t, err)

	// 3. Storage (hybrid - 10GB included)
	siStorage := entities.SubscriptionItem{
		Id:             "si_storage",
		OrgId:          tc.orgId,
		SubscriptionId: tc.subscriptionId,
		MeterId:        meterStorage.Id,
		Description:    "Storage - 10GB included",
		Currency:       "USD",
		Amount:         1000, // $10 for 10GB
		UnitPrice:      200,  // $2 per GB overage
		HasUsage:       true,
		Metadata: map[string]string{
			"price_category":     "hybrid",
			"included_usage":     "10",
			"overage_unit_price": "200",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = tc.subscriptionItemRepository.Create(ctx, siStorage)
	require.NoError(t, err)

	// Create usage events
	// API calls: 100 total
	for i := 0; i < 5; i++ {
		event := fixtures.NewUsageEventBuilder(tc.orgId, tc.subscriptionId, siAPI.Id).
			WithMeterId(meterAPI.Id).
			WithTime(periodStart.Add(time.Duration(i*24) * time.Hour)).
			WithQuantity(20).
			Build()
		err := tc.usageEventRepository.Create(ctx, event)
		require.NoError(t, err)
	}

	// Storage: max 15GB
	storageEvents := fixtures.CreateMaxUsageEvents(
		tc.orgId, tc.subscriptionId, siStorage.Id,
		periodStart, []float64{8, 12, 15, 10, 13},
	)
	for _, event := range storageEvents {
		event.MeterId = meterStorage.Id
		err := tc.usageEventRepository.Create(ctx, event)
		require.NoError(t, err)
	}

	// Calculate billing amount
	result, err := tc.billingService.CalculateBillingAmount(ctx, subscription)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, int64(3000), result.BaseAmount)  // $20 base + $10 storage base = $30
	assert.Equal(t, int64(2000), result.UsageAmount) // 100 API calls * $0.10 + 5GB overage * $2 = $10 + $10 = $20
	assert.Equal(t, int64(5000), result.TotalAmount) // $50 total

	// Verify item breakdown
	assert.Len(t, result.ItemBreakdown, 3)

	// Find and verify each item
	for _, item := range result.ItemBreakdown {
		switch item.SubscriptionItemId {
		case "si_base":
			assert.Equal(t, "subscription", item.PriceCategory)
			assert.Equal(t, int64(2000), item.Amount)
		case "si_api":
			assert.Equal(t, "usage", item.PriceCategory)
			assert.Equal(t, int64(1000), item.Amount) // 100 calls * $0.10
		case "si_storage":
			assert.Equal(t, "hybrid", item.PriceCategory)
			assert.Equal(t, int64(2000), item.Amount) // $10 base + $10 overage
		}
	}

	// Verify usage breakdown
	assert.Len(t, result.UsageBreakdown, 2) // API and Storage
}
