import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import { PaymentResponse, PaymentMethodResponse, ListResponse, PaginationParams } from '../types';

export class PaymentsResource {
  private readonly resourcePath = '/api/payments';

  constructor(private httpClient: HttpClient) {}

  /** List payments (GET /api/payments). */
  async list(params?: PaginationParams): Promise<ListResponse<PaymentResponse>> {
    return this.httpClient.get<ListResponse<PaymentResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Get a payment by id (GET /api/payments/{id}). */
  async get(paymentId: string): Promise<PaymentResponse> {
    return this.httpClient.get<PaymentResponse>(`${this.resourcePath}/${paymentId}`);
  }

  /** Get a payment method by id (GET /api/payment-methods/{id}). */
  async getPaymentMethod(paymentMethodId: string): Promise<PaymentMethodResponse> {
    return this.httpClient.get<PaymentMethodResponse>(`/api/payment-methods/${paymentMethodId}`);
  }
}
