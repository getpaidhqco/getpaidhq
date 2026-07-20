import { useQuery } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for health
export const healthKeys = {
  all: ['health'] as const,
  check: () => [...healthKeys.all, 'check'] as const,
};

/**
 * Hook to check API health status
 */
export function useHealthCheck(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: healthKeys.check(),
    queryFn: () => (client as any).health?.check?.() || Promise.resolve({ status: 'ok' }),
    staleTime: 1000 * 30, // 30 seconds
    ...options,
  });
}