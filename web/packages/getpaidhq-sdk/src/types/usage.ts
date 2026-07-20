import { Metadata } from './common';

/** A single usage event in an ingest batch (spec: IngestEventsRequest.events items). */
export interface UsageEventInput {
  customer_id?: string;
  external_customer_id?: string;
  external_id?: string;
  metadata?: Metadata;
  metric_code: string;
  subscription_id?: string;
  timestamp?: string;
}

/** Ingest usage events input (spec: IngestEventsRequest). */
export interface IngestEventsRequest {
  events: UsageEventInput[];
}

/** Per-event ingest result (spec: IngestEventsResponse.results items). */
export interface IngestEventResult {
  error?: string;
  id?: string;
  index?: number;
  status?: string;
}

/** Ingest usage events response (spec: IngestEventsResponse). */
export interface IngestEventsResponse {
  results?: IngestEventResult[];
}
