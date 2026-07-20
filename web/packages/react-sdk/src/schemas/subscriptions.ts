import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { UpdateSubscriptionRequest } from '@getpaidhq/sdk';

/**
 * Subscription update form validation — mirrors the server contract
 * (UpdateSubscriptionRequest). All fields optional (partial update). See meters.ts.
 */

const updateSubscriptionObject = z.object({
  id: z.string().optional(),
  status: z.string().optional(),
  default_payment_method: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const updateSubscriptionSchema = updateSubscriptionObject;
export type UpdateSubscriptionFormValues = z.infer<typeof updateSubscriptionObject>;

export const subscriptionResolvers = {
  update: zodResolver(updateSubscriptionSchema),
};

export const subscriptionSchemas = {
  update: updateSubscriptionSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateSubscriptionKeys: Exact<
  keyof UpdateSubscriptionFormValues,
  keyof UpdateSubscriptionRequest
> = true;
