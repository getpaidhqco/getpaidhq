import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreatePriceRequest } from '@getpaidhq/sdk';

/**
 * Price form validation — mirrors the server contract (CreatePriceRequest).
 * Enum values from gphq-server domain (price_types.go): PriceCategory, PriceScheme,
 * BillingInterval. Monetary fields are integers (minor units / cents). See meters.ts.
 */

// Server PriceCategory (price_types.go): one_time | subscription | free | variable.
// There is NO "metered"/"usage" category — metering is orthogonal to cadence; a price
// is metered by setting billable_metric_id (a metered price is a subscription + meter).
const priceCategory = z.enum(['one_time', 'subscription', 'free', 'variable']);
const priceScheme = z.enum(['fixed', 'tiered', 'volume', 'graduated', 'package']);
const billingInterval = z.enum(['none', 'hour', 'day', 'week', 'month', 'year']);

export const PRICE_CATEGORIES = priceCategory.options;
export const PRICE_SCHEMES = priceScheme.options;
export const BILLING_INTERVALS = billingInterval.options;

/**
 * Schemes a UI should offer: fixed (units × unit_price), the two tiered models,
 * and package (per started block of unit_count units — metered only).
 * `tiered` is an alias for `graduated` server-side, so it's omitted from the picker.
 */
export const SELECTABLE_PRICE_SCHEMES = ['fixed', 'package', 'graduated', 'volume'] as const;

/** Schemes that price by rate bands and therefore require at least one tier. */
export const TIERED_SCHEMES = ['graduated', 'volume', 'tiered'] as const;

/** Whether a scheme needs tiers (graduated/volume/tiered) rather than a unit price. */
export const schemeRequiresTiers = (scheme: string): boolean =>
  (TIERED_SCHEMES as readonly string[]).includes(scheme);

/**
 * Schemes that price units in blocks of `unit_count` — the per-unit rate is
 * unit_price / unit_count, letting an integer-cent price express sub-cent rates
 * ("$1 per 1,000 calls" = unit_price 100, unit_count 1000). The two differ only
 * in how a partial block is billed: `fixed` prorates it, `package` rounds up and
 * bills the full block. unit_count is rejected by the server on tiered schemes.
 */
export const UNIT_COUNT_SCHEMES = ['fixed', 'package'] as const;

/** Whether a scheme supports unit_count ("price per N units"): fixed and package. */
export const schemeSupportsUnitCount = (scheme: string): boolean =>
  (UNIT_COUNT_SCHEMES as readonly string[]).includes(scheme);

/**
 * One rate band, as entered in the form. Decimal/currency fields are kept as the
 * raw strings the user types (dollars); the form transforms them to the server's
 * cents contract (per_unit_amount as a cents string, flat_amount as cents) on submit.
 * from_value/to_value are unit quantities; an empty to_value is the unbounded last tier.
 */
const priceTierInput = z.object({
  from_value: z.string().optional(),
  to_value: z.string().optional(),
  per_unit_amount: z.string().optional(),
  flat_amount: z.string().optional(),
});

export type PriceTierInput = z.infer<typeof priceTierInput>;

const createPriceObject = z.object({
  variant_id: z.string().min(1, 'Variant is required'),
  category: priceCategory,
  scheme: priceScheme,
  currency: z.string().min(1, 'Currency is required').max(3),
  label: z.string().optional(),
  unit_price: z.number().int().optional(),
  // How many units unit_price buys (fixed/package only); 1 = per single unit.
  unit_count: z.number().int().min(1).optional(),
  min_price: z.number().int().optional(),
  suggested_price: z.number().int().optional(),
  billing_interval: billingInterval.optional(),
  billing_interval_qty: z.number().int().optional(),
  cycles: z.number().int().optional(),
  trial_interval: billingInterval.optional(),
  trial_interval_qty: z.number().int().optional(),
  billable_metric_id: z.string().optional(),
  filter_field: z.string().max(255).optional(),
  filter_value: z.string().max(255).optional(),
  tax_code: z.string().optional(),
  tiers: z.array(priceTierInput).optional(),
  // Proration controls for subscription price changes (mirror CreatePriceRequest).
  prorate_on_increase: z.boolean().optional(),
  credit_on_decrease: z.boolean().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const createPriceSchema = createPriceObject.superRefine((val, ctx) => {
  // Graduated/volume price by rate bands — they need at least one tier; fixed ignores them.
  if (schemeRequiresTiers(val.scheme) && (!val.tiers || val.tiers.length === 0)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['tiers'],
      message: 'Add at least one tier for graduated or volume pricing',
    });
  }
  // Server rules (validatePriceConfig): unit_count belongs to fixed/package only.
  if (!schemeSupportsUnitCount(val.scheme) && (val.unit_count ?? 1) > 1) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['unit_count'],
      message: 'Unit count applies only to fixed and package pricing',
    });
  }
  // Package bills started blocks of metered usage — it requires a meter and is flat.
  if (val.scheme === 'package') {
    if (!val.billable_metric_id) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['billable_metric_id'],
        message: 'Package pricing requires a meter — it bills started blocks of usage',
      });
    }
    if (val.tiers && val.tiers.length > 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['tiers'],
        message: 'Package pricing is flat — tiers are not allowed',
      });
    }
  }
});

// Inferred from the bare object (not the refined wrapper) to keep TS comparisons shallow.
export type CreatePriceFormValues = z.infer<typeof createPriceObject>;

export const priceResolvers = {
  create: zodResolver(createPriceSchema),
};

export const priceSchemas = {
  create: createPriceSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreatePriceKeys: Exact<keyof CreatePriceFormValues, keyof CreatePriceRequest> = true;
