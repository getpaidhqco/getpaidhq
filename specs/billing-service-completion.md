# BillingService Implementation Completion Specification

## Current State

The BillingService has been partially implemented with the following components:

**✅ Completed:**
- Basic interface definition
- Service structure with dependencies
- Simple billing calculations for non-usage items
- Test structure and mock implementations
- Activity layer already refactored (lines 137-150 in `order_activities.go`)
- `ProcessSubscriptionCharge` in SubscriptionService (lines 716-851)

**❌ Missing/Incomplete:**
- Proper price category support (subscription, usage, hybrid)
- Usage aggregation logic
- Tiered pricing integration
- Proration calculations
- Module registration
- Comprehensive test coverage
- Support for all unit types and aggregation types

## Implementation Tasks

### Task 1: Remove DiscountService Dependency

**File**: `internal/application/services/billing_service.go`

**Changes**:
1. Remove `discountService` field from struct (line 16)
2. Remove `discountService` parameter from constructor (line 25)
3. Remove `ApplyDiscounts` method from interface and implementation
4. Update `CalculateBillingAmount` to remove discount logic (lines 78-86)

### Task 2: Add Price Category Support

**File**: `internal/application/services/billing_service.go`

**Update `calculateItemAmount` method** (starting at line 92):
```go
func (b *BillingService) calculateItemAmount(ctx context.Context, item entities.SubscriptionItem, period interfaces.BillingPeriod) (int64, interfaces.UsageCalculationResult, error) {
    // Determine price category from item metadata or default
    priceCategory := b.getPriceCategory(item)
    
    switch priceCategory {
    case "subscription":
        return item.Amount, interfaces.UsageCalculationResult{}, nil
        
    case "usage":
        return b.calculateUsageItemAmount(ctx, item, period)
        
    case "hybrid":
        baseAmount := item.Amount
        usageAmount, usageResult, err := b.calculateUsageItemAmount(ctx, item, period)
        if err != nil {
            return 0, interfaces.UsageCalculationResult{}, err
        }
        
        // For hybrid, check if usage exceeds included amount
        if item.IncludedUsage > 0 && usageResult.Quantity <= float64(item.IncludedUsage) {
            // Usage within included amount, no additional charge
            return baseAmount, usageResult, nil
        }
        
        // Calculate overage
        if item.IncludedUsage > 0 {
            usageResult.Quantity = usageResult.Quantity - float64(item.IncludedUsage)
            usageAmount = int64(usageResult.Quantity * float64(item.OverageUnitPrice))
            usageResult.Amount = usageAmount
        }
        
        return baseAmount + usageAmount, usageResult, nil
        
    default:
        return item.Amount, interfaces.UsageCalculationResult{}, nil
    }
}

func (b *BillingService) getPriceCategory(item entities.SubscriptionItem) string {
    // Check if item has price category in metadata
    if category, ok := item.Metadata["price_category"].(string); ok {
        return category
    }
    
    // Determine based on item properties
    if item.HasUsage && item.Amount > 0 {
        return "hybrid"
    } else if item.HasUsage {
        return "usage"
    }
    
    return "subscription"
}
```

### Task 3: Implement Proper Usage Aggregation

**File**: `internal/application/services/billing_service.go`

