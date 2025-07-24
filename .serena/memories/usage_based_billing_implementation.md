# Usage-Based Billing Implementation in Payloop

## Pricing Model Categories
- **Traditional**: Fixed recurring subscription amounts
- **Usage-Based**: Pure usage billing with unit pricing, percentage fees, or transaction fees
- **Hybrid**: Fixed base amount + usage-based overage charges

## Usage Types and Implementation
- **API Calls**: Count-based billing with aggregation (sum, max, last_during_period)
- **Data Transfer**: Volume-based with unit pricing per GB/MB
- **Transaction Fees**: Percentage or fixed fee per transaction
- **Active Users**: Tiered pricing based on user count
- **Storage**: Volume-based with aggregation types

## Key Implementation Files
- **Entities**: `internal/domain/entities/price.go`, `internal/domain/entities/subscription_item.go`
- **Usage Records**: `internal/domain/entities/usage_record.go`
- **Repository**: `internal/infrastructure/db/postgres/usage_record_repository.go`
- **Types**: `internal/domain/entities/usage_types.go`

## Usage Types (from codebase)
```go
type UsageType string
const (
    UsageTypeMetered UsageType = "metered"
)

type AggregationType string  
const (
    AggregationTypeSum              = "sum"
    AggregationTypeMax              = "max" 
    AggregationTypeAverage          = "average"
    AggregationTypeLastDuringPeriod = "last_during_period"
)

type UnitType string
const (
    UnitTypeCount        = "count"
    UnitTypeGBHours      = "gb_hours"
    UnitTypeTransactions = "transactions"
    // ... more unit types
)
```

## Entity Construction Patterns
```go
// Always use factory methods for entities with validation
price, err := entities.NewPrice(orgId, variantId, input)
if err != nil {
    return err // Handle validation errors
}

// Use constructors for subscription items
subscriptionItem := entities.NewSubscriptionItem(orgId, subscriptionId, priceId, description, currency)
```

## Implementation Guidelines
- Specifications are in `specs/usage-types.md`
- Usage events processed through dedicated repositories
- Monthly aggregation for billing calculations
- Support for complex pricing schemes (tiered, volume, graduated)