import { BillingAddress, Metadata } from './common';

/** Payment (spec: PaymentResponse). */
export interface PaymentResponse {
  amount: number;
  created_at: string;
  currency: string;
  id: string;
  invoice_id: string;
  metadata: Metadata;
  net_amount: number;
  order_id: string;
  platform_fee: number;
  psp_fee: number;
  psp_id: string;
  reference: string;
  status: string;
  subscription_id: string;
  updated_at: string;
}

/** Payment method (spec: PaymentMethodResponse). */
export interface PaymentMethodResponse {
  billing_address: BillingAddress;
  created_at: string;
  customer_id: string;
  details: any;
  expire_at: string;
  id: string;
  metadata: Metadata;
  name: string;
  psp: string;
  status: string;
  type: string;
  updated_at: string;
}

/**
 * Create payment method input (spec: CreatePaymentMethodInput).
 * The spec exposes no fixed schema for this body, so it is intentionally open.
 */
export type CreatePaymentMethodInput = Record<string, any>;

/**
 * Update payment method input (spec: UpdatePaymentMethodInput).
 * The spec exposes no fixed schema for this body, so it is intentionally open.
 */
export type UpdatePaymentMethodInput = Record<string, any>;
