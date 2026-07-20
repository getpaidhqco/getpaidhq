import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for subscriptions
export const subscriptionKeys = {
  all: ['subscriptions'] as const,
  lists: () => [...subscriptionKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...subscriptionKeys.lists(), filters] as const,
  details: () => [...subscriptionKeys.all, 'detail'] as const,
  detail: (id: string) => [...subscriptionKeys.details(), id] as const,
  payments: (id: string) => [...subscriptionKeys.detail(id), 'payments'] as const,
  usage: (id: string, params: Record<string, any>) => [...subscriptionKeys.detail(id), 'usage', params] as const,
};

/**
 * Hook to fetch a list of subscriptions
 */
export function useSubscriptions(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: subscriptionKeys.list(params || {}),
    queryFn: () => client.subscriptions.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single subscription by ID
 */
export function useSubscription(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: subscriptionKeys.detail(id),
    queryFn: () => client.subscriptions.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to update an existing subscription
 */
export function useUpdateSubscription(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.subscriptions.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to cancel a subscription
 */
export function useCancelSubscription(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: { id: string; reason?: string }) =>
      client.subscriptions.cancel(params.id, { reason: params.reason }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.detail(params.id) });
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to pause a subscription
 */
export function usePauseSubscription(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: { id: string; reason?: string }) =>
      client.subscriptions.pause(params.id, { reason: params.reason }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.detail(params.id) });
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to resume a subscription
 */
export function useResumeSubscription(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: { id: string; resume_behavior?: string }) =>
      client.subscriptions.resume(params.id, { resume_behavior: params.resume_behavior }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.detail(params.id) });
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch subscription payments
 * GET /api/subscriptions/{id}/payments
 */
export function useSubscriptionPayments(id: string, params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: subscriptionKeys.payments(id),
    queryFn: () => client.subscriptions.listPayments(id, params),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch subscription usage
 * GET /api/subscriptions/{id}/usage
 */
export function useSubscriptionUsageReport(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: subscriptionKeys.usage(id, {}),
    queryFn: () => client.subscriptions.getUsage(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to update subscription billing anchor
 * PATCH /api/subscriptions/{id}/billing-anchor
 */
export function useUpdateBillingAnchor(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: { id: string; billing_anchor: number; proration_mode: string }) =>
      client.subscriptions.updateBillingAnchor(params.id, {
        billing_anchor: params.billing_anchor,
        proration_mode: params.proration_mode,
      }),
    onSuccess: (_, params) => {
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.detail(params.id) });
      queryClient.invalidateQueries({ queryKey: subscriptionKeys.lists() });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
