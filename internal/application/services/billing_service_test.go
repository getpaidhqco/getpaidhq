package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/testing/mocks"
	"testing"
	"time"
)

// Mock implementations

type MockUsageRecordRepository struct {
	usageSummaries map[string]map[string]interface{}
}

func NewMockUsageRecordRepository() *MockUsageRecordRepository {
	return &MockUsageRecordRepository{
		usageSummaries: make(map[string]map[string]interface{}),
	}
}

func (m *MockUsageRecordRepository) FindById(ctx context.Context, orgId string, id string) (entities.UsageRecord, error) {
	return entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) Create(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	return entity, nil
}

func (m *MockUsageRecordRepository) Update(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	return entity, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionItemId(ctx context.Context, orgId string, subscriptionItemId string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindByBillingPeriod(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindUnprocessed(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) MarkProcessed(ctx context.Context, orgId string, ids []string, invoiceId string) error {
	return nil
}

func (m *MockUsageRecordRepository) AggregateUsage(ctx context.Context, orgId string, subscriptionItemId string, billingPeriod string, aggregationType string) (float64, error) {
	return 0, nil
}

func (m *MockUsageRecordRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.UsageRecord, int, error) {
	return []entities.UsageRecord{}, 0, nil
}

func (m *MockUsageRecordRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockUsageRecordRepository) BatchCreate(ctx context.Context, entities []entities.UsageRecord) ([]entities.UsageRecord, error) {
	return entities, nil
}

func (m *MockUsageRecordRepository) GetUsageSummary(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) (map[string]interface{}, error) {
	key := orgId + ":" + subscriptionItemId
	if summary, exists := m.usageSummaries[key]; exists {
		return summary, nil
	}
	return map[string]interface{}{
		"quantity": float64(0),
	}, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionItem(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) ([]entities.UsageRecord, error) {
	// Return empty slice for tests that don't explicitly set up usage records
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) SetUsageSummary(orgId string, subscriptionItemId string, summary map[string]interface{}) {
	key := orgId + ":" + subscriptionItemId
	m.usageSummaries[key] = summary
}

type MockSubscriptionItemRepository struct{}

func (m *MockSubscriptionItemRepository) FindById(ctx context.Context, orgId string, id string) (entities.SubscriptionItem, error) {
	return entities.SubscriptionItem{}, nil
}

func (m *MockSubscriptionItemRepository) Create(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	return entity, nil
}

func (m *MockSubscriptionItemRepository) Update(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	return entity, nil
}

func (m *MockSubscriptionItemRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionItem, error) {
	return []entities.SubscriptionItem{}, nil
}

func (m *MockSubscriptionItemRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockSubscriptionItemRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.SubscriptionItem, int, error) {
	return []entities.SubscriptionItem{}, 0, nil
}

type MockPriceRepository struct{}

func (m *MockPriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	return entities.Price{}, nil
}

func (m *MockPriceRepository) Create(ctx context.Context, entity entities.Price) (entities.Price, error) {
	return entity, nil
}

func (m *MockPriceRepository) Update(ctx context.Context, entity entities.Price) (entities.Price, error) {
	return entity, nil
}

func (m *MockPriceRepository) FindByVariantId(ctx context.Context, orgId string, variantId string, p request.Pagination) ([]entities.Price, int, error) {
	return []entities.Price{}, 0, nil
}

func (m *MockPriceRepository) GetPriceTiers(ctx context.Context, orgId string, priceId string) ([]entities.PriceTier, error) {
	return []entities.PriceTier{}, nil
}

func (m *MockPriceRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockPriceRepository) CreatePriceTiers(ctx context.Context, tiers []entities.PriceTier) error {
	return nil
}

func (m *MockPriceRepository) UpdatePriceTiers(ctx context.Context, orgId string, priceId string, tiers []entities.PriceTier) error {
	return nil
}

func (m *MockPriceRepository) DeletePriceTiers(ctx context.Context, orgId string, priceId string) error {
	return nil
}

// We'll use the real TierCalculationService with a mock PriceRepository
func createMockTierCalculationService(mockPriceRepo *MockPriceRepository) *TierCalculationService {
	return NewTierCalculationService(mockPriceRepo).(*TierCalculationService)
}

type MockDiscountService struct{}

func (m *MockDiscountService) CalculateDiscount(ctx context.Context, orgId string, subscription entities.Subscription, amount int64) (int64, error) {
	return 0, nil
}

// Helper functions for creating test data

func createTestSubscription(orgId string) entities.Subscription {
	now := time.Now()
	return entities.Subscription{
		OrgId:              orgId,
		Id:                 "test-subscription",
		Amount:             1000, // $10.00
		Currency:           "USD",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0), // 1 month later
	}
}

func createTestSubscriptionWithItems(orgId string, items []entities.SubscriptionItem) entities.Subscription {
	sub := createTestSubscription(orgId)
	sub.Items = items
	return sub
}

func createTestSubscriptionItem(orgId, subscriptionId, id string, amount int64, hasUsage bool) entities.SubscriptionItem {
	item := entities.SubscriptionItem{
		OrgId:          orgId,
		Id:             id,
		SubscriptionId: subscriptionId,
		Name:           "Test Item",
		Description:    "Test Description",
		Status:         entities.SubscriptionItemStatusActive,
		Amount:         amount,
		Currency:       "USD",
		HasUsage:       hasUsage,
	}

	if hasUsage {
		item.UnitPrice = 100 // $1.00 per unit
		item.UsageType = entities.UsageTypeMetered
		item.UnitType = entities.UnitTypeCount
		item.AggregationType = entities.AggregationTypeSum
	}

	return item
}

// Test cases

func TestCalculateBillingAmount_NoItems(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test data
	orgId := "test-org"
	subscription := createTestSubscription(orgId)

	// Execute
	calculation, err := billingService.CalculateBillingAmount(ctx, subscription)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, subscription.Amount, calculation.BaseAmount)
	assert.Equal(t, subscription.Amount, calculation.TotalAmount)
	assert.Equal(t, subscription.Currency, calculation.Currency)
	assert.Empty(t, calculation.ItemBreakdown)
}

func TestCalculateBillingAmount_WithItems(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test data
	orgId := "test-org"
	subscriptionId := "test-subscription"

	items := []entities.SubscriptionItem{
		createTestSubscriptionItem(orgId, subscriptionId, "item-1", 500, false),
		createTestSubscriptionItem(orgId, subscriptionId, "item-2", 300, false),
	}

	subscription := createTestSubscriptionWithItems(orgId, items)

	// Execute
	calculation, err := billingService.CalculateBillingAmount(ctx, subscription)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(800), calculation.BaseAmount) // 500 + 300
	assert.Equal(t, int64(800), calculation.TotalAmount)
	assert.Equal(t, subscription.Currency, calculation.Currency)
	assert.Len(t, calculation.ItemBreakdown, 2)
}

func TestCalculateBillingAmount_WithUsageItems(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test data
	orgId := "test-org"
	subscriptionId := "test-subscription"

	// Create a usage-based item
	usageItem := createTestSubscriptionItem(orgId, subscriptionId, "item-usage", 0, true)

	// Set up usage summary for this item
	mockUsageRecordRepo.SetUsageSummary(orgId, usageItem.Id, map[string]interface{}{
		"quantity": float64(5), // 5 units
	})

	items := []entities.SubscriptionItem{
		createTestSubscriptionItem(orgId, subscriptionId, "item-1", 500, false), // Fixed price item
		usageItem, // Usage-based item
	}

	subscription := createTestSubscriptionWithItems(orgId, items)

	// Execute
	calculation, err := billingService.CalculateBillingAmount(ctx, subscription)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(500), calculation.BaseAmount)  // Fixed price item
	assert.Equal(t, int64(500), calculation.TotalAmount) // Fixed price item (usage not added to total in current implementation)
	assert.Equal(t, subscription.Currency, calculation.Currency)
	assert.Len(t, calculation.ItemBreakdown, 2)

	// Check that the usage item has the correct amount
	var usageItemAmount int64
	for _, item := range calculation.ItemBreakdown {
		if item.SubscriptionItemId == "item-usage" {
			usageItemAmount = item.Amount
		}
	}
	assert.Equal(t, int64(500), usageItemAmount) // 5 units * $1.00 per unit
}

