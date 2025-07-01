# Overage Pricing Specification

## Overview

Overage pricing enables customers to pay for usage beyond their included allowance. This is essential for modern SaaS, telecommunications, and cloud service billing models.

## Core Concepts

### Hybrid Category
Overage pricing uses the `hybrid` price category, which combines:
- **Base Plan**: Fixed monthly/yearly fee with included usage
- **Overage Charges**: Additional fees when usage exceeds included amount

### Key Fields

```go
type Price struct {
    Category         "hybrid"        // Enables overage pricing
    UnitPrice        int64          // Base plan price (e.g., $29/month)
    IncludedUsage    int64          // Free allowance (e.g., 10,000 API calls)
    OverageUnitPrice int64          // Price per unit over limit (e.g., $0.01/call)
    UsageLimit       int64          // Optional hard limit
    
    // Usage configuration
    HasUsage         true
    UsageType        "metered"
    UnitType         "count"        // What's being measured
    AggregationType  "sum"          // How to calculate
}
```

## Price Categories by Use Case

### Traditional Subscription (No Overage)
```json
{
  "category": "subscription",
  "unit_price": 2900,
  "billing_interval": "month",
  "has_usage": false
}
```

### Pure Usage-Based (No Base Fee)
```json
{
  "category": "usage", 
  "unit_price": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

### Hybrid with Overage (Base + Usage)
```json
{
  "category": "hybrid",
  "unit_price": 2900,
  "included_usage": 10000,
  "overage_unit_price": 1,
  "has_usage": true,
  "usage_type": "metered", 
  "unit_type": "count",
  "aggregation_type": "sum"
}
```

## Implementation Examples

### Example 1: SaaS API Platform

**Starter Plan**: $19/month, 5k API calls included, $0.002 per additional call
```json
{
  "label": "Starter Plan",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 1900,
  "billing_interval": "month",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count", 
  "aggregation_type": "sum",
  "included_usage": 5000,
  "overage_unit_price": 2
}
```

**Pro Plan**: $49/month, 25k API calls included, $0.001 per additional call
```json
{
  "label": "Pro Plan",
  "category": "hybrid",
  "scheme": "fixed", 
  "unit_price": 4900,
  "billing_interval": "month",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum",
  "included_usage": 25000,
  "overage_unit_price": 1
}
```

### Example 2: Cloud Storage Service

**Personal Plan**: $5/month, 100GB included, $0.05 per additional GB
```json
{
  "label": "Personal Storage",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 500,
  "billing_interval": "month", 
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "gb",
  "aggregation_type": "max",
  "included_usage": 100,
  "overage_unit_price": 5
}
```

### Example 3: Telecommunications Plan

**Mobile Plan**: $30/month, 500 minutes included, $0.05 per additional minute
```json
{
  "label": "Mobile Voice Plan",
  "category": "hybrid", 
  "scheme": "fixed",
  "unit_price": 3000,
  "billing_interval": "month",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "minutes",
  "aggregation_type": "sum",
  "included_usage": 500,
  "overage_unit_price": 5
}
```

### Example 4: Freemium with Hard Limits

**Free Plan**: $0/month, 1k API calls included, no overage allowed
```json
{
  "label": "Free Plan",
  "category": "hybrid",
  "scheme": "fixed",
  "unit_price": 0,
  "billing_interval": "month",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "count",
  "aggregation_type": "sum", 
  "included_usage": 1000,
  "overage_unit_price": 0,
  "usage_limit": 1000
}
```

## Overage Calculation Algorithm

```javascript
function calculateHybridCharges(totalUsage, priceConfig) {
  const {
    unitPrice,           // Base plan fee
    includedUsage,       // Free allowance
    overageUnitPrice,    // Overage rate
    usageLimit          // Hard limit (optional)
  } = priceConfig;
  
  // Enforce usage limits
  if (usageLimit && totalUsage > usageLimit) {
    throw new UsageLimitExceeded(
      `Usage ${totalUsage} exceeds limit ${usageLimit}`
    );
  }
  
  // Calculate components
  const baseAmount = unitPrice;
  const includedQuantity = Math.min(totalUsage, includedUsage || 0);
  const overageQuantity = Math.max(0, totalUsage - (includedUsage || 0));
  const overageAmount = overageQuantity * overageUnitPrice;
  
  return {
    baseAmount,
    includedQuantity, 
    overageQuantity,
    overageAmount,
    totalAmount: baseAmount + overageAmount,
    
    // For invoice line items
    lineItems: [
      {
        description: `${priceConfig.label} - Base Plan`,
        quantity: 1,
        unitPrice: baseAmount,
        amount: baseAmount
      },
      ...(overageQuantity > 0 ? [{
        description: `${priceConfig.label} - Overage`,
        quantity: overageQuantity,
        unitPrice: overageUnitPrice,
        amount: overageAmount
      }] : [])
    ]
  };
}
```

## Usage Monitoring & Alerts

### Threshold Notifications

Implement proactive alerts at key usage percentages:

```javascript
const USAGE_THRESHOLDS = [50, 75, 90, 100];

