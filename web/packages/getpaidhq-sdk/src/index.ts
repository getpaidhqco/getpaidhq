// Main exports
export { GetPaidHQClient } from './client';
export type { GetPaidHQClientConfig } from './client';

// Export version (tracks the GetPaidHQ OpenAPI contract this SDK is generated against)
export const SPEC_VERSION = '1.0.0';

// Export individual resources for tree-shaking
export {
  ApiKeysResource,
  BillingResource,
  CartsResource,
  CouponsResource,
  CustomersResource,
  DiscountsResource,
  DunningResource,
  GatewaysResource,
  InvoicesResource,
  MetersResource,
  OrdersResource,
  OrganizationsResource,
  PaymentsResource,
  ProductsResource,
  VariantsResource,
  PricesResource,
  SessionsResource,
  SettingsResource,
  SubscriptionsResource,
  UsageResource,
  WebhooksResource,
} from './resources';

// Export all types (1:1 with the OpenAPI spec component schemas)
export * from './types';

// Export errors
export {
  GetPaidHQError,
  AuthenticationError,
  AuthorizationError,
  NotFoundError,
  ValidationError,
  RateLimitError,
  ServerError,
} from './errors/errors';
