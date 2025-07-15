# Pricing Tiers Implementation Guide

## Overview

GetPaidHQ supports sophisticated pricing models through a dedicated `PriceTier` table that enables tiered, volume, and graduated pricing schemes. This guide covers how to implement and use pricing tiers effectively.

## Architecture

### Core Components

1. **Price Entity** - Main pricing configuration
2. **PriceTier Entity** - Individual tier definitions  
3. **PriceRepository** - Database operations with tier management
4. **Pricing Schemes** - Different calculation methods

### Database Schema

```sql
-- Main prices table (existing)
CREATE TABLE prices (
    org_id VARCHAR NOT NULL,
    id VARCHAR NOT NULL,
    -- ... other price fields
    scheme VARCHAR, -- 'fixed', 'tiered', 'volume', 'graduated'
    -- ... usage fields
    PRIMARY KEY (org_id, id)
);

-- New pricing tiers table
CREATE TABLE price_tiers (
    org_id VARCHAR NOT NULL,
    price_id VARCHAR NOT NULL,
    tier INT NOT NULL,
    from_qty INT NOT NULL,
    to_qty INT NULL, -- NULL = unlimited
    unit_price BIGINT NOT NULL,
    description VARCHAR NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (org_id, price_id, tier)
);
```

## Pricing Schemes

### 1. Tiered Pricing (Cumulative)
Each tier applies to the quantity range within that tier.

**Example**: Team seats with progressive pricing
- Tier 1: 1-5 seats at $10 each
- Tier 2: 6-100 seats at $7 each  
- Tier 3: 101+ seats at $5 each

**Usage**: 25 seats = (5 × $10) + (20 × $7) = $190

### 2. Volume Pricing (All-or-nothing)
All units are priced at the rate of the tier they fall into.

**Example**: Same tiers as above
**Usage**: 25 seats = 25 × $7 = $175 (all at tier 2 rate)

### 3. Graduated Pricing
Similar to tiered but with more complex rules (future enhancement).

## Implementation

### Creating a Tiered Price

```go
// 1. Create the main price
price := entities.Price{
    OrgId:           "org_123",
    Id:              "price_456", 
    Category:        prices.PriceCategoryUsage,
    Scheme:          prices.PriceSchemeFixed, // Fixed within each tier
    HasUsage:        true,
    UsageType:       prices.UsageTypeMetered,
    UnitType:        prices.UnitTypeSeats,
    AggregationType: prices.AggregationTypeMax,
    Currency:        "USD",
}

// 2. Define the tiers
tiers := []entities.PriceTier{
    {
        OrgId:       "org_123",
        PriceId:     "price_456",
        Tier:        1,
        FromQty:     1,
        ToQty:       &[]int{5}[0], // Pointer to 5
        UnitPrice:   1000, // $10.00
        Description: "First 5 seats",
    },
    {
        OrgId:       "org_123", 
        PriceId:     "price_456",
        Tier:        2,
        FromQty:     6,
        ToQty:       &[]int{100}[0], // Pointer to 100
        UnitPrice:   700, // $7.00
        Description: "Seats 6-100",
    },
    {
        OrgId:       "org_123",
        PriceId:     "price_456", 
        Tier:        3,
        FromQty:     101,
        ToQty:       nil, // Unlimited
        UnitPrice:   500, // $5.00
        Description: "Seats 101+",
    },
}

// 3. Assign tiers to price
price.Tiers = tiers

// 4. Save through repository
createdPrice, err := priceRepo.Create(ctx, price)
```

### API Request Format

```json
POST /api/v1/prices
{
  "label": "Team Seats - Tiered",
  "variant_id": "variant_123",
  "category": "usage",
  "scheme": "tiered", 
  "currency": "USD",
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
```

## Calculation Logic

### Tiered Calculation Implementation

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
            Tier:      tier.Tier,
            Usage:     tierUsage, 
            UnitPrice: tier.UnitPrice,
            Amount:    tierAmount,
        })
        
        remainingUsage -= tierUsage
    }
    
    return BillingResult{
        TotalAmount:   totalAmount,
        TierBreakdown: breakdown,
    }
}
```

### Volume Calculation Implementation

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
    
    if selectedTier == nil {
        return BillingResult{Error: "No tier found for usage amount"}
    }
    
    totalAmount := totalUsage * selectedTier.UnitPrice
    
    return BillingResult{
        TotalAmount: totalAmount,
        TierBreakdown: []TierBreakdown{{
            Tier:      selectedTier.Tier,
            Usage:     totalUsage,
            UnitPrice: selectedTier.UnitPrice,
            Amount:    totalAmount,
        }},
    }
}
```

## Repository Pattern

The `PriceRepository` automatically handles tier operations:

### Creation
- `Create()` saves price and calls `CreatePriceTiers()`
- Tiers are created in a transaction with the main price

### Retrieval  
- `FindById()` loads price and calls `GetPriceTiers()`
- `FindByVariantId()` loads tiers for each price in results

### Updates
- `Update()` calls `UpdatePriceTiers()` which:
  1. Deletes all existing tiers
  2. Creates new tiers from the entity

### Deletion
- `Delete()` calls `DeletePriceTiers()` before deleting the price

## Best Practices

