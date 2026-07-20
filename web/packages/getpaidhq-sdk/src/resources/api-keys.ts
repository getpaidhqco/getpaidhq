import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import { ApiKeyCreateResponse, CreateApiKeyInput, ListResponse, PaginationParams } from '../types';

export class ApiKeysResource {
  private readonly resourcePath = '/api/api-keys';

  constructor(private httpClient: HttpClient) {}

  /** List API keys (GET /api/api-keys). */
  async list(params?: PaginationParams): Promise<ListResponse> {
    return this.httpClient.get<ListResponse>(`${this.resourcePath}${buildQueryString(params)}`);
  }

  /** Create an API key (POST /api/api-keys). The full secret is returned only once. */
  async create(data: CreateApiKeyInput): Promise<ApiKeyCreateResponse> {
    return this.httpClient.post<ApiKeyCreateResponse>(this.resourcePath, data);
  }

  /** Delete an API key (DELETE /api/api-keys/{id}). */
  async delete(id: string): Promise<void> {
    return this.httpClient.delete(`${this.resourcePath}/${id}`);
  }
}
