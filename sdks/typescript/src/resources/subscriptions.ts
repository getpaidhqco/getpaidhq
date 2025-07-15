import type { BaseClient } from '@/client/base-client'
import type {
  Subscription,
  CreateSubscriptionRequest,
  UpdateSubscriptionRequest,
  PaginatedResponse,
  ListSubscriptionsParams
} from '@/types'

export class SubscriptionsResource {
  constructor(private client: BaseClient) {}

  /**
   * Create a new subscription
   */
  async create(data: CreateSubscriptionRequest): Promise<Subscription> {
    return this.client.post<Subscription>('/api/subscriptions', data)
  }

  /**
   * Retrieve a subscription by ID
   */
  async find(id: string): Promise<Subscription> {
    return this.client.get<Subscription>(`/api/subscriptions/${id}`)
  }

  /**
   * List subscriptions with pagination
   */
  async list(params?: ListSubscriptionsParams): Promise<PaginatedResponse<Subscription>> {
    const query = new URLSearchParams()
    if (params?.limit) query.set('limit', params.limit.toString())
    if (params?.offset) query.set('offset', params.offset.toString())
    if (params?.status) query.set('status', params.status)
    if (params?.customerId) query.set('customer_id', params.customerId)

    const queryString = query.toString()
    const url = `/api/subscriptions${queryString ? `?${queryString}` : ''}`

    return this.client.get<PaginatedResponse<Subscription>>(url)
  }

  /**
   * Update a subscription
   */
  async update(id: string, data: UpdateSubscriptionRequest): Promise<Subscription> {
    return this.client.put<Subscription>(`/api/subscriptions/${id}`, data)
  }

  /**
   * Cancel a subscription
   */
  async cancel(id: string, options?: {
    cancelMode?: 'immediate' | 'end_of_period'
    prorationMode?: 'none' | 'credit_unused'
  }): Promise<Subscription> {
    return this.client.post<Subscription>(`/api/subscriptions/${id}/cancel`, options)
  }

  /**
   * Pause a subscription
   */
  async pause(id: string, options?: {
    pauseMode?: 'immediate' | 'end_of_period'
    resumeAt?: string
  }): Promise<Subscription> {
    return this.client.post<Subscription>(`/api/subscriptions/${id}/pause`, options)
  }

  /**
   * Resume a paused subscription
   */
  async resume(id: string, options?: {
    prorationMode?: 'none' | 'credit_unused'
  }): Promise<Subscription> {
    return this.client.post<Subscription>(`/api/subscriptions/${id}/resume`, options)
  }

  /**
   * Change subscription plan
   */
  async changePlan(id: string, options: {
    newPriceId: string
    prorationMode?: 'none' | 'credit_unused'
    effectiveDate?: string
  }): Promise<Subscription> {
    return this.client.post<Subscription>(`/api/subscriptions/${id}/change-plan`, options)
  }

  /**
   * Get subscription usage for a period
   */
  async getUsage(id: string, options?: {
    startDate?: string
    endDate?: string
  }): Promise<any> {
    const query = new URLSearchParams()
    if (options?.startDate) query.set('start_date', options.startDate)
    if (options?.endDate) query.set('end_date', options.endDate)

    const queryString = query.toString()
    const url = `/api/subscriptions/${id}/usage${queryString ? `?${queryString}` : ''}`

    return this.client.get(url)
  }
}