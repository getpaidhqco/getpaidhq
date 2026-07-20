import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateOrderRequest } from '@getpaidhq/sdk';

/**
 * Order form validation — mirrors the server contract (CreateOrderRequest).
 * customer/cart/options are nested objects modelled permissively; the form builds
 * them up. See meters.ts for the layering rationale.
 */

const createOrderObject = z.object({
  psp_id: z.string().min(1, 'Payment provider is required'),
  customer: z.record(z.string(), z.any()),
  cart: z.record(z.string(), z.any()).optional(),
  options: z.record(z.string(), z.any()).optional(),
  payment_method_id: z.string().optional(),
  session_id: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const createOrderSchema = createOrderObject;
export type CreateOrderFormValues = z.infer<typeof createOrderObject>;

export const orderResolvers = {
  create: zodResolver(createOrderSchema),
};

export const orderSchemas = {
  create: createOrderSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateOrderKeys: Exact<keyof CreateOrderFormValues, keyof CreateOrderRequest> = true;
