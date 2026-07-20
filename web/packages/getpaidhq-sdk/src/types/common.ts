// Shared primitives and envelopes, matching the OpenAPI spec exactly.

/** Free-form string metadata map (spec: Record<string, string>). */
export type Metadata = Record<string, string>;

/** Billing address sub-object shared across customer/payment-method responses. */
export interface BillingAddress {
  city: string;
  country: string;
  email: string;
  first_name: string;
  last_name: string;
  line1: string;
  line2: string;
  phone: string;
  postal_code: string;
  state: string;
}

/** Pagination metadata returned in {@link ListResponse} (spec: ListResponse.meta). */
export interface ListMeta {
  limit: number;
  page: number;
  total: number;
}

/** Standard list envelope (spec: ListResponse). */
export interface ListResponse<T = unknown> {
  data: T[];
  meta: ListMeta;
}

/** Dunning list envelope (spec: dunningList). */
export interface DunningListResponse<T = unknown> {
  data: T[];
  total: number;
}

/** Empty response body (spec: EmptyResponse). */
export type EmptyResponse = Record<string, never>;

/** Health check response (spec: HealthResponse). */
export interface HealthResponse {
  status: string;
}

/** Error detail entry (spec: HTTPError.errors items). */
export interface HTTPErrorDetail {
  more: Record<string, unknown>;
  name: string;
  reason: string;
}

/** RFC 7807 style error (spec: HTTPError). */
export interface HTTPError {
  detail: string;
  errors: HTTPErrorDetail[];
  instance: string;
  status: number;
  title: string;
  type: string;
}

/**
 * Optional pagination query parameters accepted by list endpoints.
 * The spec returns `meta.page` / `meta.limit` / `meta.total` on list envelopes.
 */
export interface PaginationParams {
  page?: number;
  limit?: number;
  sort_by?: string;
  sort_order?: string;
}
