/**
 * Builder pattern for creating subscriptions with a fluent API
 */

import { SubscriptionsApi } from '../api'
import { CreateSubscriptionRequest } from '../models'

export class SubscriptionBuilder {
  private subscriptionData: Partial<CreateSubscriptionRequest> = {
    items: []
  }

  constructor(private subscriptionsApi: SubscriptionsApi) {}

  /**
   * Set the customer for this subscription
   */
  customer(customerId: string): this {
    this.subscriptionData.customerId = customerId
    return this
  }

  /**
   * Add an item to the subscription
   */
  addItem(priceId: string, quantity = 1): this {
    if (!this.subscriptionData.items) {
      this.subscriptionData.items = []
    }
    
    this.subscriptionData.items.push({
      priceId,
      quantity
    })
    return this
  }

  /**
   * Set trial period in days
   */
  withTrial(days?: number): this {
    if (days) {
      const trialEnd = new Date()
      trialEnd.setDate(trialEnd.getDate() + days)
      this.subscriptionData.trialEndsAt = trialEnd.toISOString()
    }
    return this
  }

  /**
   * Add metadata to the subscription
   */
  withMetadata(metadata: Record<string, any>): this {
    this.subscriptionData.metadata = {
      ...this.subscriptionData.metadata,
      ...metadata
    }
    return this
  }

  /**
   * Set payment method
   */
  withPaymentMethod(paymentMethodId: string): this {
    this.subscriptionData.paymentMethodId = paymentMethodId
    return this
  }

  /**
   * Set billing interval
   */
  billedEvery(count: number, interval: 'day' | 'week' | 'month' | 'year'): this {
    this.subscriptionData.billingInterval = interval
    this.subscriptionData.billingIntervalQty = count
    return this
  }

  /**
   * Create the subscription
   */
  async create() {
    if (!this.subscriptionData.customerId) {
      throw new Error('Customer ID is required')
    }
    
    if (!this.subscriptionData.items?.length) {
      throw new Error('At least one subscription item is required')
    }

    // Call the actual API to create subscription
    // This would need to be implemented based on your actual API
    return this.subscriptionsApi.createSubscription(this.subscriptionData as CreateSubscriptionRequest)
  }
}
