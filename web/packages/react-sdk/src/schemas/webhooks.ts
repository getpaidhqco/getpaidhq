import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateWebhookSubscriptionRequest } from '@getpaidhq/sdk';

/**
 * Webhook subscription form validation — mirrors the server contract
 * (CreateWebhookSubscriptionRequest). See meters.ts for the layering rationale.
 */

const createWebhookObject = z.object({
  url: z.string().url('Must be a valid URL'),
  events: z.array(z.string()).min(1, 'Select at least one event'),
  secret: z.string().optional(),
});

export const createWebhookSchema = createWebhookObject;
export type CreateWebhookFormValues = z.infer<typeof createWebhookObject>;

export const webhookResolvers = {
  create: zodResolver(createWebhookSchema),
};

export const webhookSchemas = {
  create: createWebhookSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateWebhookKeys: Exact<
  keyof CreateWebhookFormValues,
  keyof CreateWebhookSubscriptionRequest
> = true;
