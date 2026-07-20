import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateCouponInput, UpdateCouponInput } from '@getpaidhq/sdk';

/**
 * Coupon form validation. Mirrors the server contract (gphq-server
 * CreateCouponInput / UpdateCouponInput). Validation lives in react-sdk and is
 * guarded against the SDK types so it cannot silently drift — see products.ts /
 * meters.ts for the rationale.
 *
 * Server rules (internal/core/port/coupon.go):
 *   discount_type ∈ {percentage, fixed}   (required)
 *   duration      ∈ {once, repeating, forever} (required)
 *   percentage  → percent_off required
 *   fixed       → amount_off + currency required
 *   repeating   → duration_in_cycles required
 */

export const DISCOUNT_TYPES = ['percentage', 'fixed'] as const;
export const COUPON_DURATIONS = ['once', 'repeating', 'forever'] as const;

export type DiscountType = (typeof DISCOUNT_TYPES)[number];
export type CouponDuration = (typeof COUPON_DURATIONS)[number];

const createCouponObject = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  discount_type: z.enum(DISCOUNT_TYPES),
  // percentage discounts
  percent_off: z.number().positive('Must be greater than 0').max(100, 'Cannot exceed 100%').optional(),
  // fixed discounts (amount in minor units / cents)
  amount_off: z.number().int().positive('Must be greater than 0').optional(),
  currency: z.string().optional(),
  duration: z.enum(COUPON_DURATIONS),
  duration_in_cycles: z.number().int().positive('Must be greater than 0').optional(),
  max_redemptions: z.number().int().positive('Must be greater than 0').optional(),
  once_per_customer: z.boolean().optional(),
  redeem_by: z.string().optional(),
});

export const createCouponSchema = createCouponObject.superRefine((val, ctx) => {
  if (val.discount_type === 'percentage' && val.percent_off == null) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['percent_off'],
      message: 'Percentage off is required',
    });
  }

  if (val.discount_type === 'fixed') {
    if (val.amount_off == null) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['amount_off'],
        message: 'Amount off is required',
      });
    }
    if (!val.currency) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['currency'],
        message: 'Currency is required for a fixed amount',
      });
    }
  }

  if (val.duration === 'repeating' && val.duration_in_cycles == null) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['duration_in_cycles'],
      message: 'Number of cycles is required for a repeating discount',
    });
  }
});

export type CreateCouponFormValues = z.infer<typeof createCouponObject>;

const updateCouponObject = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  active: z.boolean(),
});

export const updateCouponSchema = updateCouponObject;
export type UpdateCouponFormValues = z.infer<typeof updateCouponObject>;

export const couponResolvers = {
  create: zodResolver(createCouponSchema),
  update: zodResolver(updateCouponSchema),
};

export const couponSchemas = {
  create: createCouponSchema,
  update: updateCouponSchema,
};

// Drift guards. The create form intentionally omits `applies_to_products` and
// `metadata` (no UI for them yet), so we assert the form keys are a SUBSET of
// the SDK input rather than an exact match — this still catches a renamed or
// stray key the moment the SDK changes.
type AssertSubset<A extends B, B> = A extends B ? true : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateCouponKeys: AssertSubset<
  keyof CreateCouponFormValues,
  keyof CreateCouponInput
> = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateCouponKeys: AssertSubset<
  keyof UpdateCouponFormValues,
  keyof UpdateCouponInput
> = true;
