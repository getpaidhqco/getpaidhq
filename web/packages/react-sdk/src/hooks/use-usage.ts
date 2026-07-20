import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for usage
export const usageKeys = {
  all: ['usage'] as const,
  subscriptionUsage: (id: string) => [...usageKeys.all, 'subscription', id] as const,
};

/**
 * Hook to ingest usage events
 * POST /api/usage/ingest
 */
export function useIngestUsage(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.usage.ingest(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: usageKeys.all });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch usage for a subscription
 * GET /api/subscriptions/{id}/usage
 */
export function useSubscriptionUsage(subscriptionId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: usageKeys.subscriptionUsage(subscriptionId),
    queryFn: () => client.subscriptions.getUsage(subscriptionId),
    enabled: !!subscriptionId && (options?.enabled !== false),
    ...options,
  });
}

// Backward-compatible alias
export const useRecordUsage = useIngestUsage;
