import type { BaseClient } from '@/client/base-client'
import type {
  UsageRecord,
  RecordUsageRequest,
  BatchRecordUsageRequest,
  UsageSummary,
  PaginatedResponse
} from '@/types'

export class UsageResource {
  constructor(private client: BaseClient) {}

  /**
   * Record usage for a subscription item
   */
  async record(data: RecordUsageRequest): Promise<UsageRecord> {
    return this.client.post<UsageRecord>('/api/usage-records', data)
  }

  /**
   * Record multiple usage events in batch
   */
  async batchRecord(data: BatchRecordUsageRequest): Promise<PaginatedResponse<UsageRecord>> {
    return this.client.post<PaginatedResponse<UsageRecord>>('/api/usage-records/batch', data)
  }

  /**
   * List usage records for a subscription item
   */
  async list(subscriptionItemId: string, options?: {
    limit?: number
    offset?: number
  }): Promise<PaginatedResponse<UsageRecord>> {
    const query = new URLSearchParams({ subscription_item_id: subscriptionItemId })
    if (options?.limit) query.set('limit', options.limit.toString())
    if (options?.offset) query.set('offset', options.offset.toString())

    return this.client.get<PaginatedResponse<UsageRecord>>(`/api/usage-records?${query}`)
  }

  /**
   * Get usage summary for a subscription item
   */
  async getSummary(subscriptionItemId: string, options?: {
    startDate?: string
    endDate?: string
    granularity?: 'hour' | 'day' | 'week' | 'month'
  }): Promise<UsageSummary> {
    const query = new URLSearchParams()
    if (options?.startDate) query.set('start_date', options.startDate)
    if (options?.endDate) query.set('end_date', options.endDate)
    if (options?.granularity) query.set('granularity', options.granularity)

    const queryString = query.toString()
    const url = `/api/subscription-items/${subscriptionItemId}/usage-summary${queryString ? `?${queryString}` : ''}`

    return this.client.get<UsageSummary>(url)
  }

  /**
   * Delete a usage record (for corrections)
   */
  async delete(id: string): Promise<void> {
    return this.client.delete(`/api/usage-records/${id}`)
  }
}