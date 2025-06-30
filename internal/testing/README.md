# Payloop Testing Framework

This testing framework provides structured, reusable testing utilities for the Payloop codebase, eliminating duplicate mock implementations and providing clean integration test patterns.

## Structure

```
internal/testing/
├── mocks/           # Reusable mock implementations
├── fixtures/        # Test data builders and factories
├── integration/     # Integration test framework
├── modules/         # FX test module configurations
└── README.md        # This file
```

## Key Features

- **Shared Mocks**: Single implementations of common mocks (Logger, PubSub, Repositories)
- **Test Fixtures**: Builder pattern for creating test entities with sensible defaults
- **Integration Framework**: FX-based integration testing with real dependencies
- **Modular Configuration**: Easy swapping between real and mock dependencies

## Usage

### 1. Unit Testing with Shared Mocks

Instead of creating mocks in each test file:

```go
// OLD - Don't do this
type MockLogger struct{}
func (m *MockLogger) Info(msg string, args ...interface{}) {}
// ... more boilerplate

// NEW - Use shared mocks
import "payloop/internal/testing/mocks"

func TestMyService(t *testing.T) {
    mockLogger := mocks.NewSilentLogger()
    mockPubSub := mocks.NewSilentPubSub()
    mockRepo := mocks.NewMockSubscriptionRepository()
    
    // Set up specific expectations for your test
    mockRepo.On("FindById", mock.Anything, "org1", "sub1").Return(subscription, nil)
    
    service := NewMyService(mockLogger, mockPubSub, mockRepo)
    // ... test logic
}
```

### 2. Creating Test Data with Fixtures

Use builders to create test entities with sensible defaults:

```go
import "payloop/internal/testing/fixtures"

func TestSubscriptionLogic(t *testing.T) {
    // Create a subscription with defaults
    subscription := fixtures.NewSubscriptionBuilder().Build()
    
    // Customize as needed
    premiumSubscription := fixtures.NewSubscriptionBuilder().
        WithAmount(4900).
        WithStatus(entities.SubscriptionStatusActive).
        WithBilling(prices.BillingIntervalMonth, 1).
        Build()
    
    customer := fixtures.NewCustomerBuilder().
        WithEmail("test@example.com").
        WithName("John", "Doe").
        Build()
}
```

### 3. Integration Testing

For testing with real database and FX dependency injection:

```go
import "payloop/internal/testing/integration"

func TestSubscriptionIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Create test suite with real database, mock pubsub
    suite := integration.NewSubscriptionTestSuite(t)
    defer suite.Cleanup()
    
    app := suite.App()
    ctx := suite.Context()
    orgId := suite.OrgId()
    
    // Create test data
    customer := fixtures.NewCustomerBuilder().WithOrgId(orgId).Build()
    createdCustomer, err := app.CustomerService.Create(ctx, customer)
    require.NoError(t, err)
    
    // Test subscription operations with real dependencies
    subscription := fixtures.NewSubscriptionBuilder().
        WithCustomerId(createdCustomer.Id).
        Build()
    
    // Test plan changes, lifecycle management, etc.
}
```

### 4. Custom Test Configurations

Configure which dependencies are real vs mocked:

```go
import "payloop/internal/testing/integration"

func TestWithCustomConfig(t *testing.T) {
    config := integration.TestConfig{
        UseRealDatabase: true,   // Real postgres
        UseMockPubSub:   true,   // Mock NATS
        UseMockLogger:   false,  // Real logger for debugging
    }
    
    app := integration.NewTestApp(t, config)
    defer app.Cleanup()
    
    // Test with your custom configuration
}
```

## Available Mocks

### Logger
- `mocks.NewMockLogger()` - Full mock with expectations
- `mocks.NewSilentLogger()` - Ignores all calls (recommended for most tests)

### PubSub
- `mocks.NewMockPubSub()` - Full mock with expectations
- `mocks.NewSilentPubSub()` - Ignores all calls (recommended for most tests)

### Repositories
- `mocks.NewMockSubscriptionRepository()`
- `mocks.NewMockCustomerRepository()`
- `mocks.NewMockVariantRepository()`
- `mocks.NewMockPriceRepository()`

## Test Builders

### Subscription
```go
subscription := fixtures.NewSubscriptionBuilder().
    WithOrgId("org_123").
    WithStatus(entities.SubscriptionStatusActive).
    WithAmount(2500).  // $25.00
    WithBilling(prices.BillingIntervalMonth, 1).
    WithMetadata("plan", "premium").
    Build()
```

### Customer
```go
customer := fixtures.NewCustomerBuilder().
    WithEmail("test@example.com").
    WithName("John", "Doe").
    Build()
```

### Variant & Price
```go
variant := fixtures.NewVariantBuilder().
    WithName("Premium Plan").
    Build()

price := fixtures.NewPriceBuilder().
    WithAmount(4900).
    WithLabel("Premium Monthly").
    Build()
```

### Plan Change
```go
planChange := fixtures.NewSubscriptionPlanChangeBuilder().
    WithSubscriptionId("sub_123").
    WithFromPrice("var_basic", "price_basic", 2500).
    WithToPrice("var_premium", "price_premium", 4900).
    WithChangeType("upgrade").
    Build()
```

## Running Tests

```bash
# Run unit tests only
go test ./... -short

# Run all tests including integration tests
go test ./...

# Run specific integration tests
go test -v ./internal/testing/integration

# Run with coverage
go test -cover ./...
```

## Best Practices

1. **Use shared mocks** instead of creating your own
2. **Use builders** for test data instead of manual struct creation
3. **Use integration tests** for critical business flows
4. **Mock external dependencies** (PubSub, external APIs) in integration tests
5. **Use real database** for integration tests to catch real issues
6. **Clean up** test data and applications properly
7. **Skip integration tests** in short mode for fast feedback

## Migration from Existing Tests

To migrate existing tests to use this framework:

1. **Replace manual mocks** with shared mocks from `internal/testing/mocks`
2. **Replace manual struct creation** with builders from `internal/testing/fixtures`
3. **Convert complex tests** to use the integration framework
4. **Remove duplicate mock definitions** from test files

## Future Enhancements

- Database seeding utilities
- More entity builders (Product, Order, Payment, etc.)
- Mock generators for custom interfaces
- Test data cleanup automation
- Performance testing utilities