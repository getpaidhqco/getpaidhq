import { BillingAddress, Metadata } from './common';

/** Customer sub-object embedded in {@link SubscriptionResponse}. */
export interface SubscriptionCustomer {
  billing_address: BillingAddress;
  created_at: string;
  email: string;
  first_name: string;
  id: string;
  last_name: string;
  metadata: Metadata;
  name: string;
  phone: string;
  updated_at: string;
}

/** Subscription (spec: SubscriptionResponse). */
export interface SubscriptionResponse {
  billing_anchor: number;
  billing_interval: string;
  billing_interval_qty: number;
  cancel_at: string;
  cancelled_at: string;
  created_at: string;
  currency: string;
  current_period_end: string;
  current_period_start: string;
  customer: SubscriptionCustomer;
  cycles: number;
  cycles_processed: number;
  end_date: string;
  ends_at: string;
  id: string;
  last_charge: string;
  metadata: Metadata;
  next_retry: string;
  order_id: string;
  payment_method_id: string;
  renews_at: string;
  retries: number;
  start_date: string;
  status: string;
  total_revenue: number;
  trial_ends_at: string;
  updated_at: string;
}

/**
 * @deprecated `GET /api/orders/{id}/subscriptions` now returns proper
 * {@link SubscriptionResponse} objects — use that type instead.
 */
export type Subscription = SubscriptionResponse;

/** Update subscription input (spec: UpdateSubscriptionRequest). */
export interface UpdateSubscriptionRequest {
  default_payment_method?: string;
  id?: string;
  metadata?: Metadata;
  status?: string;
}

/** Update billing anchor input (spec: UpdateBillingAnchorRequest). */
export interface UpdateBillingAnchorRequest {
  billing_anchor: number;
  proration_mode: string;
}

/** Proration details (spec: ProrationDetailsResponse). */
export interface ProrationDetailsResponse {
  credit_amount: number;
  current_period_end: string;
  current_period_start: string;
  days_credited: number;
  new_billing_anchor: number;
  new_period_end: string;
  new_period_start: string;
  old_billing_anchor: number;
}

/** Per-meter usage entry (spec: SubscriptionUsageResponse.meters items). */
export interface SubscriptionUsageMeter {
  aggregation?: string;
  metric_code?: string;
  quantity?: string;
}

/** Subscription usage summary (spec: SubscriptionUsageResponse). */
export interface SubscriptionUsageResponse {
  current_period_end: string;
  current_period_start: string;
  meters: SubscriptionUsageMeter[];
  subscription_id: string;
}

/** Pause subscription input (spec: PauseSubscriptionRequest). */
export interface PauseSubscriptionRequest {
  reason?: string;
}

/** Cancel subscription input (spec: CancelSubscriptionRequest). */
export interface CancelSubscriptionRequest {
  reason?: string;
  /**
   * What to do with the subscription's outstanding (open) invoice on cancel.
   * Omitted defaults to "uncollectible".
   * - "uncollectible": write off the open invoice (default)
   * - "void": void the open invoice
   * - "keep": leave the invoice as-is
   */
  outstanding_invoice?: "uncollectible" | "void" | "keep";
}

/** Resume subscription input (spec: ResumeSubscriptionRequest). */
export interface ResumeSubscriptionRequest {
  resume_behavior?: string;
}
