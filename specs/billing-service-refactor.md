# Billing Service Refactor Specification

## Overview

This specification outlines the refactoring of subscription billing logic from Temporal activities to proper domain and application services, following DDD best practices. The goal is to move business logic from `ChargeCustomerForBillingPeriod` activity into appropriate service layers while supporting multiple pricing models.

## Objectives

1. **DDD Compliance**: Activities become thin coordinators that delegate to services
2. **Multi-Pricing Support**: Support traditional, usage-based, and hybrid pricing models
3. **Multi-Tenancy**: Ensure proper orgId validation throughout the billing flow
4. **Testability**: Isolate business logic in services for comprehensive testing
5. **Extensibility**: Enable easy addition of new pricing strategies

## Architecture Components

### 1. BillingService (NEW)

**Location**: `internal/application/services/billing_service.go`

**Purpose**: Centralize all billing amount calculations and pricing logic

**Interface**:
```go
type BillingService interface {
    // Main billing calculation method
    CalculateBillingAmount(ctx context.Context, orgId string, subscription entities.Subscription) (BillingCalculation, error)
    
    // Pricing model specific calculations
    CalculateTraditionalAmount(ctx context.Context, orgId string, subscription entities.Subscription) (int64, error)
    CalculateUsageAmount(ctx context.Context, orgId string, subscription entities.Subscription, period BillingPeriod) (int64, error)
    CalculateHybridAmount(ctx context.Context, orgId string, subscription entities.Subscription, period BillingPeriod) (int64, error)
    
    // Billing adjustments
    CalculateProrationAdjustments(ctx context.Context, orgId string, subscription entities.Subscription) (int64, error)
    ApplyDiscounts(ctx context.Context, orgId string, subscription entities.Subscription, amount int64) (int64, error)
}
```

**Types**:
```go
type BillingCalculation struct {
    BaseAmount        int64                    `json:"base_amount"`
    UsageAmount       int64                    `json:"usage_amount"`
    ProrationAmount   int64                    `json:"proration_amount"`
    DiscountAmount    int64                    `json:"discount_amount"`
    TotalAmount       int64                    `json:"total_amount"`
    Currency          string                   `json:"currency"`
    ItemBreakdown     []BillingItemBreakdown   `json:"item_breakdown"`
    UsageBreakdown    []UsageCalculationResult `json:"usage_breakdown"`
}

type BillingItemBreakdown struct {
    SubscriptionItemId string `json:"subscription_item_id"`
    Description        string `json:"description"`
    PriceCategory      string `json:"price_category"`
    Amount            int64  `json:"amount"`
}

type BillingPeriod struct {
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
}
```

**Dependencies**:
- `UsageRecordRepository` - For aggregating usage data
- `TierCalculationService` - For complex pricing calculations
- `SubscriptionItemRepository` - For subscription item details
- `DiscountService` - For discount applications

### 2. SubscriptionService Enhancement

**Location**: Extend existing `internal/application/services/subscription_service.go`

**New Method**:
```go
// ProcessSubscriptionCharge handles the complete subscription charging process
// including billing calculation, payment processing, and result handling
func (s *SubscriptionService) ProcessSubscriptionCharge(ctx context.Context, orgId string, subscription entities.Subscription) (payments.ChargeResult, error)
```

**Method Responsibilities**:
1. Validate orgId matches subscription.OrgId
2. Delegate billing calculation to BillingService
3. Coordinate payment gateway integration
4. Handle success/failure scenarios
5. Update subscription state and billing metadata

**Dependencies**:
- `BillingService` - For amount calculations
- `GatewayFactory` - For payment processing
- Existing repositories and services

### 3. Activity Layer Refactor

**Location**: `internal/infrastructure/workflow/temporal/activities/order_activities.go`

