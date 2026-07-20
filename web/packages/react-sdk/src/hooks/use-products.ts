import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for products
export const productKeys = {
  all: ['products'] as const,
  lists: () => [...productKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...productKeys.lists(), filters] as const,
  details: () => [...productKeys.all, 'detail'] as const,
  detail: (id: string) => [...productKeys.details(), id] as const,
};

/**
 * Hook to fetch a list of products
 */
export function useProducts(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: productKeys.list(params || {}),
    queryFn: () => client.products.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single product by ID
 */
export function useProduct(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: productKeys.detail(id),
    queryFn: () => client.products.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new product
 */
export function useCreateProduct(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.products.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: productKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing product
 */
export function useUpdateProduct(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.products.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: productKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: productKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to archive a product (POST /products/{id}/archive).
 * Archiving hides the product from default listings and blocks new sales while
 * preserving all history. Idempotent and reversible via useUnarchiveProduct.
 */
export function useArchiveProduct(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.products.archive(id),
    onSuccess: (data, id) => {
      queryClient.invalidateQueries({ queryKey: productKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: productKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to unarchive a product (POST /products/{id}/unarchive).
 * Restores the product to active (sellable, listed). Idempotent.
 */
export function useUnarchiveProduct(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.products.unarchive(id),
    onSuccess: (data, id) => {
      queryClient.invalidateQueries({ queryKey: productKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: productKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a product
 */
export function useDeleteProduct(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.products.delete(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: productKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: productKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
