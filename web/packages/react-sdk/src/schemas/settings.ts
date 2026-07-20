import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateSettingRequest, UpdateSettingRequest } from '@getpaidhq/sdk';

// Setting form validation — mirrors the server contract. See meters.ts.

const createSettingObject = z.object({
  id: z.string().min(1, 'Key is required').max(255),
  parent_id: z.string().max(255).optional(),
  type: z.string().max(64).optional(),
  value: z.string().optional(),
});

export const createSettingSchema = createSettingObject;
export type CreateSettingFormValues = z.infer<typeof createSettingObject>;

const updateSettingObject = z.object({
  type: z.string().max(64).optional(),
  value: z.string().optional(),
});

export const updateSettingSchema = updateSettingObject;
export type UpdateSettingFormValues = z.infer<typeof updateSettingObject>;

export const settingResolvers = {
  create: zodResolver(createSettingSchema),
  update: zodResolver(updateSettingSchema),
};

export const settingSchemas = {
  create: createSettingSchema,
  update: updateSettingSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateSettingKeys: Exact<keyof CreateSettingFormValues, keyof CreateSettingRequest> = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateSettingKeys: Exact<keyof UpdateSettingFormValues, keyof UpdateSettingRequest> = true;
