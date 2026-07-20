import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for customers
export const customerKeys = {
  all: ['customers'] as const,
  lists: () => [...customerKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...customerKeys.lists(), filters] as const,
  details: () => [...customerKeys.all, 'detail'] as const,
  detail: (id: string) => [...customerKeys.details(), id] as const,
};

/**
 * Hook to fetch a list of customers
 */
export function useCustomers(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: customerKeys.list(params || {}),
    queryFn: () => client.customers.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single customer by ID
 */
export function useCustomer(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: customerKeys.detail(id),
    queryFn: () => client.customers.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new customer
 */
export function useCreateCustomer(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.customers.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: customerKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to create a customer payment method
 */
export function useCreateCustomerPaymentMethod(customerId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.customers.createPaymentMethod(customerId, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: customerKeys.detail(customerId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update a customer payment method
 */
export function useUpdateCustomerPaymentMethod(customerId: string, paymentMethodId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.customers.updatePaymentMethod(customerId, paymentMethodId, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: customerKeys.detail(customerId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch customer dunning history
 * GET /api/customers/{id}/dunning-history
 */
export function useCustomerDunningHistory(customerId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: [...customerKeys.detail(customerId), 'dunning-history'],
    queryFn: () => client.customers.getDunningHistory(customerId),
    enabled: !!customerId && (options?.enabled !== false),
    ...options,
  });
}
