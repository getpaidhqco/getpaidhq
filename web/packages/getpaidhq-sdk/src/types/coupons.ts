import { Metadata } from './common';

/** Coupon (spec: CouponResponse). */
export interface CouponResponse {
  active: boolean;
  discount_type: string;
  duration: string;
  id: string;
  name: string;
}

/** Create coupon input (spec: CreateCouponInput). */
export interface CreateCouponInput {
  amount_off?: number;
  applies_to_products?: string[];
  currency?: string;
  discount_type: string;
  duration: string;
  duration_in_cycles?: number;
  max_redemptions?: number;
  metadata?: Metadata;
  name: string;
  once_per_customer?: boolean;
  percent_off?: unknown;
  redeem_by?: string;
}

/** Update coupon input (spec: UpdateCouponInput). */
export interface UpdateCouponInput {
  active?: boolean;
  metadata?: Metadata;
  name: string;
}

/** Coupon code restrictions (spec: CreateCouponCodeInput.restrictions). */
export interface CreateCouponCodeRestrictions {
  first_time_transaction?: boolean;
  minimum_amount?: number;
  minimum_amount_currency?: string;
}

/** Coupon code (spec: CouponCodeResponse). */
export interface CouponCodeResponse {
  active: boolean;
  code: string;
  coupon_id: string;
  id: string;
}

/** Create coupon code input (spec: CreateCouponCodeInput). */
export interface CreateCouponCodeInput {
  code: string;
  customer_id?: string;
  expires_at?: string;
  max_redemptions?: number;
  metadata?: Metadata;
  restrictions?: CreateCouponCodeRestrictions;
}

/** Update coupon code input (spec: UpdateCouponCodeInput). */
export interface UpdateCouponCodeInput {
  active?: boolean;
  metadata?: Metadata;
}

/** Discount (spec: DiscountResponse). */
export interface DiscountResponse {
  coupon_id: string;
  customer_id: string;
  id: string;
  order_id: string | null;
  start_cycle: number;
  status: string;
  subscription_id: string | null;
}
