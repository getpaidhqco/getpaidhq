import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateOrgRequest } from '@getpaidhq/sdk';

// Organization form validation — mirrors the server contract (CreateOrgRequest).

const createOrgObject = z.object({
  name: z.string().min(1, 'Name is required'),
  country: z.string().min(1, 'Country is required'),
  timezone: z.string().min(1, 'Timezone is required'),
  metadata: z.record(z.string(), z.string()).optional(),
});

export const createOrgSchema = createOrgObject;
export type CreateOrgFormValues = z.infer<typeof createOrgObject>;

export const orgResolvers = {
  create: zodResolver(createOrgSchema),
};

export const orgSchemas = {
  create: createOrgSchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateOrgKeys: Exact<keyof CreateOrgFormValues, keyof CreateOrgRequest> = true;
