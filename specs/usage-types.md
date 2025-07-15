# GetPaidHQ Usage Types Guide

## Overview

GetPaidHQ simplifies usage-based billing by treating all usage as "metered" with different aggregation methods. Instead of having separate types like "event-based", "licensed", or "time-based", everything uses the same underlying model. The differentiation happens through how you aggregate the data and what units you measure.

## Why a Single Usage Type?

Traditional billing systems often have multiple usage types:
- Metered (accumulated usage)
- Licensed (seat-based)
- Event-based (discrete occurrences)
- Time-based (duration tracking)

This creates complexity because each type needs different handling logic. GetPaidHQ's approach is simpler: everything is metered usage, but you configure:
1. **How to aggregate it** (sum, max, average)
2. **What units to measure** (count, gb_hours, transactions, etc.)

This unified model can represent any billing scenario while keeping the system design clean.

## Core Concepts

### Usage Type = "Metered"
In GetPaidHQ's simplified model, all usage is recorded as "metered" type. This doesn't mean everything is literally metered like a utility - it means all usage accumulates over time and is measured periodically. The actual billing behavior is determined by the aggregation type and unit.

### Aggregation Type
Defines how usage is calculated for billing:
- **`sum`**: Add all usage during the period (most common)
- **`max`**: Bill for the highest usage point during the period
- **`average`**: Bill based on average usage during the period
- **`last_during_period`**: Bill based on the final value in the period

### Unit
Defines what is being measured:
- **`count`**: Simple quantity (API calls, SMS, emails)
- **`gb_hours`**: Storage over time
- **`minutes`**: Time-based usage
- **`mb`**: Data transfer
- **`transactions`**: Payment processing (count + value)
- **`cents`/`dollars`**: Monetary amounts for percentage calculations
- **`seats`**: Active users or licenses

## Common Usage Patterns

### API Calls
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `count`

Customer makes 15,000 API calls in January, billed at $0.001 per call = $15.00

### Storage
- Usage Type: `metered`
- Aggregation: `average`
- Unit: `gb_hours`

Customer stores 100GB for 15 days, then 150GB for 15 days = average 125GB for the month

### Active Users/Seats
- Usage Type: `metered`
- Aggregation: `max`
- Unit: `seats`

Customer has 10 users on day 1, scales to 25 users mid-month = billed for 25 seats

### SMS Notifications
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `count`

Customer sends 1,000 SMS messages throughout the month at $0.02 each = $20.00

### Bandwidth/Data Transfer
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `gb`

Customer transfers 500GB of data at $0.10 per GB = $50.00

### Compute Time
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `minutes`

Customer uses 1,500 compute minutes at $0.05 per minute = $75.00

### Payment Processing (Transaction Fees)

For percentage-based transaction fees, you need to track both the transaction count and value:
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `transactions`

When recording usage, include the transaction value (e.g., $100.00 transaction = 10000 cents).

The pricing configuration would specify:
- Percentage rate: 2.9% (0.029)
- Fixed fee per transaction: $0.30

Example: Customer processes 50 transactions totaling $10,000
- Percentage fees: $10,000 × 2.9% = $290
- Fixed fees: 50 × $0.30 = $15
- Total fees: $305

### Alternative: Monetary Units

For pure percentage-based billing without per-transaction fees:
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `cents` or `dollars`

Example: Revenue sharing at 5% of sales
- Customer sells $50,000 worth of products
- Usage recorded: 5,000,000 cents
- With 5% pricing: $50,000 × 5% = $2,500

### Marketplace Commission

For marketplaces with sliding scale fees:
- Usage Type: `metered`
- Aggregation: `sum`
- Unit: `transactions`

Example: Marketplace with tiered rates
- 0-$1,000: 5% + $0.50 per transaction
- $1,000-$10,000: 4% + $0.40 per transaction
- $10,000+: 3% + $0.30 per transaction

## Special Considerations for Transaction Percentages

Transaction percentages require tracking both:
1. **Count**: Number of transactions (for fixed fees)
2. **Value**: Monetary amount of transactions (for percentage calculation)

