/** Create API key input (spec: CreateApiKeyInput). */
export interface CreateApiKeyInput {
  name?: string;
}

/**
 * API key creation response (spec: ApiKeyCreateResponse).
 * The `key` field contains the full secret and is only returned once, at creation time.
 */
export interface ApiKeyCreateResponse {
  created_at: string;
  id: string;
  key: string;
  name?: string | null;
  updated_at: string;
}
