// Main SDK export
export { GetPaidHQ } from '@/client/getpaidhq'
export { GetPaidHQError } from '@/client/base-client'

// Export types for TypeScript users
export type * from '@/types'
export type { ClientConfig, RequestOptions } from '@/client/base-client'

// Export individual resources for advanced usage
export { SubscriptionsResource } from '@/resources/subscriptions'
export { UsageResource } from '@/resources/usage'
export { CustomersResource } from '@/resources/customers'
export { InvoicesResource } from '@/resources/invoices'
export { PaymentsResource } from '@/resources/payments'

// Convenience export for default usage
export default GetPaidHQ