**Refactored Method**:
```go
// ChargeCustomerForBillingPeriod becomes a thin coordinator
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, currentSub entities.Subscription) (payments.ChargeResult, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("ChargeCustomerForBillingPeriod", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id)
    
    // Delegate to subscription service
    return a.subscriptionService.ProcessSubscriptionCharge(ctx, currentSub.OrgId, currentSub)
}
```

**Key Changes**:
- Remove all business logic from activity
- Remove direct repository access
- Remove gateway factory usage
- Keep only logging and delegation

## Implementation Requirements

### 1. BillingService Implementation

**File**: `internal/application/services/billing_service.go`

**Key Methods to Implement**:

```go
type billingService struct {
    usageRecordRepository      repositories.UsageRecordRepository
    subscriptionItemRepository repositories.SubscriptionItemRepository
    tierCalculationService     interfaces.TierCalculationService
    discountService           interfaces.DiscountService
}

func (b *billingService) CalculateBillingAmount(ctx context.Context, orgId string, subscription entities.Subscription) (BillingCalculation, error) {
    // Validate multi-tenancy
    if subscription.OrgId != orgId {
        return BillingCalculation{}, errors.New("orgId mismatch")
    }
    
    var calculation BillingCalculation
    calculation.Currency = subscription.Currency
    
    // Get current billing period
    period := b.getCurrentBillingPeriod(subscription)
    
    // Calculate amounts for each subscription item
    for _, item := range subscription.Items {
        itemAmount, breakdown, err := b.calculateItemAmount(ctx, orgId, item, period)
        if err != nil {
            return BillingCalculation{}, err
        }
        
        calculation.ItemBreakdown = append(calculation.ItemBreakdown, breakdown)
        
        // Add to appropriate totals based on price category
        switch item.PriceCategory {
        case "subscription":
            calculation.BaseAmount += itemAmount
        case "usage":
            calculation.UsageAmount += itemAmount
        case "hybrid":
            // Split between base and usage
            calculation.BaseAmount += item.Amount
            calculation.UsageAmount += (itemAmount - item.Amount)
        }
    }
    
    // Calculate proration adjustments
    prorationAmount, err := b.CalculateProrationAdjustments(ctx, orgId, subscription)
    if err != nil {
        return BillingCalculation{}, err
    }
    calculation.ProrationAmount = prorationAmount
    
    // Apply discounts
    totalBeforeDiscount := calculation.BaseAmount + calculation.UsageAmount + calculation.ProrationAmount
    discountAmount, err := b.ApplyDiscounts(ctx, orgId, subscription, totalBeforeDiscount)
    if err != nil {
        return BillingCalculation{}, err
    }
    calculation.DiscountAmount = discountAmount
    
    // Calculate final total
    calculation.TotalAmount = totalBeforeDiscount - discountAmount
    
    return calculation, nil
}

func (b *billingService) calculateItemAmount(ctx context.Context, orgId string, item entities.SubscriptionItem, period BillingPeriod) (int64, BillingItemBreakdown, error) {
    breakdown := BillingItemBreakdown{
        SubscriptionItemId: item.Id,
        Description:        item.Description,
        PriceCategory:      item.PriceCategory,
    }
    
    switch item.PriceCategory {
    case "subscription":
        breakdown.Amount = item.Amount
        return item.Amount, breakdown, nil
        
    case "usage":
        return b.calculateUsageItemAmount(ctx, orgId, item, period, &breakdown)
        
    case "hybrid":
        usageAmount, err := b.calculateUsageItemAmount(ctx, orgId, item, period, &breakdown)
        if err != nil {
            return 0, breakdown, err
        }
        totalAmount := item.Amount + usageAmount
        breakdown.Amount = totalAmount
        return totalAmount, breakdown, nil
        
    default:
        return 0, breakdown, fmt.Errorf("unsupported price category: %s", item.PriceCategory)
    }
}

func (b *billingService) calculateUsageItemAmount(ctx context.Context, orgId string, item entities.SubscriptionItem, period BillingPeriod, breakdown *BillingItemBreakdown) (int64, error) {
    // Get usage records for this item and billing period
    usageRecords, err := b.usageRecordRepository.FindBySubscriptionItem(ctx, orgId, item.Id, period.StartDate, period.EndDate)
    if err != nil {
        return 0, err
    }
    
    // Aggregate usage based on aggregation type
    aggregatedUsage := b.aggregateUsage(usageRecords, item.AggregationType)
    
    // Calculate amount based on pricing scheme
    switch item.PricingScheme {
    case "fixed":
        return int64(aggregatedUsage * float64(item.UnitPrice)), nil
        
    case "tiered", "volume", "graduated":
        return b.tierCalculationService.CalculateTieredAmount(ctx, aggregatedUsage, item.Tiers, item.PricingScheme)
        
    default:
        return 0, fmt.Errorf("unsupported pricing scheme: %s", item.PricingScheme)
    }
}
```

