import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for prices
export const priceKeys = {
  all: ['prices'] as const,
  lists: () => [...priceKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...priceKeys.lists(), filters] as const,
  details: () => [...priceKeys.all, 'detail'] as const,
  detail: (id: string) => [...priceKeys.details(), id] as const,
};

/**
 * Hook to fetch prices for a variant
 */
export function useVariantPrices(variantId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: priceKeys.list({ variantId }),
    queryFn: () => (client as any).variants?.getPrices?.(variantId) || Promise.resolve([]),
    enabled: !!variantId && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch a list of prices (delegated to client implementation)
 */
export function usePrices(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: priceKeys.list(params || {}),
    queryFn: () => (client as any).prices?.list?.(params) || Promise.resolve([]),
    enabled: options?.enabled !== false,
    ...options,
  });
}

/**
 * Hook to fetch a single price by ID
 */
export function usePrice(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: priceKeys.detail(id),
    queryFn: () => client.prices.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new price
 */
export function useCreatePrice(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.prices.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: priceKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing price
 */
export function useUpdatePrice(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.prices.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: priceKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: priceKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a price
 */
export function useDeletePrice(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.prices.delete(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: priceKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: priceKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
