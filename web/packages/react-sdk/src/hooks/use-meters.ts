import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for meters
export const meterKeys = {
  all: ['meters'] as const,
  lists: () => [...meterKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...meterKeys.lists(), filters] as const,
  details: () => [...meterKeys.all, 'detail'] as const,
  detail: (id: string) => [...meterKeys.details(), id] as const,
};

/**
 * Hook to fetch a list of meters
 * GET /api/meters
 */
export function useMeters(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: meterKeys.list(params || {}),
    queryFn: () => client.meters.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single meter by ID
 * GET /api/meters/{id}
 */
export function useMeter(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: meterKeys.detail(id),
    queryFn: () => client.meters.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new meter
 * POST /api/meters
 */
export function useCreateMeter(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.meters.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: meterKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
