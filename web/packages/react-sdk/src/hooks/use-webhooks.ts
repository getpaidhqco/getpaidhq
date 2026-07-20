import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for webhooks
export const webhookKeys = {
  all: ['webhooks'] as const,
  lists: () => [...webhookKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...webhookKeys.lists(), filters] as const,
  details: () => [...webhookKeys.all, 'detail'] as const,
  detail: (id: string) => [...webhookKeys.details(), id] as const,
  events: () => [...webhookKeys.all, 'events'] as const,
};

/**
 * Hook to fetch a list of webhooks
 */
export function useWebhooks(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: webhookKeys.list(params || {}),
    queryFn: () => client.webhooks.list(),
    ...options,
  });
}

/**
 * Hook to fetch a single webhook by ID
 */
export function useWebhook(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: webhookKeys.detail(id),
    queryFn: () => (client as any).webhooks?.get?.(id) || Promise.resolve(null),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch available webhook events
 */
export function useWebhookEvents(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: webhookKeys.events(),
    queryFn: () => (client as any).webhooks?.getEvents?.() || Promise.resolve([]),
    ...options,
  });
}

/**
 * Hook to create a new webhook
 */
export function useCreateWebhook(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.webhooks.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing webhook
 */
export function useUpdateWebhook(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => (client as any).webhooks?.update?.(id, data) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a webhook
 */
export function useDeleteWebhook(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => (client as any).webhooks?.delete?.(id) || Promise.reject(new Error('Not supported')),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: webhookKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to test a webhook
 */
export function useTestWebhook(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useMutation({
    mutationFn: (id: string) => (client as any).webhooks?.test?.(id) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}