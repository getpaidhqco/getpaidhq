# @getpaidhq/sdk

Official TypeScript SDK for the GetPaidHQ API (v1.0.5) - A comprehensive subscription billing platform supporting traditional subscriptions, usage-based billing, hybrid models, and public payments.

- Website: [getpaidhq.co](https://getpaidhq.co)
- Developer Guide: [getpaidhq.co/docs/developer-guide](https://getpaidhq.co/docs/developer-guide)
- API Documentation: [getpaidhq.co/docs/api-guide](https://getpaidhq.co/docs/api-guide)

## Installation

```bash
npm install @getpaidhq/sdk
# or
yarn add @getpaidhq/sdk
# or
pnpm add @getpaidhq/sdk
```

## What's New in v1.0.5

- **Public Payments**: New `PublicPaymentsResource` for handling public payment pages and orders without authentication
- **Token Authentication**: Support for token-based authentication via query parameters for public endpoints
- **Cart Management**: New `CartsResource` for shopping cart functionality
- **Enhanced Invoice Payments**: Added `initiatePayment` method for invoice payment initiation
- **Complete Resource Coverage**: All GetPaidHQ API resources now supported

## Quick Start

```typescript
import { GetPaidHQClient } from '@getpaidhq/sdk';

// Initialize with API key (recommended for backend applications)
const client = new GetPaidHQClient({
  apiKey: 'sk_your_api_key',
});

// Or initialize with Bearer token (for OAuth/JWT authentication)
const client = new GetPaidHQClient({
  bearerToken: 'your_bearer_token',
});

// Or initialize with token (for public payment endpoints)
const publicClient = new GetPaidHQClient({
  token: 'your_public_payment_token',
});

// Create a customer
const customer = await client.customers.create({
  email: 'john@example.com',
  first_name: 'John',
  last_name: 'Doe',
  phone: '+27123456789',
  country: 'ZA',
  company: 'Tech Corp',
});
```

## Authentication

The SDK supports three authentication methods:

### API Key Authentication

```typescript
const client = new GetPaidHQClient({
  apiKey: 'sk_your_api_key', // Must start with 'sk_'
});
```

### Bearer Token Authentication

```typescript
const client = new GetPaidHQClient({
  bearerToken: 'your_bearer_token', // OAuth/JWT token
});
```

### Token Authentication (Public Payments)

```typescript
// For public payment endpoints only
const client = new GetPaidHQClient({
  token: 'your_public_payment_token', // Used as query parameter
});
```

You can also update authentication credentials at runtime:

```typescript
client.updateApiKey('sk_new_api_key');
// or
client.updateBearerToken('new_bearer_token');
```

## Configuration

```typescript
const client = new GetPaidHQClient({
  apiKey: 'sk_your_api_key',
  baseURL: 'https://api.test.getpaidhq.co', // Optional: custom base URL
  timeout: 60000, // Optional: request timeout in ms (default: 30000)
  retries: 5, // Optional: number of retries (default: 3)
  retryDelay: 2000, // Optional: delay between retries in ms (default: 1000)
  userAgent: 'MyApp/1.0.0', // Optional: custom user agent
});
```

## Resources

### Customers

```typescript
// List customers
const customers = await client.customers.list({
  page: 0,
  limit: 20,
  status: 'active',
  email: 'john@example.com',
});

// Get a customer
const customer = await client.customers.get('customer-123');

// Update a customer
const updated = await client.customers.update('customer-123', {
  email: 'newemail@example.com',
  company: 'New Company Ltd',
});

// Delete a customer
await client.customers.delete('customer-123');

// Suspend/Activate a customer
await client.customers.suspend('customer-123');
await client.customers.activate('customer-123');

// Customer invoices
const invoices = await client.customers.listInvoices('customer-123', {
  status: 'paid',
});

// Payment methods
const paymentMethod = await client.customers.createPaymentMethod('customer-123', {
  name: 'Primary Card',
  type: 'card',
  psp: 'paystack',
  token: 'pm_xyz',
  is_default: true,
});
```

### Products and Pricing

```typescript
// Create a product
const product = await client.products.create({
  name: 'Premium Plan',
  description: 'Our premium subscription plan',
  type: 'service',
});

// Create a variant
const variant = await client.products.createVariant(product.id, {
  name: 'Monthly',
  description: 'Monthly billing',
});

// Create a price
const price = await client.prices.create({
  variant_id: variant.id,
  currency: 'USD',
  category: 'subscription',
  scheme: 'fixed',
  unit_price: 9999, // $99.99 in cents
  billing_interval: 'month',
  billing_interval_qty: 1,
});
```

### Subscriptions

```typescript
// Create a subscription
const subscription = await client.subscriptions.create({
  customer_id: customer.id,
  currency: 'USD',
  items: [
    {
      price_id: price.id,
      name: 'Premium Plan',
      quantity: 1,
    },
  ],
});

// Pause a subscription
await client.subscriptions.pause(subscription.id, {
  reason: 'Customer requested pause',
  pause_mode: 'temporary',
  resume_at: '2024-02-01T00:00:00Z',
});

// Resume a subscription
await client.subscriptions.resume(subscription.id, {
  reason: 'Customer requested resume',
  resume_behavior: 'start_new_period',
  proration_mode: 'credit_unused',
});

// Cancel a subscription
await client.subscriptions.cancel(subscription.id, {
  reason: 'Customer requested cancellation',
  cancel_at: 'period_end',
});

// Change subscription plan
const result = await client.subscriptions.changePlan(subscription.id, {
  new_variant_id: 'variant-123',
  new_price_id: 'new-price-id',
  proration_mode: 'immediate',
  effective_date: 'immediate',
  reason: 'Customer upgrade',
});
```

### Usage-Based Billing

```typescript
// Record usage (CloudEvents format - recommended)
const usage = await client.usage.recordEvent({
  subscription_item_id: 'si_123',
  quantity: 100,
  timestamp: new Date().toISOString(),
  action: 'increment',
  idempotency_key: 'unique-key-123',
});

// Batch record usage
const batchResult = await client.usage.batchRecord({
  records: [
    {
      subscription_item_id: 'si_123',
      quantity: 50,
      timestamp: '2024-01-01T00:00:00Z',
    },
    {
      subscription_item_id: 'si_124',
      quantity: 75,
      timestamp: '2024-01-01T00:00:00Z',
    },
  ],
});

// Get usage summary
const summary = await client.usage.getSummary('si_123', {
  start_date: '2024-01-01T00:00:00Z',
  end_date: '2024-01-31T23:59:59Z',
});

// Get subscription usage estimate
const estimate = await client.subscriptions.getUsageEstimate('sub_123');
```

### Meters

```typescript
// Create a meter for usage-based billing
const meter = await client.meters.create({
  name: 'API Calls',
  slug: 'api-calls',
  event_name: 'api_request',
  aggregation_type: 'sum',
  value_property: 'count',
  unit_type: 'requests',
});

// Get meter by slug
const meter = await client.meters.getBySlug('api-calls');
```

### Invoices

```typescript
// Create an invoice
const invoice = await client.invoices.create({
  customer_id: customer.id,
  due_date: '2024-02-01T00:00:00Z',
  line_items: [
    {
      description: 'Premium subscription',
      quantity: 1,
      unit_price: 9999, // $99.99 in cents
    },
  ],
});

// Send an invoice
await client.invoices.send(invoice.id);

// Mark as paid
await client.invoices.markPaid(invoice.id);

// Generate PDF
const pdfBlob = await client.invoices.generatePdf(invoice.id);

// Initiate payment for invoice
const paymentInitiation = await client.invoices.initiatePayment(invoice.id, {
  payment_processor: 'stripe',
  success_url: 'https://yourapp.com/success',
  cancel_url: 'https://yourapp.com/cancel',
});
```

### Payment Gateways

```typescript
// Create a payment gateway
const gateway = await client.gateways.create({
  psp_id: 'paystack',
  name: 'Paystack Gateway',
  settings: {
    api_key: 'sk_paystack_key',
    connect_id: 'connect_123',
  },
  is_active: true,
});

// List gateways
const gateways = await client.gateways.list();
```

### Organizations and Settings

```typescript
// Create an organization (onboarding)
const org = await client.organizations.create({
  name: 'Tech Corp',
  country: 'US',
  timezone: 'America/New_York',
});

// Get API keys
const apiKeys = await client.organizations.getApiKeys();

// Manage settings
const settings = await client.settings.list('org_123');
await client.settings.update('org_123', {
  value: 'new_value',
  description: 'Updated setting',
});
```

### Public Payments

Public payment endpoints allow customers to pay invoices without authentication:

```typescript
// Initialize client with token for public payments
const publicClient = new GetPaidHQClient({
  token: 'your_public_payment_token',
});

// Get public payment details (no auth required)
const paymentDetails = await publicClient.publicPayments.getPublicPaymentDetails('payment-slug');

// Create public payment order
const order = await publicClient.publicPayments.createPublicPaymentOrder('payment-slug', {
  payment_processor: 'stripe',
  customer_email: 'customer@example.com',
  customer_name: 'John Doe',
  success_url: 'https://yourapp.com/success',
  cancel_url: 'https://yourapp.com/cancel',
  metadata: {
    custom_field: 'value',
  },
});

// Check order status
const orderStatus = await publicClient.publicPayments.getPublicOrderStatus(
  'payment-slug',
  order.order_id
);
```

### Carts Management

```typescript
// Add item to cart
const cart = await client.carts.addItem('cart-123', {
  price_id: 'price_123',
  quantity: 2,
});

// Remove item from cart
await client.carts.removeItem('cart-123', {
  cart_item_id: 'item_123',
});

// Validate coupon code
const cartWithCoupon = await client.carts.validateCoupon('cart-123', {
  coupon_code: 'DISCOUNT10',
});
```

## Error Handling

The SDK provides typed errors for better error handling:

```typescript
import { 
  GetPaidHQError,
  ValidationError,
  AuthenticationError,
  NotFoundError,
  RateLimitError 
} from '@getpaidhq/sdk';

try {
  const customer = await client.customers.get('invalid_id');
} catch (error) {
  if (error instanceof NotFoundError) {
    console.error('Customer not found:', error.message);
  } else if (error instanceof ValidationError) {
    console.error('Validation failed:', error.details);
  } else if (error instanceof RateLimitError) {
    console.error('Rate limited, retry after:', error.retryAfter);
  } else if (error instanceof AuthenticationError) {
    console.error('Authentication failed');
  } else if (error instanceof GetPaidHQError) {
    console.error('API error:', error.statusCode, error.message);
  } else {
    console.error('Unexpected error:', error);
  }
}
```

## Pagination

All list endpoints support pagination:

```typescript
const result = await client.customers.list({
  page: 0, // 0-based page number
  limit: 50, // Items per page (max 100)
  sort_by: 'created_at',
  sort_order: 'desc',
});

console.log('Total customers:', result.meta.total);
console.log('Current page:', result.meta.page);
console.log('Customers:', result.data);
```

## TypeScript Support

The SDK is written in TypeScript and provides full type definitions:

```typescript
import type { 
  Customer, 
  Subscription, 
  Invoice,
  CreateCustomerRequest,
  SubscriptionStatus,
  // Public Payment types
  PublicPaymentDetailsResponse,
  PublicCreateOrderRequest,
  PublicOrderResponse,
  PublicOrderStatusResponse,
  // Cart types  
  CartResponse,
  AddCartItemRequest,
  RemoveCartItemRequest,
  ValidateCouponRequest
} from '@getpaidhq/sdk';

// All types are fully typed
const createCustomer = async (data: CreateCustomerRequest): Promise<Customer> => {
  return await client.customers.create(data);
};

// Type-safe status checks
const activeSubscriptions = subscriptions.filter(
  (sub): sub is Subscription & { status: 'active' } => sub.status === 'active'
);
```

# Tree-Shaking Examples

This SDK is fully optimized for tree-shaking, allowing consumers to import only what they need.

## Full Client Usage (includes everything)

```typescript
import { GetPaidHQClient } from '@getpaidhq/sdk';

const client = new GetPaidHQClient({ apiKey: 'sk_...' });
await client.customers.list();
```

## Individual Resource Usage (tree-shakeable)

```typescript
import { CustomersResource } from '@getpaidhq/sdk';
import { HttpClient } from '@getpaidhq/sdk/client';

// Only imports customers functionality
const httpClient = new HttpClient(config, authManager);
const customers = new CustomersResource(httpClient);
await customers.list();
```

## Specific Module Imports (highly tree-shakeable)

```typescript
// Import only specific resources
import { 
  CustomersResource, 
  PaymentsResource,
  PublicPaymentsResource,
  CartsResource 
} from '@getpaidhq/sdk/resources';

// Import only specific types
import type { 
  Customer, 
  Payment,
  PublicPaymentDetailsResponse,
  CartResponse 
} from '@getpaidhq/sdk/types';

// Import only specific errors
import { ValidationError, NotFoundError } from '@getpaidhq/sdk/errors';
```

## Type-Only Imports (zero runtime cost)

```typescript
import type {
  Customer,
  CreateCustomerRequest,
  Payment,
  Invoice,
  PublicPaymentDetailsResponse,
  PublicOrderResponse,
  CartResponse
} from '@getpaidhq/sdk';

// These imports have zero runtime cost
function processCustomer(customer: Customer): void {
  // Type-safe operations
}

function processPublicPayment(payment: PublicPaymentDetailsResponse): void {
  // Type-safe public payment operations
}
```

## Bundle Analysis

The SDK is built with:
- ✅ **ES Modules** - Modern import/export syntax
- ✅ **Code Splitting** - Separate chunks for different modules
- ✅ **Tree-shaking enabled** - Dead code elimination
- ✅ **Side-effect free** - `"sideEffects": false` in package.json
- ✅ **Multiple entry points** - Granular imports possible
- ✅ **Named exports** - Better for static analysis

## Bundle Sizes

| Import Style | Approximate Size | Use Case |
|--------------|------------------|----------|
| Full client | ~18KB | Complete SDK functionality |
| Individual resources | ~2-5KB per resource | Specific API operations |
| Types only | 0KB | Type definitions only |
| Specific functions | <1KB per function | Minimal footprint |

## Optimal Usage Patterns

### For Full-Featured Apps
```typescript
import { GetPaidHQClient } from '@getpaidhq/sdk';
```

### For Micro-services/Serverless
```typescript
import { CustomersResource } from '@getpaidhq/sdk/resources';
import type { Customer } from '@getpaidhq/sdk/types';
```

### For Libraries/Components
```typescript
import type { Customer, Payment } from '@getpaidhq/sdk/types';
// Only import types for interfaces, implement your own HTTP layer
```

## Advanced Usage

### Custom HTTP Client Access

For advanced use cases, you can access the underlying Axios instance:

```typescript
const axios = client.httpClient.getAxiosInstance();

// Add custom interceptors
axios.interceptors.request.use((config) => {
  console.log('Request:', config.url);
  return config;
});
```

### Health Check

```typescript
const health = await client.healthCheck();
console.log('API Status:', health.status); // "ok"
```

## API Reference

For detailed API documentation, visit [https://docs.getpaidhq.co/api](https://docs.getpaidhq.co/api)

## Support

- Documentation: [https://docs.getpaidhq.co](https://docs.getpaidhq.co)
- Email: support@getpaidhq.co
- Issues: [GitHub Issues](https://github.com/getpaidhqco/typescript-sdk/issues)

## License

MIT License - see LICENSE file for details
