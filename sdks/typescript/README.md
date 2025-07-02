# GetPaidHQ TypeScript SDK

The official TypeScript SDK for the GetPaidHQ API, providing a modern, resource-based interface for integrating with GetPaidHQ's billing and subscription management platform.

## Installation

```bash
npm install @getpaidhq/sdk
# or
yarn add @getpaidhq/sdk
# or
pnpm add @getpaidhq/sdk
```

## Quick Start

```typescript
import { GetPaidHQ } from '@getpaidhq/sdk'

// Initialize the client
const client = new GetPaidHQ({
  apiKey: 'your-api-key',
  // Optional configuration
  baseURL: 'https://api.getpaidhq.com',
  debug: false,
  timeout: 30000,
  retries: 3
})

// Example: Create a subscription
async function createSubscription() {
  try {
    const subscription = await client.subscriptions.create({
      customerId: 'cus_123',
      paymentMethodId: 'pm_456',
      items: [
        { priceId: 'price_monthly_plan', quantity: 1 }
      ]
    })

    console.log('Subscription created:', subscription.id)
    return subscription
  } catch (error) {
    console.error('Error creating subscription:', error.message)
    throw error
  }
}
```

## Features

- **AI-Enhanced SDK**: Goes beyond basic API generation with intelligent developer experience improvements
- **Modern TypeScript**: Built with TypeScript 5.4+ for excellent type safety and developer experience  
- **Two-Layer Architecture**: Raw API access + enhanced developer-friendly wrappers
- **Builder Patterns**: Fluent APIs for complex operations like subscription creation
- **Automatic Retries**: Built-in exponential backoff for transient errors
- **Async Iterators**: Seamless pagination without manual page handling
- **Utility Classes**: Currency formatting, webhook verification, usage batching
- **Comprehensive Error Handling**: Detailed error information with proper typing
- **Full API Coverage**: Support for all GetPaidHQ API endpoints

## Resources

The SDK provides access to the following resources:

- `client.subscriptions` - Manage subscriptions
- `client.customers` - Manage customers
- `client.invoices` - Manage invoices
- `client.payments` - Process payments
- `client.usage` - Record and retrieve usage data

## Usage Examples

### Managing Subscriptions

```typescript
// Create a subscription
const newSubscription = await client.subscriptions.create({
  customerId: 'cus_123',
  paymentMethodId: 'pm_456',
  items: [
    { priceId: 'price_monthly_plan', quantity: 1 }
  ]
})

// Retrieve a subscription
const existingSubscription = await client.subscriptions.find('sub_123')

// Update a subscription
await client.subscriptions.update('sub_123', {
  metadata: { notes: 'Updated subscription' }
})

// Cancel a subscription
await client.subscriptions.cancel('sub_123', {
  cancelMode: 'end_of_period'
})
```

### Usage-Based Billing

```typescript
// Record API usage
await client.usage.record({
  subscriptionItemId: 'si_123',
  quantity: 100, // 100 API calls
  timestamp: new Date().toISOString(),
  referenceId: 'api-call-batch-1'
})

// Get usage summary
const summary = await client.usage.getSummary('si_123', {
  startDate: '2024-01-01T00:00:00Z',
  endDate: '2024-01-31T23:59:59Z',
  granularity: 'day'
})
```

### Working with Invoices

```typescript
// List invoices for a customer
const invoices = await client.invoices.list({
  customerId: 'cus_123',
  status: 'open'
})

// Pay an invoice
await client.invoices.pay('inv_123', {
  paymentMethodId: 'pm_456'
})

// Get PDF download URL
const { url } = await client.invoices.getPdfUrl('inv_123')
```

## Error Handling

The SDK provides detailed error information through the `GetPaidHQError` class:

```typescript
import { GetPaidHQ, GetPaidHQError } from '@getpaidhq/sdk'

try {
  await client.subscriptions.find('non_existent_id')
} catch (error) {
  if (error instanceof GetPaidHQError) {
    console.error('API Error:', {
      message: error.message,
      status: error.status,
      code: error.code,
      details: error.details
    })
  } else {
    console.error('Unexpected error:', error)
  }
}
```

## Advanced Configuration

```typescript
const client = new GetPaidHQ({
  apiKey: 'your-api-key',
  baseURL: 'https://api.getpaidhq.com',
  timeout: 60000, // 60 seconds
  retries: 5, // Retry up to 5 times
  debug: process.env.NODE_ENV !== 'production' // Enable debug logging in non-production
})
```

## AI Enhancement System

This SDK uses an innovative AI-enhanced development process that creates developer-friendly abstractions on top of OpenAPI generation. See [AI-ENHANCEMENT.md](./AI-ENHANCEMENT.md) for details on:

- How the enhancement system works
- Available enhanced features (builders, utilities, iterators)
- Customizing and extending enhancements
- Development workflow

## Development

```bash
# Generate basic SDK from OpenAPI spec
npm run generate

# Add AI-powered enhancements
npm run enhance

# Full generation (both steps)
npm run generate:full

# Build the final SDK
npm run build
```

## Documentation

For complete API documentation, visit [docs.getpaidhq.com](https://docs.getpaidhq.com).

## License

MIT
