/**
 * Enhanced GetPaidHQ SDK Client
 * Provides a developer-friendly interface with additional conveniences
 */

import { Configuration, ConfigurationParameters } from './configuration'
import { 
  CustomersApi, 
  SubscriptionsApi, 
  InvoicesApi, 
  ProductsApi, 
  PricesApi,
  OrdersApi,
  PaymentsApi
} from './api'
import { SubscriptionBuilder } from './builders/subscription-builder'
import { UsageRecorder } from './utilities/usage-recorder'
import { WebhookVerifier } from './utilities/webhook-verifier'
import { AmountFormatter } from './utilities/amount-formatter'
import { RetryWrapper } from './utilities/retry-wrapper'

export interface GetPaidHQConfig extends ConfigurationParameters {
  /** Automatically retry failed requests */
  enableRetries?: boolean
  /** Maximum number of retry attempts */
  maxRetries?: number
  /** Enable request/response logging */
  debug?: boolean
}

export class GetPaidHQ {
  private config: Configuration
  private retryWrapper?: RetryWrapper

  // Enhanced API clients
  public readonly customers: CustomersApi
  public readonly subscriptions: SubscriptionsApi
  public readonly invoices: InvoicesApi
  public readonly products: ProductsApi
  public readonly prices: PricesApi
  public readonly orders: OrdersApi
  public readonly payments: PaymentsApi

  // Utility classes
  public readonly usage: UsageRecorder
  public readonly webhooks: WebhookVerifier
  public readonly amounts: AmountFormatter

  constructor(config: GetPaidHQConfig) {
    this.config = new Configuration(config)
    
    // Setup retry wrapper if enabled
    if (config.enableRetries !== false) {
      this.retryWrapper = new RetryWrapper({
        maxRetries: config.maxRetries || 3,
        debug: config.debug || false
      })
    }

    // Initialize API clients
    this.customers = new CustomersApi(this.config)
    this.subscriptions = new SubscriptionsApi(this.config)
    this.invoices = new InvoicesApi(this.config)
    this.products = new ProductsApi(this.config)
    this.prices = new PricesApi(this.config)
    this.orders = new OrdersApi(this.config)
    this.payments = new PaymentsApi(this.config)

    // Initialize utilities
    this.usage = new UsageRecorder(this.subscriptions)
    this.webhooks = new WebhookVerifier()
    this.amounts = new AmountFormatter()

    // Wrap APIs with retry logic if enabled
    if (this.retryWrapper) {
      this.wrapAPIsWithRetry()
    }
  }

  /**
   * Create a subscription using the builder pattern
   */
  createSubscription() {
    return new SubscriptionBuilder(this.subscriptions)
  }

  /**
   * Quick subscription creation for simple cases
   */
  async quickSubscription(customerId: string, priceId: string, options?: {
    quantity?: number
    trialDays?: number
    metadata?: Record<string, any>
  }) {
    return this.createSubscription()
      .customer(customerId)
      .addItem(priceId, options?.quantity || 1)
      .withTrial(options?.trialDays)
      .withMetadata(options?.metadata || {})
      .create()
  }

  private wrapAPIsWithRetry() {
    if (!this.retryWrapper) return

    // Wrap each API method with retry logic
    const apiClients = [
      this.customers, this.subscriptions, this.invoices,
      this.products, this.prices, this.orders, this.payments
    ]

    apiClients.forEach(client => {
      Object.getOwnPropertyNames(Object.getPrototypeOf(client))
        .filter(name => name !== 'constructor' && typeof (client as any)[name] === 'function')
        .forEach(methodName => {
          const originalMethod = (client as any)[methodName]
          ;(client as any)[methodName] = this.retryWrapper!.wrap(originalMethod.bind(client))
        })
    })
  }
}

// Re-export types for convenience
export * from './models'
export * from './api'
export { Configuration } from './configuration'

// Enhanced exports
export { SubscriptionBuilder } from './builders/subscription-builder'
export { UsageRecorder } from './utilities/usage-recorder'
export { WebhookVerifier } from './utilities/webhook-verifier'
export { AmountFormatter } from './utilities/amount-formatter'
