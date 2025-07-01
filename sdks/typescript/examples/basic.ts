import { GetPaidHQ } from '@getpaidhq/sdk'

// Initialize the client
const client = new GetPaidHQ({
  apiKey: 'your-api-key',
  baseURL: 'https://api.getpaidhq.com', // Optional
  debug: true // Optional
})

async function basicExample() {
  try {
    // Test connection
    const health = await client.ping()
    console.log('API Status:', health.status)

    // Create a customer
    const customer = await client.customers.create({
      email: 'customer@example.com',
      name: 'John Doe'
    })

    // Create a subscription
    const subscription = await client.subscriptions.create({
      customerId: customer.id,
      paymentMethodId: 'pm_123',
      items: [
        { priceId: 'price_monthly', quantity: 1 }
      ]
    })

    console.log('Subscription created:', subscription.id)

  } catch (error) {
    console.error('Error:', error.message)
  }
}

basicExample()