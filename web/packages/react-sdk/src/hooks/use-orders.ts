import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for orders
export const orderKeys = {
  all: ['orders'] as const,
  lists: () => [...orderKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...orderKeys.lists(), filters] as const,
  details: () => [...orderKeys.all, 'detail'] as const,
  detail: (id: string) => [...orderKeys.details(), id] as const,
  carts: () => [...orderKeys.all, 'carts'] as const,
  cart: (id: string) => [...orderKeys.carts(), id] as const,
};

/**
 * Hook to fetch a list of orders
 */
export function useOrders(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: orderKeys.list(params || {}),
    queryFn: () => client.orders.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single order by ID
 */
export function useOrder(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: orderKeys.detail(id),
    queryFn: () => client.orders.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new order
 */
export function useCreateOrder(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.orders.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: orderKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing order
 */
export function useUpdateOrder(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => (client as any).orders?.update?.(id, data) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: orderKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: orderKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to cancel an order
 */
export function useCancelOrder(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => (client as any).orders?.cancel?.(id) || Promise.reject(new Error('Not supported')),
    onSuccess: (data, id) => {
      queryClient.invalidateQueries({ queryKey: orderKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: orderKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to complete an order
 */
export function useCompleteOrder(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: { id: string; data?: any }) => client.orders.complete(params.id, params.data),
    onSuccess: (data, params) => {
      queryClient.invalidateQueries({ queryKey: orderKeys.detail(params.id) });
      queryClient.invalidateQueries({ queryKey: orderKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch order subscriptions
 */
export function useOrderSubscriptions(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: [...orderKeys.detail(id), 'subscriptions'],
    queryFn: () => (client as any).orders?.getSubscriptions?.(id) || Promise.resolve([]),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}