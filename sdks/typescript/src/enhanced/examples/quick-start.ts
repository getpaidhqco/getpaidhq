/**
 * Quick Start Examples for GetPaidHQ SDK
 */

import { GetPaidHQ } from '../client'

// Initialize the client
const client = new GetPaidHQ({
  apiKey: process.env.GETPAIDHQ_API_KEY!,
  basePath: 'https://api.getpaidhq.com/v1'
})

// Example 1: Create a simple subscription
async function createSimpleSubscription() {
  const subscription = await client.quickSubscription(
    'cus_customer123',
    'price_monthly99',
    {
      quantity: 1,
      trialDays: 14,
      metadata: { source: 'website_signup' }
    }
  )
  
  console.log('Created subscription:', subscription.id)
  return subscription
}

// Example 2: Create complex subscription with builder
async function createComplexSubscription() {
  const subscription = await client.createSubscription()
    .customer('cus_customer123')
    .addItem('price_base_plan', 1)
    .addItem('price_addon_storage', 5)
    .withTrial(30)
    .withPaymentMethod('pm_card123')
    .billedEvery(1, 'month')
    .withMetadata({
      plan_type: 'enterprise',
      sales_rep: 'john@company.com'
    })
    .create()
    
  console.log('Created complex subscription:', subscription.id)
  return subscription
}

// Example 3: Record usage
async function recordUsage() {
  await client.usage.record({
    subscriptionId: 'sub_123',
    subscriptionItemId: 'si_456',
    quantity: 1000, // 1000 API calls
    timestamp: new Date(),
    metadata: { endpoint: '/api/users' }
  })
  
  console.log('Usage recorded')
}

// Example 4: Batch record usage
async function batchRecordUsage() {
  // These will be automatically batched and sent
  client.usage.batchRecord({
    subscriptionId: 'sub_123',
    subscriptionItemId: 'si_456',
    quantity: 100
  })
  
  client.usage.batchRecord({
    subscriptionId: 'sub_123',
    subscriptionItemId: 'si_456',
    quantity: 250
  })
  
  // Manually flush if needed
  await client.usage.flush()
}

// Example 5: Handle webhook
async function handleWebhook(payload: string, signature: string) {
  const isValid = client.webhooks.verify(
    payload,
    signature,
    process.env.WEBHOOK_SECRET!
  )
  
  if (!isValid) {
    throw new Error('Invalid webhook signature')
  }
  
  const event = JSON.parse(payload)
  console.log('Received webhook:', event.type)
}

// Example 6: Format amounts
function formatAmounts() {
  const amount = client.amounts.dollarsTocents(99.99) // 9999
  const formatted = client.amounts.formatCurrency(9999) // "$99.99"
  const parsed = client.amounts.parseCurrency("$99.99") // 9999
  
  console.log({ amount, formatted, parsed })
}

// Example 7: Iterate through all subscriptions
async function listAllSubscriptions() {
  for await (const subscription of client.subscriptions.listAll()) {
    console.log('Subscription:', subscription.id, subscription.status)
  }
}

export {
  createSimpleSubscription,
  createComplexSubscription,
  recordUsage,
  batchRecordUsage,
  handleWebhook,
  formatAmounts,
  listAllSubscriptions
}
