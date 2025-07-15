# AI Enhancement Context for GetPaidHQ SDK

## Project Overview
This is a TypeScript SDK for the GetPaidHQ billing and subscription management API. The SDK should provide a developer-friendly interface that abstracts complex billing operations into simple method calls.

## Key Business Domain Concepts

### Subscription Billing
- **Traditional Subscriptions**: Fixed monthly/yearly fees
- **Usage-Based Billing**: Pay-per-use (API calls, storage, transactions)
- **Hybrid Billing**: Base fee + usage overages
- **Transaction Processing**: Percentage fees + fixed fees per transaction

### Core Entities
1. **Subscriptions**: Recurring billing relationships
2. **Customers**: End users who pay for services
3. **Subscription Items**: Individual components of a subscription
4. **Usage Records**: Tracked usage for billing purposes
5. **Invoices**: Bills generated for customers
6. **Payments**: Payment transactions

### Pricing Models
- **Fixed**: Set amount per billing period
- **Tiered**: Different rates for quantity ranges
- **Volume**: All units at same rate based on total usage
- **Percentage**: Transaction-based fees (e.g., 2.9% + $0.30)

## AI Enhancement Guidelines

When enhancing the SDK, focus on:

1. **Developer Experience**:
   - Clear method names that match business operations
   - Helpful error messages with actionable guidance
   - Auto-completion friendly interfaces
   - Sensible defaults

2. **Common Use Cases**:
   - Creating subscriptions with trial periods
   - Recording usage for billing
   - Handling payment failures
   - Managing subscription lifecycle (pause/resume/cancel)
   - Generating invoices and processing payments

3. **Error Handling**:
   - Retry logic for transient failures
   - Clear error categorization (client errors vs server errors)
   - Helpful error messages for common mistakes

4. **Type Safety**:
   - Strong typing for all API responses
   - Discriminated unions for status fields
   - Optional fields clearly marked

## API Enhancement Patterns

### Resource Methods Should Follow RESTful Patterns:
```typescript
// CRUD operations
resource.create(data)
resource.find(id)
resource.list(filters)
resource.update(id, data)
resource.delete(id)

// Business operations
subscriptions.cancel(id, options)
subscriptions.pause(id, options)
subscriptions.changePlan(id, newPriceId)
usage.record(data)
usage.batchRecord(data)
```

### Builder Patterns for Complex Objects:
```typescript
// Instead of complex nested objects
const simpleSubscription = client.subscriptions.create({
  customerId: 'cus_123',
  items: [{ priceId: 'price_123', quantity: 1 }]
})

// Consider builder pattern for complex scenarios
const builderSubscription = client.subscriptions
  .builder()
  .customer('cus_123')
  .addItem('price_monthly', 1)
  .withTrial(14) // 14 days
  .withMetadata({ source: 'website' })
  .create()
```

### Async Iterators for Pagination:
```typescript
// Instead of manual pagination
for await (const subscription of client.subscriptions.listAll()) {
  console.log(subscription.id)
}
```

## Common Developer Pain Points to Address

1. Date Handling: Provide helpers for common date operations
2. Amount Formatting: Handle currency amounts (cents vs dollars)
3. Webhook Verification: Provide utilities for webhook signature verification
4. Retry Logic: Automatic retries with exponential backoff
5. Rate Limiting: Respect API rate limits automatically

## Testing Strategy

The SDK should include:
- Unit tests: For individual methods and utilities
- Integration tests: Against a test API environment
- Type tests: Ensure TypeScript definitions are correct
- Example tests: Verify all examples work correctly

## Documentation Requirements

1. Getting Started Guide: Quick setup and first API call
2. API Reference: Complete method documentation
3. Use Case Examples: Common billing scenarios
4. Migration Guide: For users upgrading from other SDKs
5. Troubleshooting: Common errors and solutions
