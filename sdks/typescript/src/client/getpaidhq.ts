import { BaseClient, type ClientConfig } from './base-client'
import { SubscriptionsResource } from '@/resources/subscriptions'
import { UsageResource } from '@/resources/usage'
import { CustomersResource } from '@/resources/customers'
import { InvoicesResource } from '@/resources/invoices'
import { PaymentsResource } from '@/resources/payments'

export class GetPaidHQ extends BaseClient {
  public readonly subscriptions: SubscriptionsResource
  public readonly usage: UsageResource
  public readonly customers: CustomersResource
  public readonly invoices: InvoicesResource
  public readonly payments: PaymentsResource

  constructor(config: ClientConfig) {
    super(config)

    // Initialize all resource classes
    this.subscriptions = new SubscriptionsResource(this)
    this.usage = new UsageResource(this)
    this.customers = new CustomersResource(this)
    this.invoices = new InvoicesResource(this)
    this.payments = new PaymentsResource(this)
  }

  /**
   * Test the API connection
   */
  async ping(): Promise<{ status: string; timestamp: string }> {
    return this.get('/api/health')
  }
}