**Add new method** after `calculateItemAmount`:
```go
func (b *BillingService) calculateUsageItemAmount(ctx context.Context, item entities.SubscriptionItem, period interfaces.BillingPeriod) (int64, interfaces.UsageCalculationResult, error) {
    // Get usage records for the period
    usageRecords, err := b.usageRecordRepository.FindBySubscriptionItem(ctx, item.OrgId, item.Id, period.StartDate, period.EndDate)
    if err != nil {
        return 0, interfaces.UsageCalculationResult{}, err
    }
    
    // Aggregate usage based on aggregation type
    aggregatedQuantity := b.aggregateUsage(usageRecords, item.AggregationType)
    
    usageResult := interfaces.UsageCalculationResult{
        SubscriptionItemId: item.Id,
        UnitType:          string(item.UnitType),
        Quantity:          aggregatedQuantity,
        UnitPrice:         item.UnitPrice,
        AggregationType:   string(item.AggregationType),
    }
    
    // Calculate amount based on pricing scheme
    var calculatedAmount int64
    
    switch item.PricingScheme {
    case "fixed":
        calculatedAmount = int64(aggregatedQuantity * float64(item.UnitPrice))
        
    case "tiered", "volume", "graduated":
        // Get price for tier calculations
        price, err := b.priceRepository.FindById(ctx, item.OrgId, item.PriceId)
        if err != nil {
            return 0, usageResult, err
        }
        
        tierResult, err := b.tierCalculationService.CalculateTieredAmount(ctx, int(aggregatedQuantity), price)
        if err != nil {
            return 0, usageResult, err
        }
        
        calculatedAmount = tierResult.TotalAmount
        
    case "percentage":
        // For transaction-based fees
        totalTransactionValue := b.sumTransactionValues(usageRecords)
        calculatedAmount = int64(totalTransactionValue * item.PercentageRate / 100)
        
    default:
        calculatedAmount = int64(aggregatedQuantity * float64(item.UnitPrice))
    }
    
    usageResult.Amount = calculatedAmount
    return calculatedAmount, usageResult, nil
}

func (b *BillingService) aggregateUsage(records []entities.UsageRecord, aggregationType entities.AggregationType) float64 {
    if len(records) == 0 {
        return 0
    }
    
    switch aggregationType {
    case entities.AggregationTypeSum:
        var total float64
        for _, record := range records {
            total += record.Quantity
        }
        return total
        
    case entities.AggregationTypeMax:
        max := records[0].Quantity
        for _, record := range records[1:] {
            if record.Quantity > max {
                max = record.Quantity
            }
        }
        return max
        
    case entities.AggregationTypeAverage:
        var total float64
        for _, record := range records {
            total += record.Quantity
        }
        return total / float64(len(records))
        
    case entities.AggregationTypeLastDuringPeriod:
        latest := records[0]
        for _, record := range records[1:] {
            if record.CreatedAt.After(latest.CreatedAt) {
                latest = record
            }
        }
        return latest.Quantity
        
    default:
        // Default to sum
        var total float64
        for _, record := range records {
            total += record.Quantity
        }
        return total
    }
}

func (b *BillingService) sumTransactionValues(records []entities.UsageRecord) float64 {
    var total float64
    for _, record := range records {
        if record.TransactionValue > 0 {
            total += float64(record.TransactionValue)
        }
    }
    return total
}
```

### Task 4: Update Repository Interface

**File**: `internal/domain/repositories/usage_record_repository.go`

**Add method**:
```go
FindBySubscriptionItem(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) ([]entities.UsageRecord, error)
```

### Task 5: Implement Proration Calculations

**File**: `internal/application/services/billing_service.go`

**Update `CalculateProrationAdjustments` method**:
```go
func (b *BillingService) CalculateProrationAdjustments(ctx context.Context, subscription entities.Subscription) (int64, error) {
    // Check if subscription has pending proration metadata
    if subscription.Metadata == nil {
        return 0, nil
    }
    
    prorationAmount := int64(0)
    
    // Check for billing anchor change proration
    if amount, ok := subscription.Metadata["pending_proration_amount"].(float64); ok {
        prorationAmount = int64(amount)
        // Note: In production, you'd clear this metadata after processing
    }
    
    // Check for plan change proration
    if planChangeId, ok := subscription.Metadata["pending_plan_change_id"].(string); ok && planChangeId != "" {
        // In production, fetch the plan change details and calculate proration
        // For now, return any stored proration amount
        if amount, ok := subscription.Metadata["plan_change_proration"].(float64); ok {
            prorationAmount += int64(amount)
        }
    }
    
    return prorationAmount, nil
}
```

### Task 6: Update Main Calculation Method

**File**: `internal/application/services/billing_service.go`

