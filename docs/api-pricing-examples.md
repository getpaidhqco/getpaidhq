# API Pricing Examples

## Overview

This document provides comprehensive examples of creating different pricing models using GetPaidHQ's pricing API. Each example includes the API request, expected response, and usage scenarios.

> **Note**: As of the latest update, all pricing categories (`one_time`, `subscription`, `usage`, `hybrid`, `free`, `variable`) are now supported in the standalone price creation endpoint. Previously, the `usage` and `hybrid` categories were only available when creating prices through the product creation flow.

## Base API Endpoints

```
POST /api/v1/prices          # Create price
GET  /api/v1/prices/:id      # Get price by ID  
PUT  /api/v1/prices/:id      # Update price
DELETE /api/v1/prices/:id    # Delete price
GET  /api/v1/variants/:id/prices # Get prices for variant
```

## Simple Fixed Pricing

### Traditional SaaS Plan

**Use Case**: Monthly subscription with fixed fee

```json
POST /api/v1/prices
{
  "label": "Pro Plan",
  "variant_id": "variant_pro",
  "category": "subscription",
  "scheme": "fixed",
  "unit_price": 9900,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": false
}
```

**Response**:
```json
{
  "id": "price_123abc",
  "label": "Pro Plan", 
  "variant_id": "variant_pro",
  "category": "subscription",
  "scheme": "fixed",
  "unit_price": 9900,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": false,
  "tiers": [],
  "created_at": "2024-01-15T10:00:00Z"
}
```

**Billing**: Customer pays $99/month regardless of usage.

## Pure Usage-Based Pricing

### API Calls

**Use Case**: Pay per API call

