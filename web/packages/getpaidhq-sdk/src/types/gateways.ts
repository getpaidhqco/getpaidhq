/** Payment service provider gateway (spec: GatewayResponse). */
export interface GatewayResponse {
  /** Non-secret PSP configuration. Credentials are never echoed back. */
  config: Record<string, string> | null;
  created_at: string;
  id: string;
  name: string;
  psp: string;
  updated_at: string;
}

/** Create gateway input (spec: CreateGatewayRequest). */
export interface CreateGatewayRequest {
  /** Non-secret PSP configuration (e.g. environment flags). */
  config?: Record<string, string>;
  /** Secret PSP credentials (API keys). Stored encrypted, never returned. */
  credentials: Record<string, string>;
  name: string;
  psp: string;
}
