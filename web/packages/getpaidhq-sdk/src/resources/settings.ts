import { HttpClient } from '../utils/http-client';
import { buildQueryString } from '../utils/query';
import {
  CreateSettingRequest,
  UpdateSettingRequest,
  SettingResponse,
  ListResponse,
  PaginationParams,
} from '../types';

export class SettingsResource {
  private readonly resourcePath = '/api/settings';

  constructor(private httpClient: HttpClient) {}

  /** List settings (GET /api/settings). */
  async list(params?: PaginationParams): Promise<ListResponse<SettingResponse>> {
    return this.httpClient.get<ListResponse<SettingResponse>>(
      `${this.resourcePath}${buildQueryString(params)}`,
    );
  }

  /** Create a setting (POST /api/settings). */
  async create(data: CreateSettingRequest): Promise<SettingResponse> {
    return this.httpClient.post<SettingResponse>(this.resourcePath, data);
  }

  /** Get a setting (GET /api/settings/{parentId}/{id}). */
  async get(parentId: string, settingId: string): Promise<SettingResponse> {
    return this.httpClient.get<SettingResponse>(`${this.resourcePath}/${parentId}/${settingId}`);
  }

  /** Update a setting (PUT /api/settings/{parentId}/{id}). */
  async update(
    parentId: string,
    settingId: string,
    data: UpdateSettingRequest,
  ): Promise<SettingResponse> {
    return this.httpClient.put<SettingResponse>(
      `${this.resourcePath}/${parentId}/${settingId}`,
      data,
    );
  }

  /** Delete a setting (DELETE /api/settings/{parentId}/{id}). */
  async delete(parentId: string, settingId: string): Promise<void> {
    return this.httpClient.delete(`${this.resourcePath}/${parentId}/${settingId}`);
  }
}
