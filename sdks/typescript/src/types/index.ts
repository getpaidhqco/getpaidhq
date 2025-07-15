// Re-export generated types with enhancements
export * from '@/generated/models'

// Enhanced types for better developer experience
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit?: number
  offset?: number
  hasMore?: boolean
}

export interface ListParams {
  limit?: number
  offset?: number
}

export interface DateRange {
  startDate?: string
  endDate?: string
}

// Subscription-specific types
export interface ListSubscriptionsParams extends ListParams {
  status?: 'active' | 'paused' | 'cancelled' | 'trial'
  customerId?: string
}

export interface CreateSubscriptionRequest {
  customerId: string
  paymentMethodId: string
  items: SubscriptionItemRequest[]
  trialDays?: number
  metadata?: Record<string, string>
}

export interface SubscriptionItemRequest {
  priceId: string
  quantity?: number
}

// Usage-specific types
export interface RecordUsageRequest {
  subscriptionItemId: string
  quantity: number
  transactionValue?: number
  percentageRate?: number
  timestamp: string
  referenceId?: string
  referenceType?: string
  metadata?: Record<string, string>
}

export interface BatchRecordUsageRequest {
  usageRecords: RecordUsageRequest[]
}

// Utility types for better DX
export type SubscriptionStatus = 'trial' | 'active' | 'paused' | 'cancelled' | 'past_due' | 'unpaid'
export type BillingInterval = 'day' | 'week' | 'month' | 'year'
export type Currency = 'USD' | 'EUR' | 'GBP' | 'CAD' | 'AUD'