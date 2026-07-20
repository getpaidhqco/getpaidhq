import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  SubscriptionResponse,
  UpdateSubscriptionRequest,
  UpdateBillingAnchorRequest,
  ProrationDetailsResponse,
  PauseSubscriptionRequest,
  CancelSubscriptionRequest,
  ResumeSubscriptionRequest,
  InvoiceResponse,
  PaymentResponse,
  SubscriptionUsageResponse,
  ListResponse,
  PaginationParams,
} from '../types';

export class SubscriptionsResource {
  private readonly resourcePath = '/api/subscriptions';

  constructor(private httpClient: HttpClient) {}

  /** List subscriptions (GET /api/subscriptions). */
  async list(params?: PaginationParams): Promise<ListResponse<SubscriptionResponse>> {
    return this.httpClient.get<ListResponse<SubscriptionResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Get a subscription by id (GET /api/subscriptions/{id}). */
  async get(subscriptionId: string): Promise<SubscriptionResponse> {
    return this.httpClient.get<SubscriptionResponse>(`${this.resourcePath}/${subscriptionId}`);
  }

  /** Update a subscription (PATCH /api/subscriptions/{id}). */
  async update(
    subscriptionId: string,
    data: UpdateSubscriptionRequest,
  ): Promise<SubscriptionResponse> {
    return this.httpClient.patch<SubscriptionResponse>(
      `${this.resourcePath}/${subscriptionId}`,
      data,
    );
  }

  /** Update a subscription's billing anchor (PATCH /api/subscriptions/{id}/billing-anchor). */
  async updateBillingAnchor(
    subscriptionId: string,
    data: UpdateBillingAnchorRequest,
  ): Promise<ProrationDetailsResponse> {
    return this.httpClient.patch<ProrationDetailsResponse>(
      `${this.resourcePath}/${subscriptionId}/billing-anchor`,
      data,
    );
  }

  /** Cancel a subscription (PUT /api/subscriptions/{id}/cancel). */
  async cancel(
    subscriptionId: string,
    data: CancelSubscriptionRequest,
  ): Promise<SubscriptionResponse> {
    return this.httpClient.put<SubscriptionResponse>(
      `${this.resourcePath}/${subscriptionId}/cancel`,
      data,
    );
  }

  /** Pause a subscription (PUT /api/subscriptions/{id}/pause). */
  async pause(
    subscriptionId: string,
    data: PauseSubscriptionRequest,
  ): Promise<SubscriptionResponse> {
    return this.httpClient.put<SubscriptionResponse>(
      `${this.resourcePath}/${subscriptionId}/pause`,
      data,
    );
  }

  /** Resume a subscription (PUT /api/subscriptions/{id}/resume). */
  async resume(
    subscriptionId: string,
    data: ResumeSubscriptionRequest,
  ): Promise<SubscriptionResponse> {
    return this.httpClient.put<SubscriptionResponse>(
      `${this.resourcePath}/${subscriptionId}/resume`,
      data,
    );
  }

  /** List a subscription's invoices (GET /api/subscriptions/{id}/invoices). */
  async listInvoices(
    subscriptionId: string,
    params?: PaginationParams,
  ): Promise<ListResponse<InvoiceResponse>> {
    return this.httpClient.get<ListResponse<InvoiceResponse>>(
      `${this.resourcePath}/${subscriptionId}/invoices${buildQueryString(params)}`,
    );
  }

  /** List a subscription's payments (GET /api/subscriptions/{id}/payments). */
  async listPayments(
    subscriptionId: string,
    params?: PaginationParams,
  ): Promise<ListResponse<PaymentResponse>> {
    return this.httpClient.get<ListResponse<PaymentResponse>>(
      `${this.resourcePath}/${subscriptionId}/payments${buildQueryString(params)}`,
    );
  }

  /** Get a subscription's usage summary (GET /api/subscriptions/{id}/usage). */
  async getUsage(subscriptionId: string): Promise<SubscriptionUsageResponse> {
    return this.httpClient.get<SubscriptionUsageResponse>(
      `${this.resourcePath}/${subscriptionId}/usage`,
    );
  }
}
