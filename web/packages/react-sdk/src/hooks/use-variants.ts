import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for variants
export const variantKeys = {
  all: ['variants'] as const,
  lists: () => [...variantKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...variantKeys.lists(), filters] as const,
  details: () => [...variantKeys.all, 'detail'] as const,
  detail: (id: string) => [...variantKeys.details(), id] as const,
};

/**
 * Hook to fetch variants for a product
 */
export function useProductVariants(productId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: variantKeys.list({ productId }),
    queryFn: () => (client as any).products?.listVariants?.(productId) || Promise.resolve([]),
    enabled: !!productId && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch a list of variants (fallback)
 */
export function useVariants(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: variantKeys.list(params || {}),
    queryFn: () => (client as any).variants?.list?.(params) || Promise.resolve([]),
    enabled: options?.enabled !== false,
    ...options,
  });
}

/**
 * Hook to fetch a single variant by ID
 */
export function useVariant(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: variantKeys.detail(id),
    queryFn: () => client.variants.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new variant for a product
 */
export function useCreateProductVariant(productId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => (client as any).products?.createVariant?.(productId, data) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: variantKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to create a new variant (fallback)
 */
export function useCreateVariant(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => (client as any).variants?.create?.(data) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: variantKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing variant
 */
export function useUpdateVariant(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.variants.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: variantKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: variantKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a variant
 */
export function useDeleteVariant(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.variants.delete(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: variantKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: variantKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
