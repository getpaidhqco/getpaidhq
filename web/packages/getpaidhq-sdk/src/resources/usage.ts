import { HttpClient } from '../utils/http-client';
import { IngestEventsRequest, IngestEventsResponse } from '../types';

export class UsageResource {
  constructor(private httpClient: HttpClient) {}

  /** Ingest a batch of usage events (POST /api/usage/ingest). */
  async ingest(data: IngestEventsRequest): Promise<IngestEventsResponse> {
    return this.httpClient.post<IngestEventsResponse>('/api/usage/ingest', data);
  }
}
