import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import { CreateMeterRequest, MeterResponse, ListResponse, PaginationParams } from '../types';

export class MetersResource {
  private readonly resourcePath = '/api/meters';

  constructor(private httpClient: HttpClient) {}

  /** List meters (GET /api/meters). */
  async list(params?: PaginationParams): Promise<ListResponse<MeterResponse>> {
    return this.httpClient.get<ListResponse<MeterResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create a meter (POST /api/meters). */
  async create(data: CreateMeterRequest): Promise<MeterResponse> {
    return this.httpClient.post<MeterResponse>(this.resourcePath, data);
  }

  /** Get a meter by id (GET /api/meters/{id}). */
  async get(meterId: string): Promise<MeterResponse> {
    return this.httpClient.get<MeterResponse>(`${this.resourcePath}/${meterId}`);
  }
}
