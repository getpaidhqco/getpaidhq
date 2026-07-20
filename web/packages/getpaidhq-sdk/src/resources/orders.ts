import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  CreateOrderRequest,
  CreateOrderResponse,
  CompleteOrderRequest,
  OrderResponse,
  SubscriptionResponse,
  ListResponse,
  PaginationParams,
} from '../types';

export class OrdersResource {
  private readonly resourcePath = '/api/orders';

  constructor(private httpClient: HttpClient) {}

  /** List orders (GET /api/orders). */
  async list(params?: PaginationParams): Promise<ListResponse<OrderResponse>> {
    return this.httpClient.get<ListResponse<OrderResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create an order (POST /api/orders). */
  async create(data: CreateOrderRequest): Promise<CreateOrderResponse> {
    return this.httpClient.post<CreateOrderResponse>(this.resourcePath, data);
  }

  /** Get an order by id (GET /api/orders/{id}). */
  async get(orderId: string): Promise<OrderResponse> {
    return this.httpClient.get<OrderResponse>(`${this.resourcePath}/${orderId}`);
  }

  /** Complete an order (POST /api/orders/{id}/complete). */
  async complete(orderId: string, data: CompleteOrderRequest): Promise<OrderResponse> {
    return this.httpClient.post<OrderResponse>(`${this.resourcePath}/${orderId}/complete`, data);
  }

  /** List subscriptions created from an order (GET /api/orders/{id}/subscriptions). */
  async listSubscriptions(orderId: string): Promise<SubscriptionResponse[]> {
    return this.httpClient.get<SubscriptionResponse[]>(`${this.resourcePath}/${orderId}/subscriptions`);
  }
}