This dual tracking allows for complex pricing models like:
- Stripe's 2.9% + $0.30 per transaction
- Tiered percentages (2.9% up to $10k, 2.5% after)
- Volume discounts based on total transaction value
- Different rates for different payment methods

The billing system calculates fees by:
1. Summing all transaction values for the period
2. Applying percentage rate(s)
3. Adding fixed fees based on transaction count
4. Generating invoice line items showing both components

## Why This Approach Works

By using a single "metered" type with different aggregation methods and units, GetPaidHQ achieves:

1. **Simplicity**: One consistent model handles all billing scenarios
2. **Flexibility**: New billing models don't require system changes
3. **Clarity**: The unit and aggregation make the billing logic transparent

Whether you're billing for API calls (sum of count), active users (max seats), storage (average gb_hours), or transaction fees (sum with percentage calculation), the same underlying model applies. This makes the system easier to understand, maintain, and extend.

## Pricing Configuration: UnitPrice vs Amount

The subscription item model separates **subscription-level pricing** from **usage-based pricing** for maximum flexibility:

### **Amount** - Fixed Subscription Pricing
```go
Amount int64 `json:"amount,omitempty"`  // Fixed amount per period (null for pure usage)
```

**Purpose**: Fixed recurring charge for the subscription item  
**When used**: Traditional subscription items with predictable costs  
**Billing**: Charged every billing cycle regardless of usage

**Examples**:
```go
// Fixed monthly seat fee
SubscriptionItem{
    Name: "Pro Seat",
    Quantity: 5,
    Amount: 2000,      // $20 per seat per month
    Currency: "USD",
    HasUsage: false,   // No usage tracking needed
}
// Total monthly charge: 5 × $20 = $100
```

### **UnitPrice** - Usage-Based Pricing
```go
UnitPrice int64 `json:"unit_price,omitempty"`  // Price per unit
```

**Purpose**: Price per unit of usage (API calls, GB, transactions, etc.)  
**When used**: Usage-based billing where customers pay for what they consume  
**Billing**: Calculated based on actual usage during the period

**Examples**:
```go
// API calls usage pricing
SubscriptionItem{
    Name: "API Calls",
    HasUsage: true,
    UsageType: UsageTypeMetered,
    UnitType: UnitTypeCount,
    AggregationType: AggregationTypeSum,
    UnitPrice: 10,     // $0.10 per 1000 API calls
    Currency: "USD",
}
// Usage: 5,000 API calls = 5 × $0.10 = $0.50
```

### **Hybrid Models** - Both Fixed + Usage

Many modern SaaS products combine both approaches:

```go
// Stripe-style payment processing
SubscriptionItem{
    Name: "Payment Processing",
    
    // Fixed base fee
    Amount: 0,              // No fixed monthly fee
    
    // Usage-based pricing
    HasUsage: true,
    UsageType: UsageTypeMetered,
    UnitType: UnitTypeTransactions,
    AggregationType: AggregationTypeSum,
    PercentageRate: 2.9,    // 2.9% of transaction value
    FixedFee: 30,          // $0.30 per transaction
    UnitPrice: 0,          // Not used for percentage-based
}
```

### **Pricing Configuration Fields**

The pricing fields work together to support complex billing models:

```go
// Pricing configuration
PercentageRate float64  // For percentage-based charges (2.9%)
FixedFee       int64    // Fixed fee per unit ($0.30 per transaction)  
UnitPrice      int64    // Price per unit ($0.001 per API call)
```

### **Common Pricing Patterns**

#### 1. **Traditional SaaS** (Fixed Amount)
```go
SubscriptionItem{
    Name: "Pro Plan",
    Quantity: 1,
    Amount: 9900,          // $99/month
    HasUsage: false,
}
```

#### 2. **Pure Usage** (UnitPrice only)
```go
SubscriptionItem{
    Name: "SMS Messages", 
    UnitPrice: 2,          // $0.02 per SMS
    HasUsage: true,
    UnitType: UnitTypeCount,
}
```

#### 3. **Base + Usage** (Amount + UnitPrice)
```go
SubscriptionItem{
    Name: "Pro Plan with API",
    Amount: 4900,          // $49 base fee
    UnitPrice: 1,          // $0.001 per API call over limit
    HasUsage: true,
}
```

