# Pricing Models Examples

This document provides examples of how to use different pricing models in Payloop.

## One-Time Pricing

One-time pricing is used for products that are purchased once and don't have recurring charges.

```json
{
  "category": "one_time",
  "scheme": "fixed",
  "currency": "USD",
  "unit_price": 2999,
  "label": "Standard License"
}
```

## Subscription Pricing

Subscription pricing is used for products with recurring charges at regular intervals.

```json
{
  "category": "subscription",
  "scheme": "fixed",
  "currency": "USD",
  "unit_price": 999,
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "label": "Monthly Plan"
}
```

## Usage-Based Pricing

Usage-based pricing charges customers based on their actual usage of a product or service.

```json
{
  "category": "usage",
  "scheme": "fixed",
  "currency": "USD",
  "unit_price": 5,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "api_calls",
  "aggregation_type": "sum",
  "label": "API Calls"
}
```

## Hybrid Pricing (Base + Usage)

Hybrid pricing combines a base subscription fee with additional usage-based charges.

```json
{
  "category": "hybrid",
  "scheme": "fixed",
  "currency": "USD",
  "unit_price": 1999,
  "billing_interval": "month",
  "billing_interval_qty": 1,
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "gb_hours",
  "aggregation_type": "sum",
  "included_usage": 100,
  "overage_unit_price": 10,
  "label": "Pro Plan with Compute"
}
```

## Transaction Fee Pricing

Transaction fee pricing is a special case of usage-based pricing that charges a percentage of transaction value plus a fixed fee.

```json
{
  "category": "usage",
  "scheme": "fixed",
  "currency": "USD",
  "has_usage": true,
  "usage_type": "metered",
  "unit_type": "transactions",
  "aggregation_type": "sum",
  "percentage_rate": 2.9,
  "fixed_fee": 30,
  "label": "Payment Processing Fee"
}
```

## Free Pricing

Free pricing is used for products that are offered at no cost.

```json
{
  "category": "free",
  "scheme": "fixed",
  "currency": "USD",
  "unit_price": 0,
  "label": "Free Plan"
}
```

## Variable Pricing

Variable pricing allows customers to choose how much they want to pay.

```json
{
  "category": "variable",
  "scheme": "fixed",
  "currency": "USD",
  "min_price": 500,
  "suggested_price": 1500,
  "label": "Pay What You Want"
}
```

## Usage Types and Aggregation Methods

### Usage Types

- `metered`: Usage is measured in real-time and billed based on actual consumption
- `licensed`: Usage is based on allocated capacity, regardless of actual consumption

### Aggregation Methods

- `sum`: Total usage over the billing period (e.g., total API calls)
- `max`: Maximum value observed during the billing period (e.g., peak concurrent users)
- `average`: Average value over the billing period (e.g., average storage used)
- `last_during_period`: Last value recorded during the billing period (e.g., active users at end of month)

### Unit Types

- `count`: Generic count of items
- `transactions`: Financial transactions
- `gb_hours`: Gigabyte-hours for compute resources
- `api_calls`: API requests
- `storage`: Storage capacity
- `bandwidth`: Data transfer
- `users`: User accounts
- `seats`: Licensed seats
- `custom`: Custom unit type