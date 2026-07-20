import { HttpClient } from '../utils/http-client';
import { CreateOrgRequest, OrgResponse } from '../types';

export class OrganizationsResource {
  constructor(private httpClient: HttpClient) {}

  /** Create an organization (POST /api/organizations). */
  async create(data: CreateOrgRequest): Promise<OrgResponse> {
    return this.httpClient.post<OrgResponse>('/api/organizations', data);
  }
}
