package integration

import (
	"context"
	"testing"

	"payloop/internal/api/controllers"
	"payloop/internal/api/middlewares"
	"payloop/internal/api/routes"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/services"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/authn/apikey"
	"payloop/internal/infrastructure/authz/cedar"
	"payloop/internal/infrastructure/cache/redis"
	"payloop/internal/infrastructure/db/postgres"
	"payloop/internal/infrastructure/email/loops"
	"payloop/internal/infrastructure/payments/paystack"
	"payloop/internal/infrastructure/pubsub/nats"
	"payloop/internal/infrastructure/queue/sqs"
	"payloop/internal/infrastructure/scheduler/cron"
	"payloop/internal/infrastructure/storage/s3"
	"payloop/internal/infrastructure/vault/aes_vault"
	"payloop/internal/infrastructure/workflow/temporal"
	"payloop/internal/lib"
	"payloop/internal/lib/logger"
	"payloop/internal/lib/pubsub"
	"payloop/internal/testing/mocks"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// TestApp represents a test application instance
type TestApp struct {
	App                       *fxtest.App
	SubscriptionService       interfaces.SubscriptionService
	SubscriptionOrchestration interfaces.SubscriptionOrchestrationService
	CustomerService           interfaces.CustomerService
	VariantService            interfaces.VariantService
	PriceService              interfaces.PriceService
	ProductService            interfaces.ProductService
	OrderService              interfaces.OrderService
	Logger                    logger.Logger
}

// TestConfig holds configuration for integration tests
type TestConfig struct {
	UseRealDatabase bool
	UseMockPubSub   bool
	UseMockLogger   bool
}

// DefaultTestConfig returns a sensible default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		UseRealDatabase: true,  // Use real database for integration tests
		UseMockPubSub:   true,  // Mock pubsub to avoid external dependencies
		UseMockLogger:   false, // Use real logger for debugging
	}
}

// NewTestApp creates a new test application with the specified configuration
func NewTestApp(t *testing.T, config TestConfig) *TestApp {
	var app *fxtest.App
	var subscriptionService interfaces.SubscriptionService
	var customerService interfaces.CustomerService
	var variantService interfaces.VariantService
	var priceService interfaces.PriceService
	var productService interfaces.ProductService
	var orderService interfaces.OrderService
	var subscriptionOrchestration interfaces.SubscriptionOrchestrationService
	var testLogger logger.Logger

	// Build modules based on configuration
	modules := []fx.Option{
		// Core modules always needed
		lib.Module,
		services.Module,
		factories.Module,
		middlewares.Module,
		controllers.Module,
		routes.Module,
	}

	// Add database module
	if config.UseRealDatabase {
		modules = append(modules, postgres.Module)
	} else {
		// Could add a mock database module here in the future
		t.Skip("Mock database not yet implemented")
	}

	// Add vault module (required by many services)
	modules = append(modules, aes_vault.Module)

	// Add auth modules (required by API layer)
	modules = append(modules, 
		apikey.Module,
		cedar.Module,
	)

	// Add mock modules based on configuration
	if config.UseMockPubSub {
		// Replace real pubsub with mock
		modules = append(modules, fx.Decorate(func() pubsub.PubSub {
			return mocks.NewSilentPubSub()
		}))
	} else {
		modules = append(modules, nats.Module)
	}

	// Add other infrastructure modules with mocks as needed
	modules = append(modules,
		// These can be mocked in the future
		temporal.Module,
		redis.Module,
		sqs.Module,
		cron.Module,
		paystack.Module,
		loops.Module,
		s3.Module(),
	)

	// Extract services for easy access
	modules = append(modules, fx.Populate(
		&subscriptionService,
		&subscriptionOrchestration,
		&customerService,
		&variantService,
		&priceService,
		&productService,
		&orderService,
		&testLogger,
	))

	// Don't start the app lifecycle (no HTTP server, etc)
	modules = append(modules, fx.NopLogger)

	app = fxtest.New(t, modules...)

	return &TestApp{
		App:                          app,
		SubscriptionService:          subscriptionService,
		SubscriptionOrchestration:    subscriptionOrchestration,
		CustomerService:              customerService,
		VariantService:               variantService,
		PriceService:                 priceService,
		ProductService:               productService,
		OrderService:                 orderService,
		Logger:                       testLogger,
	}
}

// Cleanup shuts down the test application
func (ta *TestApp) Cleanup() {
	ta.App.RequireStop()
}

// WithContext creates a context for testing
func (ta *TestApp) WithContext() context.Context {
	return context.Background()
}

// SubscriptionTestSuite provides common setup for subscription-related tests
type SubscriptionTestSuite struct {
	t       *testing.T
	app     *TestApp
	ctx     context.Context
	orgId   string
}

// NewSubscriptionTestSuite creates a new subscription test suite
func NewSubscriptionTestSuite(t *testing.T) *SubscriptionTestSuite {
	config := DefaultTestConfig()
	app := NewTestApp(t, config)
	
	return &SubscriptionTestSuite{
		t:     t,
		app:   app,
		ctx:   app.WithContext(),
		orgId: "org_test_" + generateTestId(),
	}
}

// App returns the test application
func (s *SubscriptionTestSuite) App() *TestApp {
	return s.app
}

// Context returns the test context
func (s *SubscriptionTestSuite) Context() context.Context {
	return s.ctx
}

// OrgId returns the test organization ID
func (s *SubscriptionTestSuite) OrgId() string {
	return s.orgId
}

// Cleanup cleans up the test suite
func (s *SubscriptionTestSuite) Cleanup() {
	s.app.Cleanup()
}

// Helper function to generate unique test IDs
func generateTestId() string {
	return lib.GenerateId("test")
}

// TestDatabase provides utilities for database testing
type TestDatabase struct {
	app *TestApp
}

// NewTestDatabase creates a new test database instance
func NewTestDatabase(app *TestApp) *TestDatabase {
	return &TestDatabase{app: app}
}

// TODO: Add database seeding and cleanup methods here
// Examples:
// - SeedCustomer(customer entities.Customer)
// - SeedSubscription(subscription entities.Subscription)
// - CleanupOrg(orgId string)
// - CleanupAll()

// TestAsserter provides common assertions for integration tests
type TestAsserter struct {
	t *testing.T
}

// NewTestAsserter creates a new test asserter
func NewTestAsserter(t *testing.T) *TestAsserter {
	return &TestAsserter{t: t}
}

// AssertSubscriptionEquals asserts that two subscriptions are equal
func (a *TestAsserter) AssertSubscriptionEquals(expected, actual interface{}) {
	// TODO: Implement custom subscription comparison logic
	// This could include ignoring timestamps, IDs, etc.
	if expected != actual {
		a.t.Errorf("Subscriptions not equal. Expected: %+v, Actual: %+v", expected, actual)
	}
}

// AssertNoError asserts that an error is nil
func (a *TestAsserter) AssertNoError(err error) {
	if err != nil {
		a.t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError asserts that an error is not nil
func (a *TestAsserter) AssertError(err error) {
	if err == nil {
		a.t.Fatal("Expected an error, got nil")
	}
}