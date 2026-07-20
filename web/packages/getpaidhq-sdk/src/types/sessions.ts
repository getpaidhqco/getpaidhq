import { Metadata } from './common';

/** Create session input (spec: CreateSessionRequest). */
export interface CreateSessionRequest {
  country: string;
  currency: string;
  metadata?: Metadata;
}

/** Create session response (spec: CreateSessionResponse). */
export interface CreateSessionResponse {
  cart_id: string;
  id: string;
}
