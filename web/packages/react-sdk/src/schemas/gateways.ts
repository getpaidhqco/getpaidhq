import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import type { CreateGatewayRequest } from '@getpaidhq/sdk';

/**
 * Gateway form validation — mirrors the server contract (CreateGatewayRequest).
 * `psp` is a free string at the contract level (e.g. "paystack", "checkout"); the form
 * supplies the choices. `credentials` holds the secret PSP API keys (stored encrypted
 * server-side, never returned); `config` is the optional non-secret settings bag.
 */

const createGatewayObject = z.object({
  name: z.string().min(1, 'Name is required'),
  psp: z.string().min(1, 'Provider is required'),
  credentials: z
    .record(z.string(), z.string())
    .refine((c) => Object.keys(c).length > 0, 'Credentials are required'),
  config: z.record(z.string(), z.string()).optional(),
});

export const createGatewaySchema = createGatewayObject;
export type CreateGatewayFormValues = z.infer<typeof createGatewayObject>;

export const gatewayResolvers = {
  create: zodResolver(createGatewaySchema),
};

export const gatewaySchemas = {
  create: createGatewaySchema,
};

type Exact<A, B> = A extends B ? (B extends A ? true : never) : never;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _assertCreateGatewayKeys: Exact<keyof CreateGatewayFormValues, keyof CreateGatewayRequest> = true;