**Update `CalculateBillingAmount` method**:
```go
func (b *BillingService) CalculateBillingAmount(ctx context.Context, subscription entities.Subscription) (interfaces.BillingCalculation, error) {
    var calculation interfaces.BillingCalculation
    calculation.Currency = subscription.Currency
    
    // Get current billing period
    period := b.getCurrentBillingPeriod(subscription)
    
    // Get subscription items from repository if not loaded
    var items []entities.SubscriptionItem
    if len(subscription.Items) > 0 {
        items = subscription.Items
    } else {
        var err error
        items, err = b.subscriptionItemRepository.FindBySubscriptionId(ctx, subscription.OrgId, subscription.Id)
        if err != nil {
            return interfaces.BillingCalculation{}, err
        }
    }
    
    // For legacy subscriptions without items, use the subscription amount
    if len(items) == 0 {
        calculation.BaseAmount = subscription.Amount
        calculation.TotalAmount = subscription.Amount
        calculation.ItemBreakdown = []interfaces.BillingItemBreakdown{{
            SubscriptionItemId: subscription.Id,
            Description:        "Legacy subscription",
            PriceCategory:      "subscription",
            Amount:            subscription.Amount,
        }}
        return calculation, nil
    }
    
    // Calculate amounts for each subscription item
    for _, item := range items {
        itemAmount, usageResult, err := b.calculateItemAmount(ctx, item, period)
        if err != nil {
            return interfaces.BillingCalculation{}, err
        }
        
        priceCategory := b.getPriceCategory(item)
        
        // Create breakdown entry
        breakdown := interfaces.BillingItemBreakdown{
            SubscriptionItemId: item.Id,
            Description:        item.Description,
            PriceCategory:      priceCategory,
            Amount:             itemAmount,
        }
        calculation.ItemBreakdown = append(calculation.ItemBreakdown, breakdown)
        
        // Add usage calculation details if applicable
        if usageResult.Quantity > 0 {
            calculation.UsageBreakdown = append(calculation.UsageBreakdown, usageResult)
        }
        
        // Add to appropriate totals based on price category
        switch priceCategory {
        case "subscription":
            calculation.BaseAmount += itemAmount
        case "usage":
            calculation.UsageAmount += itemAmount
        case "hybrid":
            calculation.BaseAmount += item.Amount
            overageAmount := itemAmount - item.Amount
            if overageAmount > 0 {
                calculation.UsageAmount += overageAmount
            }
        default:
            calculation.BaseAmount += itemAmount
        }
    }
    
    // Calculate proration adjustments
    prorationAmount, err := b.CalculateProrationAdjustments(ctx, subscription)
    if err != nil {
        return interfaces.BillingCalculation{}, err
    }
    calculation.ProrationAmount = prorationAmount
    
    // Calculate final total (no discounts)
    calculation.TotalAmount = calculation.BaseAmount + calculation.UsageAmount + calculation.ProrationAmount
    
    return calculation, nil
}
```

### Task 7: Update getCurrentBillingPeriod

**File**: `internal/application/services/billing_service.go`

**Update method**:
```go
func (b *BillingService) getCurrentBillingPeriod(subscription entities.Subscription) interfaces.BillingPeriod {
    // Use provided period dates if available
    if subscription.CurrentPeriodStart != nil && subscription.CurrentPeriodEnd != nil {
        return interfaces.BillingPeriod{
            StartDate: *subscription.CurrentPeriodStart,
            EndDate:   *subscription.CurrentPeriodEnd,
        }
    }
    
    // Calculate based on billing interval
    now := time.Now().UTC()
    startDate := subscription.CreatedAt
    
    // Find the current period based on billing interval
    switch subscription.BillingInterval {
    case "monthly":
        monthsSinceStart := int(now.Sub(startDate).Hours() / 24 / 30)
        startDate = startDate.AddDate(0, monthsSinceStart, 0)
        endDate := startDate.AddDate(0, 1, 0)
        return interfaces.BillingPeriod{
            StartDate: startDate,
            EndDate:   endDate,
        }
    case "yearly":
        yearsSinceStart := int(now.Sub(startDate).Hours() / 24 / 365)
        startDate = startDate.AddDate(yearsSinceStart, 0, 0)
        endDate := startDate.AddDate(1, 0, 0)
        return interfaces.BillingPeriod{
            StartDate: startDate,
            EndDate:   endDate,
        }
    case "weekly":
        weeksSinceStart := int(now.Sub(startDate).Hours() / 24 / 7)
        startDate = startDate.AddDate(0, 0, weeksSinceStart*7)
        endDate := startDate.AddDate(0, 0, 7)
        return interfaces.BillingPeriod{
            StartDate: startDate,
            EndDate:   endDate,
        }
    default:
        // Default to monthly
        endDate := startDate.AddDate(0, 1, 0)
        return interfaces.BillingPeriod{
            StartDate: startDate,
            EndDate:   endDate,
        }
    }
}
```

### Task 8: Module Registration

**File**: `internal/application/bootstrap/modules.go`

**Add to fx.Provide section**:
```go
// Billing service
fx.Provide(services.NewBillingService),
fx.Provide(func(bs *services.BillingService) interfaces.BillingService { return bs }),
```

### Task 9: Update Interface to Remove Discount Method

**File**: `internal/application/interfaces/billing_service.go`

