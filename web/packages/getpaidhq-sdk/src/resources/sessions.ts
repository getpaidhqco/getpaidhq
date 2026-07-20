import { HttpClient } from '../utils/http-client';
import { CreateSessionRequest, CreateSessionResponse } from '../types';

export class SessionsResource {
  constructor(private httpClient: HttpClient) {}

  /** Create a checkout session (POST /api/sessions). */
  async create(data: CreateSessionRequest): Promise<CreateSessionResponse> {
    return this.httpClient.post<CreateSessionResponse>('/api/sessions', data);
  }
}
