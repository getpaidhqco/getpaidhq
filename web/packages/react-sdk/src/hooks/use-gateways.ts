import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for gateways
export const gatewayKeys = {
  all: ['gateways'] as const,
  lists: () => [...gatewayKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...gatewayKeys.lists(), filters] as const,
  details: () => [...gatewayKeys.all, 'detail'] as const,
  detail: (id: string) => [...gatewayKeys.details(), id] as const,
};

/**
 * Hook to fetch a list of payment gateways
 */
export function useGateways(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: gatewayKeys.list(params || {}),
    queryFn: () => (client as any).gateways?.list?.(params) || Promise.resolve([]),
    ...options,
  });
}

/**
 * Hook to fetch a single gateway by ID
 */
export function useGateway(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: gatewayKeys.detail(id),
    queryFn: () => (client as any).gateways?.get?.(id) || Promise.resolve(null),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new gateway
 */
export function useCreateGateway(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.gateways.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update a gateway
 */
export function useUpdateGateway(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, ...data }: { id: string; [key: string]: any }) =>
      (client as any).gateways?.update?.(id, data) || Promise.reject(new Error('Not supported')),
    onSuccess: (data, { id }) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: gatewayKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a gateway
 */
export function useDeleteGateway(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => (client as any).gateways?.delete?.(id) || Promise.reject(new Error('Not supported')),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.lists() });
      queryClient.removeQueries({ queryKey: gatewayKeys.detail(id) });
      options?.onSuccess?.(id);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}