### Tier Design
1. **Start Simple**: Begin with 2-3 tiers, expand as needed
2. **Clear Boundaries**: Ensure no gaps or overlaps in quantity ranges
3. **Logical Progression**: Higher quantities should have equal or lower unit prices
4. **Customer Communication**: Use clear descriptions that customers understand

### Implementation
1. **Validation**: Validate tier ranges don't overlap
2. **Ordering**: Always order tiers by `tier` field ascending
3. **Null Handling**: Last tier should have `to_qty = null` for unlimited
4. **Transaction Safety**: Use database transactions for tier operations

### Performance
1. **Eager Loading**: Load tiers with prices when doing calculations
2. **Caching**: Consider caching pricing calculations for high-volume usage
3. **Indexing**: Ensure proper database indexes on `(org_id, price_id, tier)`

## Common Patterns

### SaaS Seat Pricing
```go
// 1-10 seats: $15 each
// 11-50 seats: $12 each  
// 51+ seats: $10 each
tiers := []PriceTier{
    {Tier: 1, FromQty: 1, ToQty: &[]int{10}[0], UnitPrice: 1500},
    {Tier: 2, FromQty: 11, ToQty: &[]int{50}[0], UnitPrice: 1200},
    {Tier: 3, FromQty: 51, ToQty: nil, UnitPrice: 1000},
}
```

### API Call Pricing
```go
// 0-10k calls: $0.001 each
// 10k-100k calls: $0.0008 each
// 100k+ calls: $0.0005 each  
tiers := []PriceTier{
    {Tier: 1, FromQty: 1, ToQty: &[]int{10000}[0], UnitPrice: 100}, // $0.001 in cents
    {Tier: 2, FromQty: 10001, ToQty: &[]int{100000}[0], UnitPrice: 80}, // $0.0008
    {Tier: 3, FromQty: 100001, ToQty: nil, UnitPrice: 50}, // $0.0005
}
```

### Storage Pricing
```go
// 0-100GB: $0.10/GB
// 100-1000GB: $0.08/GB
// 1000GB+: $0.05/GB
tiers := []PriceTier{
    {Tier: 1, FromQty: 1, ToQty: &[]int{100}[0], UnitPrice: 1000}, // $0.10
    {Tier: 2, FromQty: 101, ToQty: &[]int{1000}[0], UnitPrice: 800}, // $0.08
    {Tier: 3, FromQty: 1001, ToQty: nil, UnitPrice: 500}, // $0.05
}
```

## Testing

### Unit Tests
```go
func TestTieredPricing(t *testing.T) {
    tiers := []entities.PriceTier{
        {Tier: 1, FromQty: 1, ToQty: &[]int{5}[0], UnitPrice: 1000},
        {Tier: 2, FromQty: 6, ToQty: &[]int{100}[0], UnitPrice: 700},
        {Tier: 3, FromQty: 101, ToQty: nil, UnitPrice: 500},
    }
    
    // Test tier 1 only
    result := CalculateTieredCharges(3, tiers)
    assert.Equal(t, int64(3000), result.TotalAmount) // 3 × $10
    
    // Test spanning two tiers  
    result = CalculateTieredCharges(8, tiers)
    assert.Equal(t, int64(7100), result.TotalAmount) // (5 × $10) + (3 × $7)
    
    // Test all three tiers
    result = CalculateTieredCharges(105, tiers)
    expected := (5 * 1000) + (95 * 700) + (5 * 500) // $118,000
    assert.Equal(t, int64(expected), result.TotalAmount)
}
```

### Integration Tests
```go
func TestPriceRepositoryWithTiers(t *testing.T) {
    price := entities.Price{
        OrgId:    "test_org",
        Id:       "test_price",
        Category: prices.PriceCategoryUsage,
        Scheme:   prices.PriceSchemeFixed,
        Tiers: []entities.PriceTier{
            {Tier: 1, FromQty: 1, ToQty: &[]int{10}[0], UnitPrice: 1000},
            {Tier: 2, FromQty: 11, ToQty: nil, UnitPrice: 800},
        },
    }
    
    // Create
    created, err := repo.Create(ctx, price)
    assert.NoError(t, err)
    assert.Len(t, created.Tiers, 2)
    
    // Retrieve
    found, err := repo.FindById(ctx, "test_org", "test_price")
    assert.NoError(t, err)
    assert.Len(t, found.Tiers, 2)
    assert.Equal(t, int64(1000), found.Tiers[0].UnitPrice)
    
    // Update
    found.Tiers[0].UnitPrice = 1200
    updated, err := repo.Update(ctx, found)
    assert.NoError(t, err)
    assert.Equal(t, int64(1200), updated.Tiers[0].UnitPrice)
    
    // Delete
    err = repo.Delete(ctx, "test_org", "test_price")
    assert.NoError(t, err)
}
```

## Migration Considerations

When migrating existing prices to use tiers:

1. **Single-tier Migration**: Convert fixed prices to single tier
2. **Preserve Behavior**: Ensure calculations remain the same
3. **Backward Compatibility**: Support prices without tiers
4. **Data Validation**: Verify tier ranges and pricing logic

See the [Pricing Migration Guide](./pricing-migration-guide.md) for detailed migration steps.