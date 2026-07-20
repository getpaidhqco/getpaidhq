import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateVariantRequest, UpdateVariantRequest } from '@getpaidhq/sdk';

// Variant form validation — mirrors the server contract. See meters.ts for rationale.

const variantObject = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  description: z.string().optional(),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const createVariantSchema = variantObject;
export const updateVariantSchema = variantObject;
export type CreateVariantFormValues = z.infer<typeof variantObject>;
export type UpdateVariantFormValues = z.infer<typeof variantObject>;

export const variantResolvers = {
  create: zodResolver(createVariantSchema),
  update: zodResolver(updateVariantSchema),
};

export const variantSchemas = {
  create: createVariantSchema,
  update: updateVariantSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateVariantKeys: Exact<keyof CreateVariantFormValues, keyof CreateVariantRequest> = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateVariantKeys: Exact<keyof UpdateVariantFormValues, keyof UpdateVariantRequest> = true;