### 2. SubscriptionService Enhancement

**Add to existing**: `internal/application/services/subscription_service.go`

```go
func (s *SubscriptionService) ProcessSubscriptionCharge(ctx context.Context, orgId string, subscription entities.Subscription) (payments.ChargeResult, error) {
    // Validate multi-tenancy
    if subscription.OrgId != orgId {
        return payments.ChargeResult{}, fmt.Errorf("orgId mismatch: expected %s, got %s", subscription.OrgId, orgId)
    }
    
    // Get latest subscription data
    currentSubscription, err := s.subscriptionRepository.FindById(ctx, orgId, subscription.Id)
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to get subscription: %w", err)
    }
    
    // Calculate billing amount
    billingCalculation, err := s.billingService.CalculateBillingAmount(ctx, orgId, currentSubscription)
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to calculate billing amount: %w", err)
    }
    
    // Skip charging if amount is zero
    if billingCalculation.TotalAmount <= 0 {
        return payments.ChargeResult{
            Status:      payments.PaymentStatusSucceeded,
            Amount:      0,
            Currency:    billingCalculation.Currency,
            ProcessedAt: time.Now(),
        }, nil
    }
    
    // Get payment gateway
    gateway, err := s.gatewayFactory.NewGateway(ctx, orgId, string(currentSubscription.PspId))
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to get payment gateway: %w", err)
    }
    
    // Get customer and payment method
    customer, err := s.GetSubscriptionCustomer(ctx, currentSubscription)
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to get customer: %w", err)
    }
    
    paymentMethod, err := s.GetSubscriptionPaymentMethod(ctx, currentSubscription)
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to get payment method: %w", err)
    }
    
    decryptedToken, err := paymentMethod.GetToken(ctx)
    if err != nil {
        return payments.ChargeResult{}, fmt.Errorf("failed to decrypt payment token: %w", err)
    }
    
    // Process payment
    chargeResult := gateway.ChargePayment(ctx, payment_providers.ChargePaymentCommand{
        OrgId:          orgId,
        OrderId:        currentSubscription.OrderId,
        SubscriptionId: currentSubscription.Id,
        Amount:         billingCalculation.TotalAmount,
        Currency:       billingCalculation.Currency,
        PaymentMethod: payment_providers.PaymentMethod{
            PspId:       paymentMethod.Id,
            Name:        paymentMethod.Name,
            Type:        string(paymentMethod.Type),
            IsRecurring: true,
            Token:       decryptedToken,
        },
        Customer: customer,
        Metadata: map[string]interface{}{
            "billing_calculation": billingCalculation,
        },
    })
    
    // Handle gateway errors
    if chargeResult.Status == payment_providers.GatewayError {
        s.errorReporter.ReportError(ctx, errors.New("gateway error while charging subscription"), map[string]interface{}{
            "org_id":          orgId,
            "subscription_id": currentSubscription.Id,
            "error":           chargeResult.ErrorReason,
            "psp":             string(currentSubscription.PspId),
            "amount":          billingCalculation.TotalAmount,
        })
        return payments.ChargeResult{}, temporal.NewApplicationError(chargeResult.ErrorReason, "gateway_error", nil)
    }
    
    // Convert to domain charge result
    domainChargeResult := s.convertToChargeResult(chargeResult, billingCalculation.Currency)
    
    // Handle charge result (success or failure)
    if domainChargeResult.Status == payments.PaymentStatusSucceeded {
        _, err = s.HandleSubscriptionChargeSuccess(ctx, subscriptions.SubscriptionChargeInput{
            Subscription: currentSubscription,
            ChargeResult: domainChargeResult,
        })
    } else {
        _, err = s.HandleSubscriptionChargeFailure(ctx, subscriptions.SubscriptionChargeInput{
            Subscription: currentSubscription,
            ChargeResult: domainChargeResult,
        })
    }
    
    if err != nil {
        return domainChargeResult, fmt.Errorf("failed to handle charge result: %w", err)
    }
    
    return domainChargeResult, nil
}

func (s *SubscriptionService) convertToChargeResult(gatewayResult payment_providers.ChargeResult, currency string) payments.ChargeResult {
    var status payments.PaymentStatus
    var completedAt time.Time
    
    switch gatewayResult.Status {
    case payment_providers.ChargePaymentStatusSuccess:
        status = payments.PaymentStatusSucceeded
        completedAt = time.Now()
    case payment_providers.ChargePaymentStatusPending:
        status = payments.PaymentStatusPending
    case payment_providers.ChargePaymentStatusError:
        status = payments.PaymentStatusFailed
    }
    
    rawData, _ := json.Marshal(gatewayResult.PspResponse)
    
    return payments.ChargeResult{
        Psp:         gatewayResult.Psp,
        Amount:      gatewayResult.AmountCharged,
        Status:      status,
        Currency:    currency,
        ErrorReason: gatewayResult.ErrorReason,
        ErrorCode:   gatewayResult.ErrorCode,
        PspId:       gatewayResult.PspId,
        Reference:   gatewayResult.Reference,
        ProcessedAt: completedAt,
        RawData:     string(rawData),
    }
}
```

