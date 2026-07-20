import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import { CreateWebhookSubscriptionRequest, ListResponse, PaginationParams } from '../types';

export class WebhooksResource {
  private readonly resourcePath = '/api/webhooks';

  constructor(private httpClient: HttpClient) {}

  /** List webhook subscriptions (GET /api/webhooks). */
  async list(params?: PaginationParams): Promise<ListResponse> {
    return this.httpClient.get<ListResponse>(`${this.resourcePath}${buildQueryString(params)}`);
  }

  /** Create a webhook subscription (POST /api/webhooks). */
  async create(data: CreateWebhookSubscriptionRequest): Promise<unknown> {
    return this.httpClient.post<unknown>(this.resourcePath, data);
  }

  /** Deliver/process an incoming webhook notification (POST /api/notify). */
  async notify(p?: string): Promise<unknown> {
    const query = p ? `?p=${encodeURIComponent(p)}` : '';
    return this.httpClient.post<unknown>(`/api/notify${query}`, {});
  }
}
