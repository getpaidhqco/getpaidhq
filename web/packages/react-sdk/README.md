# @getpaidhq/react-sdk

A React SDK for the GetPaidHQ API, built on top of TanStack Query and @getpaidhq/sdk.

## Installation

```bash
# Using npm
npm install @getpaidhq/react-sdk @tanstack/react-query

# Using yarn
yarn add @getpaidhq/react-sdk @tanstack/react-query

# Using pnpm
pnpm add @getpaidhq/react-sdk @tanstack/react-query
```

## Setup

Wrap your application with the `GetPaidHQProvider` component:

```tsx
import { GetPaidHQProvider } from '@getpaidhq/react-sdk';

function App() {
  return (
    <GetPaidHQProvider apiKey="your-api-key">
      <YourApp />
    </GetPaidHQProvider>
  );
}
```

## Usage

### Customers

```tsx
import { useCustomers, useCustomer, useCreateCustomer, useUpdateCustomer, useDeleteCustomer } from '@getpaidhq/react-sdk';

// Fetch a list of customers
function CustomersList() {
  const { data, isLoading, error } = useCustomers();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.customers.map(customer => (
        <li key={customer.id}>{customer.name}</li>
      ))}
    </ul>
  );
}

// Fetch a single customer
function CustomerDetails({ id }) {
  const { data, isLoading, error } = useCustomer(id);

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h1>{data.name}</h1>
      <p>{data.email}</p>
    </div>
  );
}

// Create a new customer
function CreateCustomerForm() {
  const { mutate, isLoading, error } = useCreateCustomer();

  const handleSubmit = (e) => {
    e.preventDefault();
    mutate({
      name: 'John Doe',
      email: 'john@example.com',
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* Form fields */}
      <button type="submit" disabled={isLoading}>
        {isLoading ? 'Creating...' : 'Create Customer'}
      </button>
      {error && <div>Error: {error.message}</div>}
    </form>
  );
}
```

### Organizations

```tsx
import { useOrganizations, useOrganization, useCreateOrganization, useUpdateOrganization } from '@getpaidhq/react-sdk';

// Fetch a list of organizations
function OrganizationsList() {
  const { data, isLoading, error } = useOrganizations();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.organizations.map(org => (
        <li key={org.id}>{org.name}</li>
      ))}
    </ul>
  );
}
```

### Subscriptions

```tsx
import { 
  useSubscriptions, 
  useSubscription, 
  useCreateSubscription, 
  useUpdateSubscription,
  useCancelSubscription,
  usePauseSubscription,
  useResumeSubscription
} from '@getpaidhq/react-sdk';

// Fetch a list of subscriptions
function SubscriptionsList() {
  const { data, isLoading, error } = useSubscriptions();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.subscriptions.map(subscription => (
        <li key={subscription.id}>{subscription.id}</li>
      ))}
    </ul>
  );
}

// Cancel a subscription
function CancelSubscriptionButton({ id }) {
  const { mutate, isLoading } = useCancelSubscription();

  const handleCancel = () => {
    mutate(id);
  };

  return (
    <button onClick={handleCancel} disabled={isLoading}>
      {isLoading ? 'Cancelling...' : 'Cancel Subscription'}
    </button>
  );
}
```

### Products

```tsx
import { useProducts, useProduct, useCreateProduct, useUpdateProduct, useDeleteProduct } from '@getpaidhq/react-sdk';

// Fetch a list of products
function ProductsList() {
  const { data, isLoading, error } = useProducts();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.products.map(product => (
        <li key={product.id}>{product.name}</li>
      ))}
    </ul>
  );
}
```

### Variants

```tsx
import { useVariants, useVariant, useCreateVariant, useUpdateVariant, useDeleteVariant } from '@getpaidhq/react-sdk';

// Fetch variants for a product
function ProductVariants({ productId }) {
  const { data, isLoading, error } = useVariants({ productId });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.variants.map(variant => (
        <li key={variant.id}>{variant.name}</li>
      ))}
    </ul>
  );
}
```

### Prices

```tsx
import { usePrices, usePrice, useCreatePrice, useUpdatePrice, useDeletePrice } from '@getpaidhq/react-sdk';

// Fetch prices for a product
function ProductPrices({ productId }) {
  const { data, isLoading, error } = usePrices({ productId });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.prices.map(price => (
        <li key={price.id}>${price.amount} / {price.interval}</li>
      ))}
    </ul>
  );
}
```

### Orders

```tsx
import { useOrders, useOrder, useCreateOrder, useUpdateOrder, useCancelOrder, useFulfillOrder } from '@getpaidhq/react-sdk';

// Fetch a list of orders
function OrdersList() {
  const { data, isLoading, error } = useOrders();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.orders.map(order => (
        <li key={order.id}>{order.id} - {order.status}</li>
      ))}
    </ul>
  );
}
```

### Invoices

```tsx
import { 
  useInvoices, 
  useInvoice, 
  useCreateInvoice, 
  useUpdateInvoice,
  useVoidInvoice,
  useFinalizeInvoice,
  usePayInvoice,
  useSendInvoice
} from '@getpaidhq/react-sdk';

// Fetch a list of invoices
function InvoicesList() {
  const { data, isLoading, error } = useInvoices();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.invoices.map(invoice => (
        <li key={invoice.id}>{invoice.id} - ${invoice.total}</li>
      ))}
    </ul>
  );
}
```

### Usage