function checkUsageAlerts(currentUsage, includedUsage, customerEmail) {
  const percentage = (currentUsage / includedUsage) * 100;
  
  USAGE_THRESHOLDS.forEach(threshold => {
    if (percentage >= threshold && !alertSent[threshold]) {
      sendUsageAlert({
        customer: customerEmail,
        threshold,
        currentUsage,
        includedUsage,
        projectedOverage: calculateProjectedOverage(currentUsage, includedUsage)
      });
      
      alertSent[threshold] = true;
    }
  });
}
```

### Sample Alert Messages

**75% Usage Alert:**
```
📊 Usage Alert: You've used 75% of your included API calls

Current usage: 7,500 / 10,000 calls this month
Remaining: 2,500 calls
Overage rate: $0.01 per additional call

View usage dashboard: https://app.example.com/usage
Consider upgrading: https://app.example.com/plans
```

**100% Usage Alert:**
```
⚠️ Overage Billing: You've exceeded your included usage

Usage this month: 12,500 / 10,000 calls 
Overage charges: 2,500 calls × $0.01 = $25.00
Your next invoice will include these overage charges.

View detailed usage: https://app.example.com/usage
Upgrade to avoid overage: https://app.example.com/plans
```

## Invoice Generation

### Line Item Structure

Overage charges should appear as separate line items for transparency:

```json
{
  "invoice": {
    "subscription_id": "sub_123",
    "period": "2025-01",
    "line_items": [
      {
        "subscription_item_id": "si_456",
        "description": "Pro Plan - Base",
        "quantity": 1,
        "unit_price": 4900,
        "amount": 4900,
        "metadata": {
          "type": "base_plan",
          "included_usage": 25000
        }
      },
      {
        "subscription_item_id": "si_456",
        "description": "API Calls - Overage", 
        "quantity": 5000,
        "unit_price": 1,
        "amount": 5000,
        "metadata": {
          "type": "overage",
          "total_usage": 30000,
          "included_usage": 25000,
          "overage_quantity": 5000
        }
      }
    ],
    "subtotal": 9900,
    "total": 9900
  }
}
```

## Best Practices

### 1. Clear Communication
- Always show included usage amounts in plan descriptions
- Provide usage dashboards for real-time monitoring  
- Send proactive alerts before overage charges occur

### 2. Reasonable Limits
- Set sensible usage limits to prevent bill shock
- Offer plan upgrade paths before hitting limits
- Consider graduated overage rates for high usage

### 3. Invoice Transparency
- Break down base vs overage charges clearly
- Show usage totals and per-unit rates
- Provide usage period details

### 4. Technical Implementation
- Implement real-time usage tracking
- Cache usage calculations for performance
- Queue overage alerts to prevent spam
- Validate usage limits before recording

## Testing Scenarios

### Test Cases

1. **Under Limit Usage**
   - Usage: 8,000 calls (under 10,000 limit)
   - Expected: Base price only ($29)

2. **Exact Limit Usage**  
   - Usage: 10,000 calls (exactly at limit)
   - Expected: Base price only ($29)

3. **Overage Usage**
   - Usage: 15,000 calls (5,000 over limit)
   - Expected: Base price + overage ($29 + $50 = $79)

4. **Hard Limit Enforcement**
   - Usage: 1,200 calls (200 over hard limit of 1,000)
   - Expected: Usage blocked, service throttled

5. **Zero Overage Rate**
   - Free plan with usage_limit but overage_unit_price = 0
   - Expected: Service stops at limit, no additional charges

6. **Multiple Usage Types**
   - API calls + storage in same subscription
   - Expected: Separate overage calculations per item

This specification ensures consistent implementation of overage pricing across all subscription types and use cases.