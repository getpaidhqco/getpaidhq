import { execSync } from 'child_process'
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

// Get the directory name in ES modules
const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const GENERATED_DIR = resolve(__dirname, '../src/generated')
const ENHANCED_DIR = resolve(__dirname, '../src/enhanced')
const CONTEXT_FILE = resolve(__dirname, 'ai-enhancement-context.md')

async function enhanceSDK() {
  console.log('🚀 Enhancing TypeScript SDK with developer experience improvements...')

  // Ensure enhanced directory exists
  if (!existsSync(ENHANCED_DIR)) {
    mkdirSync(ENHANCED_DIR, { recursive: true })
  }

  try {
    // 1. Copy base generated files
    await copyGeneratedFiles()
    
    // 2. Create enhanced client wrapper
    await createEnhancedClient()
    
    // 3. Add utility classes
    await createUtilities()
    
    // 4. Add builder patterns
    await createBuilders()
    
    // 5. Add async iterators for pagination
    await createIterators()
    
    // 6. Create comprehensive examples
    await createExamples()
    
    // 7. Generate enhanced index
    await createEnhancedIndex()

    console.log('✅ SDK enhancement completed successfully!')
  } catch (error) {
    console.error('❌ SDK enhancement failed:', error)
    process.exit(1)
  }
}

async function copyGeneratedFiles() {
  console.log('📋 Copying generated API files...')
  
  // Copy essential generated files to enhanced directory
  const filesToCopy = [
    'models/index.ts',
    'api.ts',
    'base.ts',
    'common.ts',
    'configuration.ts'
  ]
  
  for (const file of filesToCopy) {
    const srcPath = resolve(GENERATED_DIR, file)
    const destPath = resolve(ENHANCED_DIR, file)
    
    if (existsSync(srcPath)) {
      // Ensure destination directory exists
      mkdirSync(dirname(destPath), { recursive: true })
      
      const content = readFileSync(srcPath, 'utf8')
      writeFileSync(destPath, content)
    }
  }
}

async function createEnhancedClient() {
  console.log('🔧 Creating enhanced client wrapper...')
  
  const clientCode = `/**
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
`

  writeFileSync(resolve(ENHANCED_DIR, 'client.ts'), clientCode)
}