### 3. Activity Layer Changes

**Modify**: `internal/infrastructure/workflow/temporal/activities/order_activities.go`

**Constructor Update**:
```go
type OrderActivities struct {
    // Remove direct dependencies that are now handled by services
    // orderService           interfaces.OrderWorkflowService
    subscriptionService    interfaces.SubscriptionService
    dunningService         interfaces.DunningService
    // Remove: subscriptionRepository, settingRepository, paymentRepository, gatewayFactory
    pubsub                 events.PubSub
    errorReporter          lib.ErrorReporter
}
```

**Method Update**:
```go
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, currentSub entities.Subscription) (payments.ChargeResult, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("ChargeCustomerForBillingPeriod", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "amount", currentSub.Amount)
    
    // Thin delegation to subscription service
    chargeResult, err := a.subscriptionService.ProcessSubscriptionCharge(ctx, currentSub.OrgId, currentSub)
    if err != nil {
        logger.Error("Failed to process subscription charge", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "error", err.Error())
        return payments.ChargeResult{}, err
    }
    
    logger.Info("Subscription charge completed", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "status", chargeResult.Status, "amount", chargeResult.Amount)
    return chargeResult, nil
}
```

## Interface Updates

### 1. Add BillingService Interface

**Location**: `internal/application/interfaces/billing_service.go`

