import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  CreateCustomerInput,
  CustomerResponse,
  CustomerDunningHistoryResponse,
  CreatePaymentMethodInput,
  UpdatePaymentMethodInput,
  PaymentMethodResponse,
  ListResponse,
  PaginationParams,
} from '../types';

export class CustomersResource {
  private readonly resourcePath = '/api/customers';

  constructor(private httpClient: HttpClient) {}

  /** List customers (GET /api/customers). */
  async list(params?: PaginationParams): Promise<ListResponse<CustomerResponse>> {
    return this.httpClient.get<ListResponse<CustomerResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create a customer (POST /api/customers). */
  async create(data: CreateCustomerInput): Promise<CustomerResponse> {
    return this.httpClient.post<CustomerResponse>(this.resourcePath, data);
  }

  /** Get a customer by id (GET /api/customers/{id}). */
  async get(customerId: string): Promise<CustomerResponse> {
    return this.httpClient.get<CustomerResponse>(`${this.resourcePath}/${customerId}`);
  }

  /** Get a customer's dunning history (GET /api/customers/{id}/dunning-history). */
  async getDunningHistory(customerId: string): Promise<CustomerDunningHistoryResponse> {
    return this.httpClient.get<CustomerDunningHistoryResponse>(
      `${this.resourcePath}/${customerId}/dunning-history`,
    );
  }

  /** Add a payment method to a customer (POST /api/customers/{id}/payment-methods). */
  async createPaymentMethod(
    customerId: string,
    data: CreatePaymentMethodInput,
  ): Promise<PaymentMethodResponse> {
    return this.httpClient.post<PaymentMethodResponse>(
      `${this.resourcePath}/${customerId}/payment-methods`,
      data,
    );
  }

  /** Update a customer's payment method (PUT /api/customers/{id}/payment-methods/{pmid}). */
  async updatePaymentMethod(
    customerId: string,
    paymentMethodId: string,
    data: UpdatePaymentMethodInput,
  ): Promise<PaymentMethodResponse> {
    return this.httpClient.put<PaymentMethodResponse>(
      `${this.resourcePath}/${customerId}/payment-methods/${paymentMethodId}`,
      data,
    );
  }
}
