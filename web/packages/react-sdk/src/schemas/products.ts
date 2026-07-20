import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateProductRequest, UpdateProductRequest } from '@getpaidhq/sdk';

/**
 * Product form validation. Mirrors the server contract
 * (gphq-server CreateProductRequest / UpdateProductRequest). See meters.ts for the
 * rationale: validation lives in react-sdk, guarded against the SDK types.
 */

const variantInput = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

const createProductObject = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
  variants: z.array(variantInput).min(1, 'At least one variant is required'),
});

export const createProductSchema = createProductObject;
export type CreateProductFormValues = z.infer<typeof createProductObject>;

const updateProductObject = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const updateProductSchema = updateProductObject;
export type UpdateProductFormValues = z.infer<typeof updateProductObject>;

export const productResolvers = {
  create: zodResolver(createProductSchema),
  update: zodResolver(updateProductSchema),
};

export const productSchemas = {
  create: createProductSchema,
  update: updateProductSchema,
};

// Drift guards (shallow — see meters.ts).
type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateProductKeys: Exact<
  keyof CreateProductFormValues,
  keyof CreateProductRequest
> = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateProductKeys: Exact<
  keyof UpdateProductFormValues,
  keyof UpdateProductRequest
> = true;