**Remove line 20**:
```go
// Remove: ApplyDiscounts(ctx context.Context, subscription entities.Subscription, amount int64) (int64, error)
```

**Remove DiscountAmount from BillingCalculation struct** (line 27):
```go
type BillingCalculation struct {
    BaseAmount      int64                    `json:"base_amount"`
    UsageAmount     int64                    `json:"usage_amount"`
    ProrationAmount int64                    `json:"proration_amount"`
    // Remove: DiscountAmount  int64                    `json:"discount_amount"`
    TotalAmount     int64                    `json:"total_amount"`
    Currency        string                   `json:"currency"`
    ItemBreakdown   []BillingItemBreakdown   `json:"item_breakdown"`
    UsageBreakdown  []UsageCalculationResult `json:"usage_breakdown"`
}
```

### Task 10: Delete Unused DiscountService Interface

**Action**: Delete file `internal/application/interfaces/discount_service.go`

### Task 11: Update Tests

**File**: `internal/application/services/billing_service_test.go`

**Key updates needed**:
1. Remove all references to `MockDiscountService`
2. Update `NewBillingService` calls to remove discount service parameter
3. Add tests for:
   - Usage aggregation types (sum, max, average, last_during_period)
   - Hybrid pricing with overage calculations
   - Tiered pricing integration
   - Percentage-based pricing
   - Proration calculations
   - Price category determination logic

**Example test update**:
```go
func TestBillingService_HybridPricingWithOverage(t *testing.T) {
    // Setup
    ctx := context.Background()
    mockUsageRecordRepo := NewMockUsageRecordRepository()
    mockSubscriptionItemRepo := &MockSubscriptionItemRepository{}
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
    
    // Create hybrid item with included usage
    hybridItem := entities.SubscriptionItem{
        Id:              "hybrid-item",
        OrgId:           orgId,
        SubscriptionId:  subscriptionId,
        Amount:          1000, // $10 base
        UnitPrice:       100,  // $1 per overage unit
        IncludedUsage:   10,   // 10 units included
        HasUsage:        true,
        Metadata: map[string]interface{}{
            "price_category": "hybrid",
        },
    }
    
    // Mock 15 units of usage (5 overage)
    mockUsageRecordRepo.On("FindBySubscriptionItem", ctx, orgId, hybridItem.Id, mock.Anything, mock.Anything).
        Return([]entities.UsageRecord{
            {Quantity: 15},
        }, nil)
    
    subscription := createTestSubscriptionWithItems(orgId, []entities.SubscriptionItem{hybridItem})
    
    // Execute
    calculation, err := billingService.CalculateBillingAmount(ctx, subscription)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, int64(1000), calculation.BaseAmount)  // Base amount
    assert.Equal(t, int64(500), calculation.UsageAmount)  // 5 overage units * $1
    assert.Equal(t, int64(1500), calculation.TotalAmount) // Total
}
```

## Testing Checklist

- [ ] Test traditional fixed pricing
- [ ] Test pure usage-based pricing
- [ ] Test hybrid pricing with included usage
- [ ] Test hybrid pricing with overage
- [ ] Test all aggregation types (sum, max, average, last_during_period)
- [ ] Test tiered pricing integration
- [ ] Test percentage-based pricing
- [ ] Test proration calculations
- [ ] Test legacy subscription support (no items)
- [ ] Test error handling
- [ ] Test multi-tenancy validation

## Success Criteria

✅ **Removes unused DiscountService dependency**
✅ **Supports all pricing categories (subscription, usage, hybrid)**
✅ **Implements proper usage aggregation**
✅ **Integrates with TierCalculationService**
✅ **Handles proration calculations**
✅ **Maintains backward compatibility**
✅ **Comprehensive test coverage**
✅ **Module properly registered**

## Implementation Order

1. Remove DiscountService dependency (Tasks 1, 9, 10)
2. Add repository method for usage records (Task 4)
3. Implement price category support (Task 2)
4. Implement usage aggregation (Task 3)
5. Update main calculation method (Task 6)
6. Implement proration (Task 5)
7. Update billing period logic (Task 7)
8. Register module (Task 8)
9. Update and add tests (Task 11)

## Notes

- The implementation maintains backward compatibility with legacy subscriptions
- Price categories are determined from metadata or item properties
- Usage aggregation follows the specification in `specs/usage-types.md`
- Tiered pricing delegates to existing TierCalculationService
- Proration uses subscription metadata for pending adjustments
- All monetary values are in smallest currency unit (cents)