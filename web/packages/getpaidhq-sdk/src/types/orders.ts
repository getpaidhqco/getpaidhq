import { BillingAddress, Metadata } from './common';
import { PriceResponse } from './products';

/** Customer sub-object embedded in {@link OrderResponse}. */
export interface OrderCustomer {
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

/** Order line item (spec: OrderResponse.items items). */
export interface OrderItem {
  created_at: string;
  description: string;
  discount_total: number;
  id: string;
  metadata: Metadata;
  order_id: string;
  price: PriceResponse;
  price_id: string;
  product_id: string;
  quantity: number;
  sub_total: number;
  tax_total: number;
  total: number;
  updated_at: string;
  variant_id: string;
}

/** Order (spec: OrderResponse). */
export interface OrderResponse {
  cart_id: string;
  created_at: string;
  currency: string;
  customer: OrderCustomer;
  customer_id: string;
  id: string;
  items: OrderItem[];
  metadata: Metadata;
  reference: string;
  session_id: string;
  status: string;
  total: number;
}

/** Cart item supplied when creating an order (spec: CreateOrderRequest.cart.items items). */
export interface CreateOrderCartItem {
  price_id: string;
  product_id: string;
  quantity: number;
}

/** Customer supplied when creating an order (spec: CreateOrderRequest.customer). */
export interface CreateOrderCustomer {
  email?: string;
  first_name?: string;
  id?: string;
  last_name?: string;
  metadata?: Metadata;
  phone?: string;
}

/** Create order input (spec: CreateOrderRequest). */
export interface CreateOrderRequest {
  cart?: {
    currency?: string;
    items?: CreateOrderCartItem[];
  };
  customer: CreateOrderCustomer;
  metadata?: Metadata;
  options?: Record<string, string>;
  payment_method_id?: string;
  psp_id: string;
  session_id?: string;
}

/** Create order response (spec: CreateOrderResponse). */
export interface CreateOrderResponse {
  order: OrderResponse;
  psp: any;
}

/** Payment block supplied when completing an order (spec: CompleteOrderRequest.payment). */
export interface CompleteOrderPayment {
  amount?: number;
  completed_at?: string;
  currency?: string;
  metadata?: Metadata;
  psp_id?: string;
  reference?: string;
}

/** Payment method block supplied when completing an order (spec: CompleteOrderRequest.payment_method). */
export interface CompleteOrderPaymentMethod {
  billing_address?: BillingAddress;
  details?: any;
  is_default?: boolean;
  metadata?: Metadata;
  name?: string;
  psp?: string;
  token?: string;
  type?: string;
}

/** Complete order input (spec: CompleteOrderRequest). */
export interface CompleteOrderRequest {
  metadata?: Metadata;
  payment?: CompleteOrderPayment;
  payment_method?: CompleteOrderPaymentMethod;
  payment_method_id?: string;
}
