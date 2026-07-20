import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type {
  CreateDunningConfigurationRequest,
  UpdateDunningConfigurationRequest,
} from '@getpaidhq/sdk';

/**
 * Dunning configuration form validation — mirrors the server contract.
 * `config`/`target_rules` are nested structures modelled permissively. See meters.ts.
 */

const createDunningObject = z.object({
  name: z.string().min(1, 'Name is required'),
  applies_to: z.string().min(1, 'Applies-to is required'),
  config: z.record(z.string(), z.any()),
  description: z.string().optional(),
  priority: z.number().int().optional(),
  is_ab_test: z.boolean().optional(),
  ab_test_percentage: z.number().optional(),
  target_rules: z.record(z.string(), z.any()).optional(),
});

export const createDunningSchema = createDunningObject;
export type CreateDunningFormValues = z.infer<typeof createDunningObject>;

const updateDunningObject = z.object({
  name: z.string().min(1).optional(),
  applies_to: z.string().optional(),
  config: z.record(z.string(), z.any()).optional(),
  description: z.string().optional(),
  priority: z.number().int().optional(),
  is_ab_test: z.boolean().optional(),
  ab_test_percentage: z.number().optional(),
  status: z.string().optional(),
  target_rules: z.record(z.string(), z.any()).optional(),
});

export const updateDunningSchema = updateDunningObject;
export type UpdateDunningFormValues = z.infer<typeof updateDunningObject>;

export const dunningResolvers = {
  create: zodResolver(createDunningSchema),
  update: zodResolver(updateDunningSchema),
};

export const dunningSchemas = {
  create: createDunningSchema,
  update: updateDunningSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateDunningKeys: Exact<
  keyof CreateDunningFormValues,
  keyof CreateDunningConfigurationRequest
> = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertUpdateDunningKeys: Exact<
  keyof UpdateDunningFormValues,
  keyof UpdateDunningConfigurationRequest
> = true;
