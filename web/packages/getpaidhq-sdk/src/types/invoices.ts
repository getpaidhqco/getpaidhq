import { Metadata } from './common';

/** Invoice line item (spec: InvoiceResponse.line_items items). */
export interface InvoiceLineItem {
  description: string;
  id: string;
  kind: string;
  metadata: Metadata;
  price_id: string;
  quantity: string;
  total: number;
  unit_amount: string;
}

/** Invoice (spec: InvoiceResponse). */
export interface InvoiceResponse {
  created_at: string;
  currency: string;
  customer_id: string;
  cycle: number;
  id: string;
  line_items: InvoiceLineItem[];
  metadata: Metadata;
  order_id: string;
  period_end: string;
  period_start: string;
  /**
   * Invoice lifecycle status. One of:
   * "draft" | "open" | "paid" | "uncollectible" | "void".
   * (The invoice-level "unpaid" status was removed in favour of "uncollectible".)
   */
  status: string;
  subscription_id: string;
  subtotal: number;
  total: number;
  updated_at: string;
}
