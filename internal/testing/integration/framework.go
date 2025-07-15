package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"payloop/internal/application/lib/logger"
	"payloop/internal/infrastructure/events/nats"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
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
	"payloop/internal/infrastructure/queue/sqs"
	"payloop/internal/infrastructure/scheduler/cron"
	"payloop/internal/infrastructure/storage/s3"
	"payloop/internal/infrastructure/vault/aes_vault"
	"payloop/internal/infrastructure/workflow/temporal"
	"payloop/internal/lib"
)

// TestApp represents a test application instance
type TestApp struct {
	App                       *fxtest.App
	SubscriptionService       interfaces.SubscriptionService
	SubscriptionOrchestration interfaces.SubscriptionOrchestrationService
	CustomerService           interfaces.CustomerService
	ProductService            interfaces.ProductService
	BillingService            interfaces.BillingService
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
		UseMockPubSub:   false, // Mock pubsub to avoid external dependencies
		UseMockLogger:   false, // Use real logger for debugging
	}
}

// NewTestApp creates a new test application with the specified configuration
func NewTestApp(t *testing.T, config TestConfig) *TestApp {
	var app *fxtest.App
	var subscriptionService interfaces.SubscriptionService
	var customerService interfaces.CustomerService
	var productService interfaces.ProductService
	var billingService interfaces.BillingService
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
		cedar.Module)

	modules = append(modules, nats.Module)

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
		&productService,
		&orderService,
		&billingService,
		&testLogger,
	))

	// Don't start the app lifecycle (no HTTP server, etc)
	modules = append(modules, fx.NopLogger)

	app = fxtest.New(t, modules...)

	return &TestApp{
		App:                       app,
		SubscriptionService:       subscriptionService,
		SubscriptionOrchestration: subscriptionOrchestration,
		CustomerService:           customerService,
		ProductService:            productService,
		OrderService:              orderService,
		BillingService:            billingService,
		Logger:                    testLogger,
	}
}

// Cleanup shuts down the test application
func (ta *TestApp) Cleanup() {
	ta.App.RequireStop()
}

// WithContext creates a context for testing with a default timeout
func (ta *TestApp) WithContext() context.Context {
	// Create a context with a 30-second timeout for tests
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx
}

// SubscriptionTestSuite provides common setup for subscription-related tests
type SubscriptionTestSuite struct {
	t     *testing.T
	app   *TestApp
	ctx   context.Context
	orgId string
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
	// Generate a unique ID with "test" prefix
	id := lib.GenerateId("test")

	// If the ID is empty for some reason, fall back to a timestamp-based ID
	if id == "" {
		return fmt.Sprintf("test_%d", time.Now().UnixNano())
	}

	return id
}

// TestDatabase provides utilities for database testing
type TestDatabase struct {
	app *TestApp
	ctx context.Context
}

// NewTestDatabase creates a new test database instance
func NewTestDatabase(app *TestApp) *TestDatabase {
	return &TestDatabase{
		app: app,
		ctx: app.WithContext(),
	}
}

// SeedCustomer seeds a customer entity into the database for testing
func (db *TestDatabase) SeedCustomer(orgId string, customer interface{}) error {
	// This is a simplified implementation
	// In a real implementation, you would use the repository to create the customer
	fmt.Printf("Seeding customer for testing, orgId: %s\n", orgId)

	// Example implementation using customer service
	// The actual implementation would depend on the specific repository structure
	// customerEntity := mapToCustomerEntity(customer, orgId)
	// _, err := db.app.CustomerService.Create(db.ctx, orgId, customerEntity)
	// return err

	return nil // Placeholder
}

// SeedSubscription seeds a subscription entity into the database for testing
func (db *TestDatabase) SeedSubscription(orgId string, subscription interface{}) error {
	// This is a simplified implementation
	fmt.Printf("Seeding subscription for testing, orgId: %s\n", orgId)

	// Example implementation
	// subscriptionEntity := mapToSubscriptionEntity(subscription, orgId)
	// _, err := db.app.SubscriptionService.Create(db.ctx, orgId, subscriptionEntity)
	// return err

	return nil // Placeholder
}

// CleanupOrg removes all test data for a specific organization
func (db *TestDatabase) CleanupOrg(orgId string) error {
	if !strings.HasPrefix(orgId, "org_test_") {
		return fmt.Errorf("refusing to clean up non-test org ID: %s", orgId)
	}

	fmt.Printf("Cleaning up test data for organization, orgId: %s\n", orgId)

	// In a real implementation, you would execute database queries to delete test data
	// Example: DELETE FROM customers WHERE org_id = $1 AND id LIKE 'test_%'

	return nil // Placeholder
}

// CleanupAll removes all test data across all test organizations
func (db *TestDatabase) CleanupAll() error {
	fmt.Println("Cleaning up all test data")

	// In a real implementation, you would execute database queries to delete all test data
	// Example: DELETE FROM customers WHERE org_id LIKE 'org_test_%'

	return nil // Placeholder
}

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
	// Convert interfaces to maps for flexible comparison
	expectedMap, ok1 := a.toMap(expected)
	actualMap, ok2 := a.toMap(actual)

	if !ok1 || !ok2 {
		a.t.Fatalf("Failed to convert subscriptions to maps for comparison")
		return
	}

	// Fields to ignore in comparison (timestamps, auto-generated IDs)
	ignoreFields := map[string]bool{
		"CreatedAt": true,
		"UpdatedAt": true,
		// Add other fields to ignore as needed
	}

	// Compare maps, ignoring specified fields
	for key, expectedValue := range expectedMap {
		if ignoreFields[key] {
			continue
		}

		actualValue, exists := actualMap[key]
		if !exists {
			a.t.Errorf("Field %s missing in actual subscription", key)
			continue
		}

		if fmt.Sprintf("%v", expectedValue) != fmt.Sprintf("%v", actualValue) {
			a.t.Errorf("Field %s not equal. Expected: %v, Actual: %v", key, expectedValue, actualValue)
		}
	}
}

// toMap converts an interface to a map for flexible comparison
func (a *TestAsserter) toMap(obj interface{}) (map[string]interface{}, bool) {
	// Try to convert to map directly if it's already a map
	if m, ok := obj.(map[string]interface{}); ok {
		return m, true
	}

	// Otherwise, convert to JSON and back to map
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, false
	}

	return result, true
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