#### 4. **Transaction Fees** (PercentageRate + FixedFee)
```go
SubscriptionItem{
    Name: "Payment Processing",
    PercentageRate: 2.9,   // 2.9% of transaction value
    FixedFee: 30,         // + $0.30 per transaction
    HasUsage: true,
    UnitType: UnitTypeTransactions,
}
```

### **Overage Pricing Models**

The `hybrid` category enables overage pricing - base plans with included usage and additional charges for overages:

#### **5. Hybrid Base + Overage** (Base fee + Usage overage)
```go
SubscriptionItem{
    Name: "Pro API Plan",
    Amount: 2900,          // $29 base fee  
    UnitPrice: 1,          // $0.01 per overage call
    HasUsage: true,
    UnitType: UnitTypeCount,
}
```

#### **6. Freemium with Hard Limits** (Free base + Optional overage)
```go
SubscriptionItem{
    Name: "Free Plan", 
    Amount: 0,             // Free base plan
    UnitPrice: 0,          // No overage allowed
    HasUsage: true,
    UnitType: UnitTypeCount,
    // Usage enforcement at service level
}
```

#### **7. Tiered Pricing** (Progressive pricing tiers)
```go
Price{
    Category: PriceCategoryUsage,
    Scheme: Fixed,              // Fixed rate within each tier
    HasUsage: true,
    UsageType: UsageTypeMetered,
    UnitType: UnitTypeSeats,
    AggregationType: AggregationTypeMax,
    Tiers: []PriceTier{
        {Tier: 1, FromQty: 1, ToQty: 5, UnitPrice: 1000},   // $10 per seat for 1-5
        {Tier: 2, FromQty: 6, ToQty: 100, UnitPrice: 700},  // $7 per seat for 6-100
        {Tier: 3, FromQty: 101, ToQty: nil, UnitPrice: 500}, // $5 per seat for 101+
    },
}
```

#### **8. Volume Pricing** (All units at same rate)
```go
Price{
    Category: PriceCategoryUsage,
    Scheme: Volume,
    HasUsage: true,
    UsageType: UsageTypeMetered,
    UnitType: UnitTypeSeats,
    AggregationType: AggregationTypeMax,
    Tiers: []PriceTier{
        {Tier: 1, FromQty: 1, ToQty: 5, UnitPrice: 1000},   // If ≤5 seats: all at $10
        {Tier: 2, FromQty: 6, ToQty: 100, UnitPrice: 700},  // If 6-100: all at $7
        {Tier: 3, FromQty: 101, ToQty: nil, UnitPrice: 500}, // If 101+: all at $5
    },
}
```

### **Pricing Calculation Logic**

#### **Overage Calculation (Hybrid Category)**
```go
func CalculateOverageCharges(totalUsage int64, price Price) BillingResult {
    // Base plan charge
    baseAmount := price.UnitPrice
    
    // Calculate overage
    includedUsage := price.IncludedUsage
    overageQuantity := max(0, totalUsage - includedUsage)
    overageAmount := overageQuantity * price.OverageUnitPrice
    
    return BillingResult{
        BaseAmount: baseAmount,
        OverageQuantity: overageQuantity, 
        OverageAmount: overageAmount,
        TotalAmount: baseAmount + overageAmount,
    }
}
```

#### **Tiered Calculation (Cumulative)**
```go
func CalculateTieredCharges(totalUsage int64, tiers []PriceTier) BillingResult {
    var totalAmount int64
    var breakdown []TierBreakdown
    remainingUsage := totalUsage
    
    for _, tier := range tiers {
        if remainingUsage <= 0 {
            break
        }
        
        // Calculate usage in this tier
        tierUsage := remainingUsage
        if tier.ToQty != nil && remainingUsage > (*tier.ToQty - tier.FromQty + 1) {
            tierUsage = *tier.ToQty - tier.FromQty + 1
        }
        
        tierAmount := tierUsage * tier.UnitPrice
        totalAmount += tierAmount
        
        breakdown = append(breakdown, TierBreakdown{
            Tier: tier.Tier,
            Usage: tierUsage,
            UnitPrice: tier.UnitPrice,
            Amount: tierAmount,
        })
        
        remainingUsage -= tierUsage
    }
    
    return BillingResult{
        TotalAmount: totalAmount,
        TierBreakdown: breakdown,
    }
}
```