```go
package interfaces

import (
    "context"
    "payloop/internal/domain/entities"
)

type BillingService interface {
    CalculateBillingAmount(ctx context.Context, orgId string, subscription entities.Subscription) (BillingCalculation, error)
    CalculateTraditionalAmount(ctx context.Context, orgId string, subscription entities.Subscription) (int64, error)
    CalculateUsageAmount(ctx context.Context, orgId string, subscription entities.Subscription, period BillingPeriod) (int64, error)
    CalculateHybridAmount(ctx context.Context, orgId string, subscription entities.Subscription, period BillingPeriod) (int64, error)
    CalculateProrationAdjustments(ctx context.Context, orgId string, subscription entities.Subscription) (int64, error)
    ApplyDiscounts(ctx context.Context, orgId string, subscription entities.Subscription, amount int64) (int64, error)
}

type BillingCalculation struct {
    BaseAmount        int64                    `json:"base_amount"`
    UsageAmount       int64                    `json:"usage_amount"`
    ProrationAmount   int64                    `json:"proration_amount"`
    DiscountAmount    int64                    `json:"discount_amount"`
    TotalAmount       int64                    `json:"total_amount"`
    Currency          string                   `json:"currency"`
    ItemBreakdown     []BillingItemBreakdown   `json:"item_breakdown"`
    UsageBreakdown    []UsageCalculationResult `json:"usage_breakdown"`
}

type BillingItemBreakdown struct {
    SubscriptionItemId string `json:"subscription_item_id"`
    Description        string `json:"description"`
    PriceCategory      string `json:"price_category"`
    Amount            int64  `json:"amount"`
}

type BillingPeriod struct {
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
}

type UsageCalculationResult struct {
    SubscriptionItemId string  `json:"subscription_item_id"`
    UnitType          string  `json:"unit_type"`
    Quantity          float64 `json:"quantity"`
    UnitPrice         int64   `json:"unit_price"`
    Amount            int64   `json:"amount"`
    AggregationType   string  `json:"aggregation_type"`
}
```

### 2. Update SubscriptionService Interface

**Location**: `internal/application/interfaces/subscription_service.go`

**Add Method**:
```go
ProcessSubscriptionCharge(ctx context.Context, orgId string, subscription entities.Subscription) (payments.ChargeResult, error)
```

## Module Registration

**Location**: `internal/application/bootstrap/modules.go`

**Add BillingService**:
```go
// Add to the existing fx.Options slice
fx.Provide(services.NewBillingService),
fx.Provide(func(bs *services.BillingService) interfaces.BillingService { return bs }),
```

## Testing Requirements

### 1. BillingService Tests

**Location**: `internal/application/services/billing_service_test.go`

**Test Cases**:
- Traditional subscription billing (fixed amounts)
- Usage-based billing with different aggregation types
- Hybrid billing (base + usage)
- Tiered pricing calculations
- Proration adjustments
- Multi-tenant validation (orgId mismatch)
- Zero amount billing
- Error handling for invalid pricing schemes

### 2. SubscriptionService Tests

**Location**: `internal/application/services/subscription_service_test.go`

**New Test Cases for ProcessSubscriptionCharge**:
- Successful charge processing
- Gateway error handling
- Multi-tenancy validation
- Zero amount handling
- Payment method failures
- Charge result handling

### 3. Integration Tests

**Location**: `internal/infrastructure/workflow/temporal/activities/order_activities_test.go`

**Test Cases**:
- Activity delegation to service
- Error propagation
- Logging verification

## Implementation Steps

1. **Create BillingService interface and implementation**
2. **Add ProcessSubscriptionCharge method to SubscriptionService**
3. **Refactor OrderActivities to use thin delegation**
4. **Update module registration**
5. **Write comprehensive tests**
6. **Update existing workflows to handle enhanced billing data**

## Migration Strategy

1. **Phase 1**: Implement new services alongside existing activity logic
2. **Phase 2**: Update activity to delegate to new service while maintaining backwards compatibility
3. **Phase 3**: Remove old business logic from activity after testing
4. **Phase 4**: Clean up unused dependencies and methods

## Success Criteria

- ✅ Activities contain no business logic (only delegation and logging)
- ✅ All pricing models (traditional, usage, hybrid) are supported
- ✅ Multi-tenancy is enforced throughout the billing flow
- ✅ Billing calculations are testable and isolated
- ✅ Zero breaking changes to existing workflows
- ✅ Comprehensive test coverage for all pricing scenarios
- ✅ Error handling and logging maintain current quality standards