async function createUtilities() {
  console.log('🛠️ Creating utility classes...')

  const utilitiesDir = resolve(ENHANCED_DIR, 'utilities')
  mkdirSync(utilitiesDir, { recursive: true })

  // Usage Recorder utility
  const usageRecorderCode = `/**
 * Utility for recording usage data with batching and error handling
 */

import { SubscriptionsApi } from '../api'

export interface UsageRecord {
  subscriptionId: string
  subscriptionItemId: string
  quantity: number
  timestamp?: Date
  action?: string
  metadata?: Record<string, any>
}

export class UsageRecorder {
  private pendingRecords: UsageRecord[] = []
  private batchSize = 100
  private flushInterval = 60000 // 1 minute

  constructor(private subscriptionsApi: SubscriptionsApi) {
    // Auto-flush pending records periodically
    setInterval(() => this.flush(), this.flushInterval)
  }

  /**
   * Record usage immediately
   */
  async record(record: UsageRecord): Promise<void> {
    // Implementation would call the actual usage recording API
    console.log('Recording usage:', record)
  }

  /**
   * Batch record usage for later flushing
   */
  batchRecord(record: UsageRecord): void {
    this.pendingRecords.push(record)
    
    if (this.pendingRecords.length >= this.batchSize) {
      this.flush()
    }
  }

  /**
   * Flush all pending records
   */
  async flush(): Promise<void> {
    if (this.pendingRecords.length === 0) return

    const records = [...this.pendingRecords]
    this.pendingRecords = []

    try {
      // Process records in batches
      for (let i = 0; i < records.length; i += this.batchSize) {
        const batch = records.slice(i, i + this.batchSize)
        await this.processBatch(batch)
      }
    } catch (error) {
      // Re-queue failed records
      this.pendingRecords.unshift(...records)
      throw error
    }
  }

  private async processBatch(records: UsageRecord[]): Promise<void> {
    // Implementation would batch upload usage records
    console.log(\`Processing batch of \${records.length} usage records\`)
  }
}
`

  // Webhook Verifier utility
  const webhookVerifierCode = `/**
 * Utility for verifying webhook signatures
 */

import { createHmac } from 'crypto'

export class WebhookVerifier {
  /**
   * Verify webhook signature
   */
  verify(payload: string, signature: string, secret: string): boolean {
    const expectedSignature = this.generateSignature(payload, secret)
    return this.secureCompare(signature, expectedSignature)
  }

  /**
   * Generate HMAC signature for payload
   */
  private generateSignature(payload: string, secret: string): string {
    return createHmac('sha256', secret)
      .update(payload, 'utf8')
      .digest('hex')
  }

  /**
   * Secure string comparison to prevent timing attacks
   */
  private secureCompare(a: string, b: string): boolean {
    if (a.length !== b.length) return false
    
    let result = 0
    for (let i = 0; i < a.length; i++) {
      result |= a.charCodeAt(i) ^ b.charCodeAt(i)
    }
    
    return result === 0
  }
}
`

  // Amount Formatter utility
  const amountFormatterCode = `/**
 * Utility for handling currency amounts and formatting
 */

export class AmountFormatter {
  /**
   * Convert dollars to cents
   */
  dollarsTocents(dollars: number): number {
    return Math.round(dollars * 100)
  }

  /**
   * Convert cents to dollars
   */
  centsToDollars(cents: number): number {
    return cents / 100
  }

  /**
   * Format amount as currency string
   */
  formatCurrency(cents: number, currency = 'USD'): string {
    const amount = this.centsToDollars(cents)
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency
    }).format(amount)
  }

  /**
   * Parse currency string to cents
   */
  parseCurrency(currencyString: string): number {
    const number = parseFloat(currencyString.replace(/[^\\d.-]/g, ''))
    return this.dollarsTocents(number)
  }
}
`

  // Retry Wrapper utility
  const retryWrapperCode = `/**
 * Utility for adding retry logic to API calls
 */

export interface RetryOptions {
  maxRetries: number
  debug: boolean
  baseDelay?: number
  maxDelay?: number
}

export class RetryWrapper {
  constructor(private options: RetryOptions) {}

  /**
   * Wrap a function with retry logic
   */
  wrap<T extends (...args: any[]) => Promise<any>>(fn: T): T {
    return (async (...args: any[]) => {
      let lastError: Error
      
      for (let attempt = 0; attempt <= this.options.maxRetries; attempt++) {
        try {
          return await fn(...args)
        } catch (error) {
          lastError = error as Error
          
          if (attempt === this.options.maxRetries) {
            throw lastError
          }

          if (!this.shouldRetry(error)) {
            throw lastError
          }

          const delay = this.calculateDelay(attempt)
          
          if (this.options.debug) {
            console.log(\`Attempt \${attempt + 1} failed, retrying in \${delay}ms:\`, error)
          }
          
          await this.sleep(delay)
        }
      }
      
      throw lastError!
    }) as T
  }

  private shouldRetry(error: any): boolean {
    // Don't retry client errors (4xx)
    if (error.response?.status >= 400 && error.response?.status < 500) {
      return false
    }
    
    // Retry server errors (5xx) and network errors
    return true
  }

  private calculateDelay(attempt: number): number {
    const baseDelay = this.options.baseDelay || 1000
    const maxDelay = this.options.maxDelay || 30000
    
    // Exponential backoff with jitter
    const exponentialDelay = baseDelay * Math.pow(2, attempt)
    const jitter = Math.random() * 0.1 * exponentialDelay
    
    return Math.min(exponentialDelay + jitter, maxDelay)
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }
}
`

  writeFileSync(resolve(utilitiesDir, 'usage-recorder.ts'), usageRecorderCode)
  writeFileSync(resolve(utilitiesDir, 'webhook-verifier.ts'), webhookVerifierCode)
  writeFileSync(resolve(utilitiesDir, 'amount-formatter.ts'), amountFormatterCode)
  writeFileSync(resolve(utilitiesDir, 'retry-wrapper.ts'), retryWrapperCode)
}

