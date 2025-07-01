import { GetPaidHQ } from '@getpaidhq/sdk'

const client = new GetPaidHQ({ apiKey: 'your-api-key' })

async function usageBillingExample() {
  // Record API usage
  await client.usage.record({
    subscriptionItemId: 'si_123',
    quantity: 100,
    timestamp: new Date().toISOString(),
    referenceId: 'api-call-batch-1',
    metadata: {
      endpoint: '/api/users',
      method: 'GET'
    }
  })

  // Record transaction usage (for payment processing fees)
  await client.usage.record({
    subscriptionItemId: 'si_456',
    quantity: 1, // 1 transaction
    transactionValue: 10000, // $100.00 in cents
    percentageRate: 2.9, // 2.9%
    timestamp: new Date().toISOString(),
    referenceId: 'payment-txn-789',
    referenceType: 'payment'
  })

  // Get usage summary
  const summary = await client.usage.getSummary('si_123', {
    startDate: '2024-01-01T00:00:00Z',
    endDate: '2024-01-31T23:59:59Z',
    granularity: 'day'
  })

  console.log('Usage summary:', summary)
}

usageBillingExample()