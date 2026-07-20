import { AuthManager } from './auth/auth';
import { HttpClient, HttpClientConfig } from './utils/http-client';

// Export HttpClient for advanced use cases
export { HttpClient } from './utils/http-client';
import {
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
import { HealthResponse } from './types';

export interface GetPaidHQClientConfig {
  /**
   * API key for authentication (starts with 'sk_')
   */
  apiKey?: string;

  /**
   * Async function that returns bearer token for OAuth authentication
   */
  getToken?: () => Promise<string | null>;
  /**
   * Bearer token for OAuth authentication
   */
  bearerToken?: string;

  /**
   * Token for public payment endpoints (used as query parameter)
   */
  token?: string;

  /**
   * Base URL for the API
   * @default https://api.getpaidhq.co
   */
  baseURL?: string;

  /**
   * Request timeout in milliseconds
   * @default 30000 (30 seconds)
   */
  timeout?: number;

  /**
   * Number of retry attempts for failed requests
   * @default 3
   */
  retries?: number;

  /**
   * Delay between retries in milliseconds
   * @default 1000 (1 second)
   */
  retryDelay?: number;

  /**
   * Custom user agent string
   */
  userAgent?: string;
}

export class GetPaidHQClient {
  private authManager: AuthManager;
  public readonly httpClient: HttpClient;

  // Resources
  public readonly apiKeys: ApiKeysResource;
  public readonly billing: BillingResource;
  public readonly carts: CartsResource;
  public readonly coupons: CouponsResource;
  public readonly customers: CustomersResource;
  public readonly discounts: DiscountsResource;
  public readonly dunning: DunningResource;
  public readonly gateways: GatewaysResource;
  public readonly invoices: InvoicesResource;
  public readonly meters: MetersResource;
  public readonly orders: OrdersResource;
  public readonly organizations: OrganizationsResource;
  public readonly payments: PaymentsResource;
  public readonly products: ProductsResource;
  public readonly variants: VariantsResource;
  public readonly prices: PricesResource;
  public readonly sessions: SessionsResource;
  public readonly settings: SettingsResource;
  public readonly subscriptions: SubscriptionsResource;
  public readonly usage: UsageResource;
  public readonly webhooks: WebhooksResource;

  constructor(config: GetPaidHQClientConfig) {
    // Validate auth config - allow token as standalone auth for public endpoints only
    if (!config.apiKey && !config.getToken && !config.token) {
      throw new Error('Either apiKey, bearerToken, or token must be provided');
    }

    // Initialize auth manager
    this.authManager = new AuthManager({
      apiKey: config.apiKey,
      getToken: config.getToken,
      token: config.token,
    });

    // Initialize HTTP client
    const httpConfig: HttpClientConfig = {
      baseURL: config.baseURL || 'https://api.getpaidhq.co',
      timeout: config.timeout,
      retries: config.retries,
      retryDelay: config.retryDelay,
      userAgent: config.userAgent,
    };

    this.httpClient = new HttpClient(httpConfig, this.authManager);

    // Initialize resources
    this.apiKeys = new ApiKeysResource(this.httpClient);
    this.billing = new BillingResource(this.httpClient);
    this.carts = new CartsResource(this.httpClient);
    this.coupons = new CouponsResource(this.httpClient);
    this.customers = new CustomersResource(this.httpClient);
    this.discounts = new DiscountsResource(this.httpClient);
    this.dunning = new DunningResource(this.httpClient);
    this.gateways = new GatewaysResource(this.httpClient);
    this.invoices = new InvoicesResource(this.httpClient);
    this.meters = new MetersResource(this.httpClient);
    this.orders = new OrdersResource(this.httpClient);
    this.organizations = new OrganizationsResource(this.httpClient);
    this.payments = new PaymentsResource(this.httpClient);
    this.products = new ProductsResource(this.httpClient);
    this.variants = new VariantsResource(this.httpClient);
    this.prices = new PricesResource(this.httpClient);
    this.sessions = new SessionsResource(this.httpClient);
    this.settings = new SettingsResource(this.httpClient);
    this.subscriptions = new SubscriptionsResource(this.httpClient);
    this.usage = new UsageResource(this.httpClient);
    this.webhooks = new WebhooksResource(this.httpClient);
  }

  /**
   * Update API key for authentication
   */
  updateApiKey(apiKey: string): void {
    this.authManager.updateApiKey(apiKey);
  }

  /**
   * Update bearer token for authentication
   */
  updateBearerToken(_bearerToken: string): void {}

  /**
   * Check API health (GET /api/health).
   */
  async healthCheck(): Promise<HealthResponse> {
    return this.httpClient.get<HealthResponse>('/api/health');
  }
}