func TestCalculateTraditionalAmount(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test cases
	tests := []struct {
		name           string
		subscription   entities.Subscription
		expectedAmount int64
		expectError    bool
	}{
		{
			name:           "No items",
			subscription:   createTestSubscription("test-org"),
			expectedAmount: 1000, // Subscription amount
			expectError:    false,
		},
		{
			name: "With non-usage items",
			subscription: createTestSubscriptionWithItems("test-org", []entities.SubscriptionItem{
				createTestSubscriptionItem("test-org", "test-subscription", "item-1", 500, false),
				createTestSubscriptionItem("test-org", "test-subscription", "item-2", 300, false),
			}),
			expectedAmount: 800, // 500 + 300
			expectError:    false,
		},
		{
			name: "With mixed items",
			subscription: createTestSubscriptionWithItems("test-org", []entities.SubscriptionItem{
				createTestSubscriptionItem("test-org", "test-subscription", "item-1", 500, false),
				createTestSubscriptionItem("test-org", "test-subscription", "item-2", 0, true),
			}),
			expectedAmount: 500, // Only non-usage items
			expectError:    false,
		},
		{
			name:           "OrgId mismatch",
			subscription:   createTestSubscription("test-org"),
			expectedAmount: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := billingService.CalculateTraditionalAmount(ctx, tt.subscription)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAmount, amount)
			}
		})
	}
}

