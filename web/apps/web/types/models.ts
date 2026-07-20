type BillingIntervalType = string;

export type Subscription = {
  org_id: string
  id: string
  order_id: string
  order_item_id: string
  customer_id: string
  status: string
  payment_method_id: string
  start_date: string
  end_date: any
  billing_interval: BillingIntervalType
  billing_interval_qty: number
  cycles: number
  billing_anchor: number
  trial_ends_at: string
  cancel_at: any
  ends_at: any
  last_charge: string
  renews_at: string
  current_period_start: string
  current_period_end: string
  retries: number
  next_retry: any
  currency: string
  amount: number
  metadata: any
  cycles_processed: number
  total_revenue: number
  cancelled_at: any
  created_at: string
  updated_at: string
}
