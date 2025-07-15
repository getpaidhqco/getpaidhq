/**
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
    console.log(`Processing batch of ${records.length} usage records`)
  }
}
