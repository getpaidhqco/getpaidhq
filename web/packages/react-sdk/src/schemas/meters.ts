import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateMeterRequest } from '@getpaidhq/sdk';

/**
 * Meter form validation.
 *
 * Lives in the react-sdk (the frontend integration package) — NOT in @getpaidhq/sdk —
 * because validating a frontend form is a frontend concern; the core SDK stays pure
 * transport + types. Consumers import a ready-made resolver and never write inline
 * rules or import zod themselves.
 *
 * Rules mirror the server contract, the single source of truth:
 *   - gphq-server/internal/adapter/http/meter_request.go (CreateMeterRequest tags)
 *   - gphq-server/internal/core/service/meter.go (MeterService.Create)
 *
 * Note these rules are RICHER than the SDK type can express (the `field_name`
 * cross-field rule, the aggregation/rounding enums) — which is exactly why validation
 * earns its own maintained layer instead of being inferred from the type. The
 * `_assertCreateMeterContract` guard at the bottom keeps it from drifting from the type.
 */

const aggregation = z.enum([
  'count',
  'sum',
  'max',
  'latest',
  'weighted_sum',
  'unique_count',
]);

const roundingMode = z.enum(['round', 'ceil', 'floor']);

/** A meter aggregation value. */
export type MeterAggregation = z.infer<typeof aggregation>;

/** Valid aggregation values — use to populate a UI dropdown. */
export const AGGREGATION_TYPES = aggregation.options;
/** Valid rounding modes — use to populate a UI dropdown. */
export const ROUNDING_MODES = roundingMode.options;

/**
 * Aggregations a carry-over (stock) meter may use — the standing-level readings.
 * Mirrors the switch in gphq-server validateCarryOver; `count`/`sum` are flow-only.
 */
export const CARRY_OVER_AGGREGATIONS = [
  'latest',
  'max',
  'unique_count',
  'weighted_sum',
] as const satisfies readonly MeterAggregation[];

/**
 * Aggregations that REQUIRE carry_over: a time-averaged quantity is a standing
 * level by definition, so a flow weighted_sum would reset and underbill.
 */
export const CARRY_OVER_REQUIRED_AGGREGATIONS = [
  'weighted_sum',
] as const satisfies readonly MeterAggregation[];

/** Whether an aggregation is valid on a carry-over meter. */
export const isCarryOverAggregation = (a: MeterAggregation): boolean =>
  (CARRY_OVER_AGGREGATIONS as readonly MeterAggregation[]).includes(a);

/** Whether an aggregation can only be used with carry_over enabled. */
export const requiresCarryOver = (a: MeterAggregation): boolean =>
  (CARRY_OVER_REQUIRED_AGGREGATIONS as readonly MeterAggregation[]).includes(a);

const meterFilter = z.object({
  field: z.string().min(1, 'Field is required').max(255),
  values: z
    .array(z.string().min(1).max(255))
    .min(1, 'At least one value is required'),
});

// The bare object shape. The form's type is inferred from THIS (a plain ZodObject) —
// inferring from the .superRefine() wrapper below makes TS comparisons "excessively deep".
const createMeterObject = z.object({
  code: z.string().min(1, 'Code is required').max(255),
  name: z.string().min(1, 'Name is required').max(255),
  aggregation,
  field_name: z.string().max(255).optional(),
  carry_over: z.boolean().optional(),
  rounding_mode: roundingMode.optional(),
  rounding_scale: z.number().int().gte(0).lte(18).optional(),
  // Filters = rate dimensions; group_by = breakout dimensions (v1: at most one).
  filters: z.array(meterFilter).optional(),
  group_by: z
    .array(z.string().min(1).max(255))
    .max(1, 'At most one group dimension is supported')
    .optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const createMeterSchema = createMeterObject.superRefine((val, ctx) => {
  // Server rule: every aggregation except `count` needs a field to read from.
  if (val.aggregation !== 'count' && !val.field_name) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['field_name'],
      message: 'Field name is required for this aggregation',
    });
  }

  // Carry-over (stock) rules — mirror gphq-server validateCarryOver.
  if (val.carry_over) {
    // Stock meters read a standing level; flow aggregations have no meaning.
    if (!isCarryOverAggregation(val.aggregation)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['aggregation'],
        message: `${val.aggregation} is not supported for carry-over meters`,
      });
    }
    // Filters/group_by have no defined replay semantics on a stock ledger.
    if (val.filters && val.filters.length > 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['filters'],
        message: 'Filters are not supported on carry-over meters',
      });
    }
    if (val.group_by && val.group_by.length > 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['group_by'],
        message: 'Group by is not supported on carry-over meters',
      });
    }
  } else if (requiresCarryOver(val.aggregation)) {
    // e.g. weighted_sum: a time-average is a standing level, so it needs carry_over.
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['carry_over'],
      message: `${val.aggregation} requires carry over to be enabled`,
    });
  }
});

/** Form values for creating a meter (inferred from the schema — the form's type). */
export type CreateMeterFormValues = z.infer<typeof createMeterObject>;

/** Pre-built React Hook Form resolvers — `useForm({ resolver: meterResolvers.create })`. */
export const meterResolvers = {
  create: zodResolver(createMeterSchema),
};

/** Raw zod schemas, exposed for advanced composition (extend/pick/merge). */
export const meterSchemas = {
  create: createMeterSchema,
};

/**
 * Compile-time drift guard. If the server adds or removes a CreateMeterRequest field
 * and the SDK regenerates its type, one of these fails to typecheck until the schema
 * above is updated — the safety net the old hand-rolled web schema never had.
 *
 * This compares the FIELD SET in both directions (shallow). A full structural compare
 * of the zod-inferred type against the interface trips TS's "excessively deep"
 * instantiation limit, so we deliberately assert keys, not value types — field
 * add/remove is the drift that actually occurs across the spec.
 */
type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateMeterKeys: Exact<
  keyof CreateMeterFormValues,
  keyof CreateMeterRequest
> = true;