```tsx
import { useUsageRecords, useUsageRecord, useCreateUsageRecord, useReportUsage, useUsageSummary } from '@getpaidhq/react-sdk';

// Fetch usage summary
function UsageSummary({ subscriptionId }) {
  const { data, isLoading, error } = useUsageSummary({ subscriptionId });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Usage Summary</h2>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
```

### Meters

```tsx
import { useMeters, useMeter, useCreateMeter, useUpdateMeter, useDeleteMeter, useMeterUsage } from '@getpaidhq/react-sdk';

// Fetch a list of meters
function MetersList() {
  const { data, isLoading, error } = useMeters();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.meters.map(meter => (
        <li key={meter.id}>{meter.name}</li>
      ))}
    </ul>
  );
}
```

### Dunning

```tsx
import { 
  useDunningAttempts, 
  useDunningAttempt, 
  useRetryDunningAttempt,
  useDunningProfiles,
  useDunningProfile,
  useCreateDunningProfile,
  useUpdateDunningProfile
} from '@getpaidhq/react-sdk';

// Fetch dunning profiles
function DunningProfilesList() {
  const { data, isLoading, error } = useDunningProfiles();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.profiles.map(profile => (
        <li key={profile.id}>{profile.name}</li>
      ))}
    </ul>
  );
}
```

### Webhooks

```tsx
import { useWebhooks, useWebhook, useCreateWebhook, useUpdateWebhook, useDeleteWebhook, useWebhookEvents, useTestWebhook } from '@getpaidhq/react-sdk';

// Fetch a list of webhooks
function WebhooksList() {
  const { data, isLoading, error } = useWebhooks();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.webhooks.map(webhook => (
        <li key={webhook.id}>{webhook.url} - {webhook.events.join(', ')}</li>
      ))}
    </ul>
  );
}
```

### Reports

```tsx
import { useRevenueReport, useSubscriptionReport, useCustomerReport, useMRRReport, useChurnReport, useLTVReport } from '@getpaidhq/react-sdk';

// Fetch MRR report
function MRRReportChart() {
  const { data, isLoading, error } = useMRRReport({ period: 'monthly' });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Monthly Recurring Revenue</h2>
      {/* Render chart with data */}
    </div>
  );
}
```

### Settings

```tsx
import { 
  useGeneralSettings, 
  useUpdateGeneralSettings, 
  useBillingSettings,
  useUpdateBillingSettings,
  useApiKeys,
  useCreateApiKey,
  useDeleteApiKey
} from '@getpaidhq/react-sdk';

// Fetch API keys
function ApiKeysList() {
  const { data, isLoading, error } = useApiKeys();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.apiKeys.map(key => (
        <li key={key.id}>{key.name} - Created: {key.createdAt}</li>
      ))}
    </ul>
  );
}
```

### Payments

```tsx
import { usePayments, usePayment, useCreatePayment, useRefundPayment, useCapturePayment } from '@getpaidhq/react-sdk';

// Fetch a list of payments
function PaymentsList() {
  const { data, isLoading, error } = usePayments();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {data.payments.map(payment => (
        <li key={payment.id}>{payment.id} - ${payment.amount}</li>
      ))}
    </ul>
  );
}

// Refund a payment
function RefundPaymentButton({ id }) {
  const { mutate, isLoading } = useRefundPayment();

  const handleRefund = () => {
    mutate({ id, amount: 1000 }); // Refund $10.00
  };

  return (
    <button onClick={handleRefund} disabled={isLoading}>
      {isLoading ? 'Refunding...' : 'Refund Payment'}
    </button>
  );
}
```

## Server Components (Next.js)

For server components in Next.js, you can use the @getpaidhq/sdk directly:

```tsx
import { GetPaidHQClient } from '@getpaidhq/sdk';

export default async function CustomerPage({ params }) {
  const client = new GetPaidHQClient({
    apiKey: process.env.GETPAIDHQ_API_KEY,
  });

  const customer = await client.customers.get(params.id);

  return (
    <div>
      <h1>{customer.name}</h1>
      <p>{customer.email}</p>

      {/* Pass data to client components */}
      <ClientComponent initialData={customer} />
    </div>
  );
}
```

## Implementing Additional Resources

This SDK provides hooks for the most commonly used resources. To implement hooks for additional resources, follow the pattern established in the existing hooks:

1. Create a new file in the `src/hooks` directory (e.g., `use-invoices.ts`)
2. Define query keys for the resource
3. Implement hooks for common operations (list, get, create, update, delete)
4. Export the hooks from the file
5. Add the export to `src/hooks/index.ts`

Example:

```tsx
// src/hooks/use-invoices.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client';
import { QueryOptions } from '../types';

// Query keys for invoices
export const invoiceKeys = {
  all: ['invoices'] as const,
  lists: () => [...invoiceKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...invoiceKeys.lists(), filters] as const,
  details: () => [...invoiceKeys.all, 'detail'] as const,
  detail: (id: string) => [...invoiceKeys.details(), id] as const,
};

// Implement hooks for invoices
export function useInvoices(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: invoiceKeys.list(params || {}),
    queryFn: () => client.invoices.list(params),
    ...options,
  });
}

// ... implement other hooks for invoices
```

Then add to `src/hooks/index.ts`:

```tsx
export * from './use-invoices';
```

## API Reference

For a complete list of available resources and operations, refer to the [GetPaidHQ API documentation](https://api.getpaidhq.com).