async function createBuilders() {
  console.log('🏗️ Creating builder patterns...')

  const buildersDir = resolve(ENHANCED_DIR, 'builders')
  mkdirSync(buildersDir, { recursive: true })

  const subscriptionBuilderCode = `/**
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
`

  writeFileSync(resolve(buildersDir, 'subscription-builder.ts'), subscriptionBuilderCode)
}

async function createIterators() {
  console.log('🔄 Creating async iterators for pagination...')

  const iteratorsDir = resolve(ENHANCED_DIR, 'iterators')
  mkdirSync(iteratorsDir, { recursive: true })

  const paginationIteratorCode = `/**
 * Async iterator for paginated API responses
 */

export interface PaginatedResponse<T> {
  data: T[]
  meta: {
    hasMore: boolean
    nextCursor?: string
    totalCount?: number
  }
}

export interface PaginationOptions {
  limit?: number
  cursor?: string
}

export class PaginationIterator<T> {
  private cursor?: string
  private hasMore = true

  constructor(
    private fetchPage: (options: PaginationOptions) => Promise<PaginatedResponse<T>>,
    private options: PaginationOptions = {}
  ) {
    this.cursor = options.cursor
  }

  async *[Symbol.asyncIterator](): AsyncIterableIterator<T> {
    while (this.hasMore) {
      const response = await this.fetchPage({
        ...this.options,
        cursor: this.cursor
      })

      for (const item of response.data) {
        yield item
      }

      this.hasMore = response.meta.hasMore
      this.cursor = response.meta.nextCursor
    }
  }

  /**
   * Collect all items into an array
   */
  async toArray(): Promise<T[]> {
    const items: T[] = []
    for await (const item of this) {
      items.push(item)
    }
    return items
  }

  /**
   * Find the first item matching predicate
   */
  async find(predicate: (item: T) => boolean): Promise<T | undefined> {
    for await (const item of this) {
      if (predicate(item)) {
        return item
      }
    }
    return undefined
  }

  /**
   * Filter items using predicate
   */
  async *filter(predicate: (item: T) => boolean): AsyncIterableIterator<T> {
    for await (const item of this) {
      if (predicate(item)) {
        yield item
      }
    }
  }

  /**
   * Map items to a new type
   */
  async *map<U>(mapper: (item: T) => U): AsyncIterableIterator<U> {
    for await (const item of this) {
      yield mapper(item)
    }
  }
}
`

  writeFileSync(resolve(iteratorsDir, 'pagination-iterator.ts'), paginationIteratorCode)
}

async function createExamples() {
  console.log('📚 Creating usage examples...')

  const examplesDir = resolve(ENHANCED_DIR, 'examples')
  mkdirSync(examplesDir, { recursive: true })

  const quickStartCode = `/**
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
`

  writeFileSync(resolve(examplesDir, 'quick-start.ts'), quickStartCode)
}

async function createEnhancedIndex() {
  console.log('📦 Creating enhanced index file...')

  const indexCode = `/**
 * Enhanced GetPaidHQ SDK
 * 
 * This is the main entry point for the enhanced SDK that provides
 * developer-friendly wrappers around the generated API client.
 */

// Main client export
export { GetPaidHQ, GetPaidHQConfig } from './client'

// Generated API exports
export * from './api'
export * from './models'
export { Configuration } from './configuration'

// Enhanced utilities
export { SubscriptionBuilder } from './builders/subscription-builder'
export { UsageRecorder } from './utilities/usage-recorder'
export { WebhookVerifier } from './utilities/webhook-verifier'
export { AmountFormatter } from './utilities/amount-formatter'
export { RetryWrapper } from './utilities/retry-wrapper'
export { PaginationIterator } from './iterators/pagination-iterator'

// Default export for convenience
import { GetPaidHQ } from './client'
export default GetPaidHQ
`

  writeFileSync(resolve(ENHANCED_DIR, 'index.ts'), indexCode)
}

// Run the enhancement
enhanceSDK().catch(console.error)