#!/usr/bin/env npx tsx

/**
 * GetPaidHQ SDK Usage Examples
 * 
 * This file demonstrates how to use the GetPaidHQ TypeScript SDK
 * with examples matching the original specification.
 */

import { GetPaidHQClient } from '../src';

// Initialize the client
const getpaidhq = new GetPaidHQClient({ 
  apiKey: 'sk_test_your_api_key_here',
  // For test environment:
  baseURL: 'https://api.test.getpaidhq.co'
});

async function main() {
  try {
    console.log('🚀 GetPaidHQ SDK Examples\n');

    // Health check
    console.log('📊 Health Check');
    const health = await getpaidhq.healthCheck();
    console.log('API Status:', health.status);
    console.log('');

    // CREATE a customer
    console.log('👤 Creating Customer');
    const newCustomer = await getpaidhq.customers.create({
      email: 'john@example.com',
      first_name: 'John',
      last_name: 'Doe',
      phone: '+27123456789',
      company: 'Tech Corp',
      billing_address: {
        line1: '123 Main St',
        city: 'Cape Town',
        country: 'ZA',
      }
    });
    console.log('Created customer:', newCustomer.id);
    console.log('');

    // READ a customer
    console.log('🔍 Reading Customer');
    const customer = await getpaidhq.customers.get(newCustomer.id);
    console.log('Customer:', customer.email, customer.company);
    console.log('');

    // LIST customers with filters
    console.log('📋 Listing Customers');
    const customers = await getpaidhq.customers.list({
      page: 0,
      limit: 20,
      status: 'active',
      email: 'john@example.com'
    });
    console.log('Found customers:', customers.data.length);
    console.log('Total customers:', customers.meta.total);
    console.log('');

    // CREATE a product and pricing
    console.log('📦 Creating Product');
    const product = await getpaidhq.products.create({
      name: 'Premium Plan',
      description: 'Our premium subscription plan',
      type: 'service'
    });

    const variant = await getpaidhq.products.createVariant(product.id, {
      name: 'Monthly',
      description: 'Monthly billing'
    });

    const price = await getpaidhq.prices.create({
      variant_id: variant.id,
      currency: 'USD',
      type: 'recurring',
      billing_scheme: 'per_unit',
      amount: 9999, // $99.99 in cents
      recurring: {
        interval: 'month',
        interval_count: 1,
        trial_period_days: 14
      }
    });

    console.log('Created product:', product.name);
    console.log('Created price:', price.id, '$99.99/month');
    console.log('');

    // CREATE a subscription
    console.log('📅 Creating Subscription');
    const subscription = await getpaidhq.subscriptions.create({
      customer_id: customer.id,
      items: [
        {
          price_id: price.id,
          quantity: 1,
          description: 'Premium Plan Subscription'
        }
      ],
      trial_period_days: 14
    });
    console.log('Created subscription:', subscription.id);
    console.log('Status:', subscription.status);
    console.log('');

    // UPDATE customer
    console.log('✏️ Updating Customer');
    const updated = await getpaidhq.customers.update(customer.id, {
      email: 'john.doe@newemail.com',
      company: 'New Company Ltd'
    });
    console.log('Updated email:', updated.email);
    console.log('Updated company:', updated.company);
    console.log('');

    // Subscription management
    console.log('⏸️ Subscription Management');
    
    // Pause subscription
    await getpaidhq.subscriptions.pause(subscription.id, {
      behavior: 'keep_as_draft'
    });
    console.log('Subscription paused');

    // Resume subscription
    await getpaidhq.subscriptions.resume(subscription.id, {
      billing_cycle_anchor: 'unchanged'
    });
    console.log('Subscription resumed');
    console.log('');

    // Usage-based billing example
    console.log('📊 Usage-Based Billing');
    
    // Create a usage-based price
    const usagePrice = await getpaidhq.prices.create({
      variant_id: variant.id,
      currency: 'USD',
      type: 'usage_based',
      billing_scheme: 'per_unit',
      usage: {
        unit: 'api_calls',
        aggregation_method: 'sum',
        unit_amount: 10, // $0.10 per API call
      }
    });

    // Create subscription with usage pricing
    const usageSubscription = await getpaidhq.subscriptions.create({
      customer_id: customer.id,
      items: [
        {
          price_id: usagePrice.id,
          quantity: 1,
          description: 'API Usage'
        }
      ]
    });

    // Record usage
    await getpaidhq.usage.recordEvent({
      subscription_item_id: usageSubscription.items[0].id,
      quantity: 100,
      timestamp: new Date().toISOString(),
      action: 'increment',
      idempotency_key: `usage-${Date.now()}`
    });

    console.log('Recorded 100 API calls');
    console.log('');

    // Invoice management
    console.log('🧾 Invoice Management');
    
    const invoice = await getpaidhq.invoices.create({
      customer_id: customer.id,
      line_items: [
        {
          description: 'One-time setup fee',
          quantity: 1,
          unit_amount: 5000, // $50.00
        }
      ],
      due_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString()
    });

    console.log('Created invoice:', invoice.id);
    console.log('Total amount:', `$${invoice.total / 100}`);
    console.log('');

    // Additional methods demonstration
    console.log('🔧 Additional Methods');
    
    // Suspend customer
    await getpaidhq.customers.suspend(customer.id);
    console.log('Customer suspended');

    // Activate customer
    await getpaidhq.customers.activate(customer.id);
    console.log('Customer activated');

    // List customer invoices
    const customerInvoices = await getpaidhq.customers.listInvoices(customer.id);
    console.log('Customer invoices:', customerInvoices.data.length);
    
    console.log('');
    console.log('✅ All examples completed successfully!');

  } catch (error) {
    console.error('❌ Error running examples:', error);
    
    if (error instanceof Error) {
      console.error('Message:', error.message);
      
      // Type-safe error handling
      const typedError = error as any;
      if (typedError.statusCode) {
        console.error('Status Code:', typedError.statusCode);
      }
      if (typedError.details) {
        console.error('Details:', typedError.details);
      }
    }
  }
}

// Run examples if this file is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { main as runExamples };
