import {useGetPaidHQContext} from '../providers/getpaidhq-provider.js';
import {createApiClient} from '../core/api-client.js';

/**
 * Hook to access the GetPaidHQ API client.
 * This hook should be used by all resource-specific hooks.
 */
export function useGetPaidHQClient() {
  const {apiKey, getToken, baseUrl} = useGetPaidHQContext();

  return createApiClient({
    apiKey,
    getToken,
    baseURL: baseUrl
  });
}