#### **Volume Calculation (All at same rate)**
```go
func CalculateVolumeCharges(totalUsage int64, tiers []PriceTier) BillingResult {
    // Find appropriate tier based on total usage
    var selectedTier *PriceTier
    for _, tier := range tiers {
        if totalUsage >= tier.FromQty && (tier.ToQty == nil || totalUsage <= *tier.ToQty) {
            selectedTier = &tier
            break
        }
    }
    
    totalAmount := totalUsage * selectedTier.UnitPrice
    
    return BillingResult{
        TotalAmount: totalAmount,
        TierBreakdown: []TierBreakdown{{
            Tier: selectedTier.Tier,
            Usage: totalUsage,
            UnitPrice: selectedTier.UnitPrice,
            Amount: totalAmount,
        }},
    }
}
```

### **Pricing Examples by Scheme**

#### **Hybrid Overage Pricing**
```json
// API Platform: $19/month, 5k calls included, $0.002 overage
{
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 1900,
  "included_usage": 5000,
  "overage_unit_price": 2,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}

// Telecom: $30/month, 500 minutes included, $0.05 overage
{
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 3000,
  "included_usage": 500,
  "overage_unit_price": 5,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "minutes",
  "aggregation_type": "sum"
}
```

#### **Tiered Pricing (Cumulative)**
```json
// Team Seats: 1-5 seats at $10, 6-100 at $7, 101+ at $5
{
  "category": "usage",
  "scheme": "tiered",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "seats",
  "aggregation_type": "max",
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 5,
      "unit_price": 1000,
      "description": "First 5 seats"
    },
    {
      "tier": 2,
      "from_qty": 6,
      "to_qty": 100,
      "unit_price": 700,
      "description": "Seats 6-100"
    },
    {
      "tier": 3,
      "from_qty": 101,
      "to_qty": null,
      "unit_price": 500,
      "description": "Seats 101+"
    }
  ]
}

// Usage Example: 8 seats
// Calculation: (5 × $10) + (3 × $7) = $50 + $21 = $71
```

#### **Volume Pricing (All units at same rate)**
```json
// Team Seats with Volume Discounts
{
  "category": "usage",
  "scheme": "volume",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "seats",
  "aggregation_type": "max",
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 5,
      "unit_price": 1000,
      "description": "1-5 seats"
    },
    {
      "tier": 2,
      "from_qty": 6,
      "to_qty": 100,
      "unit_price": 700,
      "description": "6-100 seats"
    },
    {
      "tier": 3,
      "from_qty": 101,
      "to_qty": null,
      "unit_price": 500,
      "description": "101+ seats"
    }
  ]
}

// Usage Example: 8 seats
// Calculation: 8 × $7 = $56 (all seats at tier 2 rate)
```

#### **Pure Usage-Based (No base fee)**
```json
// API Calls: Simple per-call pricing
{
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}

// Usage Example: 15,000 calls
// Calculation: 15,000 × $0.001 = $15
```

### **Key Benefits**

1. **Flexibility**: Can model any pricing strategy from simple fixed to complex overage
2. **Evolution**: Can change from fixed to usage-based pricing without system changes
3. **Transparency**: Clear separation of base fees vs usage charges
4. **Accuracy**: Precise billing calculations for complex scenarios
5. **Customer Value**: Predictable base costs with pay-for-what-you-use flexibility

This design allows a single subscription item to handle everything from simple fixed fees to complex transaction-based pricing with multiple components, including modern freemium and overage models.

## Best Practices

- Choose aggregation type based on business logic (e.g., max for seat limits, sum for consumption)
- Use units that customers will understand on invoices
- For transaction fees, track both count and value to support flexible pricing models
- Consider future pricing changes when designing your usage model
- Keep audit trails by storing original values (not just calculated fees)
- Design for clarity - customers should easily understand what they're being charged for