func TestCalculateUsageAmount(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test data
	orgId := "test-org"
	subscriptionId := "test-subscription"

	// Create a usage-based item
	usageItem := createTestSubscriptionItem(orgId, subscriptionId, "item-usage", 0, true)

	// Set up usage summary for this item
	mockUsageRecordRepo.SetUsageSummary(orgId, usageItem.Id, map[string]interface{}{
		"quantity": float64(5), // 5 units
	})

	subscription := createTestSubscriptionWithItems(orgId, []entities.SubscriptionItem{
		createTestSubscriptionItem(orgId, subscriptionId, "item-1", 500, false), // Fixed price item
		usageItem, // Usage-based item
	})

	// Create billing period
	period := interfaces.BillingPeriod{
		StartDate: subscription.CurrentPeriodStart,
		EndDate:   subscription.CurrentPeriodEnd,
	}

	// Execute
	amount, err := billingService.CalculateUsageAmount(ctx, subscription, period)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(500), amount) // 5 units * $1.00 per unit
}

func TestCalculateHybridAmount(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockUsageRecordRepo := mocks.NewMockUsageRecordRepository()
	mockSubscriptionItemRepo := mocks.NewMockSubscriptionItemRepository()
	mockPriceRepo := &MockPriceRepository{}
	mockTierCalculationService := createMockTierCalculationService(mockPriceRepo)

	billingService := NewBillingService(
		mockUsageRecordRepo,
		mockSubscriptionItemRepo,
		mockPriceRepo,
		mockTierCalculationService,
	)

	// Test data
	orgId := "test-org"
	subscriptionId := "test-subscription"

	// Create a usage-based item
	usageItem := createTestSubscriptionItem(orgId, subscriptionId, "item-usage", 0, true)

	// Set up usage summary for this item
	mockUsageRecordRepo.SetUsageSummary(orgId, usageItem.Id, map[string]interface{}{
		"quantity": float64(5), // 5 units
	})

	// Create a subscription with both fixed and usage items
	subscription := createTestSubscriptionWithItems(orgId, []entities.SubscriptionItem{
		createTestSubscriptionItem(orgId, subscriptionId, "item-1", 500, false), // Fixed price item
		usageItem, // Usage-based item
	})

	// Create billing period
	period := interfaces.BillingPeriod{
		StartDate: subscription.CurrentPeriodStart,
		EndDate:   subscription.CurrentPeriodEnd,
	}

	// Execute
	amount, err := billingService.CalculateHybridAmount(ctx, subscription, period)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), amount) // 500 (fixed) + 500 (usage)
}

// TestApplyDiscounts removed as the method has been removed from the BillingService interface
