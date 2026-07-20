import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import { InvoiceResponse, ListResponse, PaginationParams } from '../types';

export class InvoicesResource {
  private readonly resourcePath = '/api/invoices';

  constructor(private httpClient: HttpClient) {}

  /** List invoices (GET /api/invoices). */
  async list(params?: PaginationParams): Promise<ListResponse<InvoiceResponse>> {
    return this.httpClient.get<ListResponse<InvoiceResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Get an invoice by id (GET /api/invoices/{id}). */
  async get(invoiceId: string): Promise<InvoiceResponse> {
    return this.httpClient.get<InvoiceResponse>(`${this.resourcePath}/${invoiceId}`);
  }
}
