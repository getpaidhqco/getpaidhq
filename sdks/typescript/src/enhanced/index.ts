/**
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