```json
POST /api/v1/prices
{
  "label": "API Calls",
  "variant_id": "variant_api",
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 1,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

**Usage Scenario**: 
- Customer makes 15,000 API calls in January
- Billing: 15,000 × $0.001 = $15.00

### SMS Messages

**Use Case**: Pay per SMS sent

```json
POST /api/v1/prices
{
  "label": "SMS Messages",
  "variant_id": "variant_sms", 
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 2,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Customer sends 1,000 SMS messages
- Billing: 1,000 × $0.02 = $20.00

## Hybrid Pricing (Base + Usage)

### Overage Pricing

**Use Case**: Base plan with included usage, overage charges

```json
POST /api/v1/prices
{
  "label": "API Platform - Starter",
  "variant_id": "variant_api_starter",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 1900,
  "included_usage": 5000,
  "overage_unit_price": 2,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

**Usage Scenarios**:

*Scenario 1: Under limit*
- Base fee: $19/month
- Usage: 3,000 API calls (under 5,000 limit)
- Total: $19.00

*Scenario 2: Over limit*  
- Base fee: $19/month
- Usage: 8,000 API calls 
- Overage: (8,000 - 5,000) × $0.002 = $6.00
- Total: $25.00

### Freemium with Hard Limits

**Use Case**: Free plan with usage limits, no overage

```json
POST /api/v1/prices
{
  "label": "Free Plan",
  "variant_id": "variant_free",
  "category": "free", 
  "scheme": "fixed",
  "unit_price": 0,
  "usage_limit": 1000,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Monthly charge: $0
- Usage limit: 1,000 API calls
- Enforcement: Service blocks requests after limit

## Transaction-Based Pricing

### Payment Processing Fees

**Use Case**: Stripe-style transaction fees

```json
POST /api/v1/prices
{
  "label": "Payment Processing",
  "variant_id": "variant_payments",
  "category": "usage",
  "scheme": "fixed",
  "percentage_rate": 2.9,
  "fixed_fee": 30,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered", 
  "unit_type": "transactions",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Customer processes 50 transactions totaling $10,000
- Percentage fees: $10,000 × 2.9% = $290.00
- Fixed fees: 50 × $0.30 = $15.00
- Total: $305.00

### Marketplace Commission

**Use Case**: Marketplace with percentage fees

```json
POST /api/v1/prices
{
  "label": "Marketplace Commission",
  "variant_id": "variant_marketplace",
  "category": "usage",
  "scheme": "fixed", 
  "percentage_rate": 5.0,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "cents",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Vendor sells $50,000 worth of products
- Commission: $50,000 × 5% = $2,500

## Tiered Pricing

### Team Seats (Cumulative)

**Use Case**: Progressive pricing for team seats

```json
POST /api/v1/prices
{
  "label": "Team Seats - Tiered",
  "variant_id": "variant_team",
  "category": "usage",
  "scheme": "tiered",
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
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

**Usage Scenarios**:

*Scenario 1: Small team (3 seats)*
- Calculation: 3 × $10 = $30

*Scenario 2: Medium team (25 seats)*
- Calculation: (5 × $10) + (20 × $7) = $190

*Scenario 3: Large team (150 seats)*
- Calculation: (5 × $10) + (95 × $7) + (50 × $5) = $965

### API Calls with Volume Discounts

**Use Case**: Higher volume = lower per-call price

```json
POST /api/v1/prices
{
  "label": "API Calls - Tiered Volume",
  "variant_id": "variant_api_tiered",
  "category": "usage",
  "scheme": "tiered",
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count", 
  "aggregation_type": "sum",
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 10000,
      "unit_price": 100,
      "description": "First 10k calls at $0.001"
    },
    {
      "tier": 2,
      "from_qty": 10001,
      "to_qty": 100000,
      "unit_price": 80,
      "description": "10k-100k calls at $0.0008"
    },
    {
      "tier": 3,
      "from_qty": 100001,
      "to_qty": null,
      "unit_price": 50,
      "description": "100k+ calls at $0.0005"
    }
  ]
}
```

**Usage Scenario (75,000 calls)**:
- Tier 1: 10,000 × $0.001 = $10.00
- Tier 2: 65,000 × $0.0008 = $52.00  
- Total: $62.00

## Volume Pricing

### Team Seats (All-or-Nothing)

**Use Case**: All seats priced at tier rate

```json
POST /api/v1/prices
{
  "label": "Team Seats - Volume",
  "variant_id": "variant_team_volume",
  "category": "usage",
  "scheme": "volume",
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "seats",
  "aggregation_type": "max",
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 10,
      "unit_price": 1500,
      "description": "1-10 seats at $15 each"
    },
    {
      "tier": 2, 
      "from_qty": 11,
      "to_qty": 50,
      "unit_price": 1200,
      "description": "11-50 seats at $12 each"
    },
    {
      "tier": 3,
      "from_qty": 51,
      "to_qty": null,
      "unit_price": 1000,
      "description": "51+ seats at $10 each"
    }
  ]
}
```

**Usage Scenarios**:

*Scenario 1: 8 seats*
- Falls in tier 1 (1-10 seats)
- Calculation: 8 × $15 = $120

*Scenario 2: 25 seats* 
- Falls in tier 2 (11-50 seats)
- Calculation: 25 × $12 = $300

*Scenario 3: 75 seats*
- Falls in tier 3 (51+ seats)  
- Calculation: 75 × $10 = $750

## Seat-Based Pricing

### Max Seats During Period

**Use Case**: Bill for highest seat count during month

```json
POST /api/v1/prices
{
  "label": "Max Seats",
  "variant_id": "variant_max_seats",
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 1500,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1, 
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "seats",
  "aggregation_type": "max"
}
```

**Usage Scenario**:
- Day 1-15: 10 seats active
- Day 16-30: 25 seats active  
- Billing: 25 × $15 = $375 (max during period)

### Average Seats During Period

**Use Case**: Bill based on average seat usage

```json
POST /api/v1/prices
{
  "label": "Average Seats",
  "variant_id": "variant_avg_seats",
  "category": "usage", 
  "scheme": "fixed",
  "unit_price": 1500,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "seats",
  "aggregation_type": "average"
}
```

**Usage Scenario**:
- Day 1-15: 10 seats (15 days)
- Day 16-30: 25 seats (15 days)
- Average: ((10 × 15) + (25 × 15)) / 30 = 17.5 seats
- Billing: 17.5 × $15 = $262.50

## Storage & Bandwidth

### Storage with Average Billing

**Use Case**: Cloud storage based on average usage

```json
POST /api/v1/prices
{
  "label": "Cloud Storage",
  "variant_id": "variant_storage",
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 1000,
  "currency": "USD", 
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "gb_hours",
  "aggregation_type": "average"
}
```

**Usage Scenario**:
- Customer stores 100GB for 15 days, then 150GB for 15 days
- Average: 125GB for the month
- Billing: 125 × $0.10 = $12.50

### Bandwidth with Sum Billing

**Use Case**: Data transfer charges

```json
POST /api/v1/prices
{
  "label": "Data Transfer",
  "variant_id": "variant_bandwidth",
  "category": "usage",
  "scheme": "fixed", 
  "unit_price": 1000,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "gb",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Customer transfers 500GB during the month
- Billing: 500 × $0.10 = $50.00

## Complex Multi-Component Pricing

### Enterprise Platform

**Use Case**: Multiple usage types in one price

```json
POST /api/v1/prices
{
  "label": "Enterprise Platform",
  "variant_id": "variant_enterprise",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 99900,
  "currency": "USD",
  "billing_interval": "month", 
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum",
  "included_usage": 100000,
  "overage_unit_price": 1,
  "tiers": [
    {
      "tier": 1,
      "from_qty": 100001,
      "to_qty": 500000,
      "unit_price": 80,
      "description": "Overage calls 100k-500k at $0.0008"
    },
    {
      "tier": 2,
      "from_qty": 500001,
      "to_qty": null,
      "unit_price": 50,
      "description": "Overage calls 500k+ at $0.0005"
    }
  ]
}
```

**Usage Scenario (750k API calls)**:
- Base fee: $999/month
- Included: 100k calls (free)
- Overage tier 1: 400k calls × $0.0008 = $320
- Overage tier 2: 250k calls × $0.0005 = $125
- Total: $1,444

## Time-Based Pricing

### Compute Minutes

**Use Case**: Pay for compute time used

```json
POST /api/v1/prices
{
  "label": "Compute Minutes",
  "variant_id": "variant_compute",
  "category": "usage",
  "scheme": "fixed",
  "unit_price": 5,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "minutes",
  "aggregation_type": "sum"
}
```

**Usage Scenario**:
- Customer uses 1,500 compute minutes
- Billing: 1,500 × $0.05 = $75.00

### Phone Call Minutes

**Use Case**: Telecom-style billing

```json
POST /api/v1/prices
{
  "label": "Phone Minutes",
  "variant_id": "variant_phone",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 3000,
  "included_usage": 500,
  "overage_unit_price": 5,
  "currency": "USD",
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "minutes",
  "aggregation_type": "sum"
}
```

**Usage Scenario (750 minutes)**:
- Base fee: $30/month (includes 500 minutes)
- Overage: (750 - 500) × $0.05 = $12.50
- Total: $42.50

## Update Examples

### Adding Tiers to Existing Price

```json
PUT /api/v1/prices/price_123abc
{
  "label": "API Calls - Now with Tiers",
  "category": "usage",
  "scheme": "tiered",
  "unit_price": 0,
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered", 
  "unit_type": "count",
  "aggregation_type": "sum",
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 10000,
      "unit_price": 100,
      "description": "First 10k calls"
    },
    {
      "tier": 2,
      "from_qty": 10001,
      "to_qty": null,
      "unit_price": 50,
      "description": "10k+ calls"
    }
  ]
}
```

### Updating Tier Pricing

```json
PUT /api/v1/prices/price_456def
{
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 5,
      "unit_price": 1200,
      "description": "First 5 seats - Updated to $12"
    },
    {
      "tier": 2,
      "from_qty": 6,
      "to_qty": null,
      "unit_price": 800,
      "description": "6+ seats - Updated to $8"
    }
  ]
}
```

## Error Handling

### Invalid Tier Configuration

```json
POST /api/v1/prices
{
  "tiers": [
    {
      "tier": 1,
      "from_qty": 1,
      "to_qty": 10,
      "unit_price": 1000
    },
    {
      "tier": 2,
      "from_qty": 15,
      "to_qty": null,
      "unit_price": 800
    }
  ]
}
```

**Error Response**:
```json
{
  "error": "validation_error",
  "message": "Gap in tier ranges: tier 1 ends at 10, tier 2 starts at 15",
  "field": "tiers"
}
```

### Overlapping Tiers

```json
{
  "error": "validation_error", 
  "message": "Overlapping tier ranges: tier 1 (1-10) overlaps with tier 2 (5-20)",
  "field": "tiers"
}
```

## Best Practices

### API Design
1. **Consistent Pricing**: Use the same unit types across similar features
2. **Clear Descriptions**: Include human-readable tier descriptions
3. **Validation**: Validate tier ranges on creation/update
4. **Versioning**: Consider price versioning for plan changes

### Usage Recording
1. **Accurate Tracking**: Ensure usage records match billing units
2. **Timely Recording**: Record usage close to actual usage time
3. **Aggregation Alignment**: Match aggregation type to business logic
4. **Audit Trails**: Keep detailed usage logs for support

### Customer Experience  
1. **Transparency**: Show tier breakdowns on invoices
2. **Predictability**: Provide usage alerts and estimates
3. **Flexibility**: Allow plan changes with prorations
4. **Documentation**: Clearly explain pricing models to customers