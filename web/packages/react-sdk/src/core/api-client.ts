import {GetPaidHQClient, GetPaidHQClientConfig} from '@getpaidhq/sdk';

/**
 * Creates a GetPaidHQ API client configured with the provided API key and base URL.
 */
export function createApiClient(options?: GetPaidHQClientConfig): GetPaidHQClient {
  if (!options?.apiKey && !options?.getToken) {
    throw new Error('API key or bearer token is required to create a GetPaidHQ API client');
  }

  return new GetPaidHQClient(options);
}
