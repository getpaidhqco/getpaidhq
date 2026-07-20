import { Metadata } from './common';

/** Pricing tier (spec: tiers items). */
export interface PriceTier {
  flat_amount: number;
  from_value: string;
  per_unit_amount: string;
  to_value: string;
}

/** Price (spec: PriceResponse). */
export interface PriceResponse {
  billable_metric_id: string;
  billing_interval: string;
  billing_interval_qty: number;
  category: string;
  created_at: string;
  credit_on_decrease: boolean;
  currency: string;
  cycles: number;
  filter_field: string;
  filter_value: string;
  id: string;
  label: string;
  metadata: Metadata;
  min_price: number;
  prorate_on_increase: boolean;
  scheme: string;
  suggested_price: number;
  tax_code: string;
  tiers: PriceTier[];
  trial_interval: string;
  trial_interval_qty: number;
  /** Units unit_price buys (fixed/package schemes); 0/1 = per single unit. */
  unit_count: number;
  unit_price: number;
  updated_at: string;
  variant_id: string;
}

/** Create price input (spec: CreatePriceRequest). */
export interface CreatePriceRequest {
  billable_metric_id?: string;
  billing_interval?: string;
  billing_interval_qty?: number;
  category: string;
  credit_on_decrease?: boolean;
  currency: string;
  cycles?: number;
  filter_field?: string;
  filter_value?: string;
  label?: string;
  metadata?: Metadata;
  min_price?: number;
  prorate_on_increase?: boolean;
  scheme: string;
  suggested_price?: number;
  tax_code?: string;
  tiers?: PriceTier[];
  trial_interval?: string;
  trial_interval_qty?: number;
  /** Units unit_price buys (fixed/package schemes); omit or 1 = per single unit. */
  unit_count?: number;
  unit_price?: number;
  variant_id: string;
}

/** Variant (spec: VariantResponse). */
export interface VariantResponse {
  created_at: string;
  id: string;
  name: string;
  prices: PriceResponse[];
  updated_at: string;
}

/** Create variant input (spec: CreateVariantRequest). */
export interface CreateVariantRequest {
  description?: string;
  metadata?: Metadata;
  name: string;
}

/** Update variant input (spec: UpdateVariantRequest). */
export interface UpdateVariantRequest {
  description?: string;
  metadata?: Metadata;
  name: string;
}

/** Product (spec: ProductResponse). */
export interface ProductResponse {
  archived_at: string;
  created_at: string;
  description: string;
  id: string;
  metadata: Metadata;
  name: string;
  status: string;
  updated_at: string;
  variants: VariantResponse[];
}

/** Nested price input when creating a product+variants in one call. */
export interface CreateProductPriceInput {
  billable_metric_id?: string;
  billing_interval?: string;
  billing_interval_qty?: number;
  category: string;
  credit_on_decrease?: boolean;
  currency: string;
  cycles?: number;
  filter_field?: string;
  filter_value?: string;
  label?: string;
  metadata?: Metadata;
  min_price?: number;
  prorate_on_increase?: boolean;
  scheme: string;
  suggested_price?: number;
  tax_code?: string;
  tiers?: PriceTier[];
  trial_interval?: string;
  trial_interval_qty?: number;
  /** Units unit_price buys (fixed/package schemes); omit or 1 = per single unit. */
  unit_count?: number;
  unit_price?: number;
}

/** Nested variant input when creating a product (spec: CreateProductRequest.variants items). */
export interface CreateProductVariantInput {
  description?: string;
  metadata?: Metadata;
  name: string;
  prices: CreateProductPriceInput[];
}

/** Create product input (spec: CreateProductRequest). */
export interface CreateProductRequest {
  description?: string;
  metadata?: Metadata;
  name: string;
  variants: CreateProductVariantInput[];
}

/** Update product input (spec: UpdateProductRequest). */
export interface UpdateProductRequest {
  description?: string;
  metadata?: Metadata;
  name: string;
}
