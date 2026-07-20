import { useQuery } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for payments
export const paymentKeys = {
  all: ['payments'] as const,
  lists: () => [...paymentKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...paymentKeys.lists(), filters] as const,
  details: () => [...paymentKeys.all, 'detail'] as const,
  detail: (id: string) => [...paymentKeys.details(), id] as const,
  methods: () => [...paymentKeys.all, 'methods'] as const,
  method: (id: string) => [...paymentKeys.methods(), id] as const,
};

/**
 * Hook to fetch a list of payments
 * GET /api/payments
 */
export function usePayments(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: paymentKeys.list(params || {}),
    queryFn: () => client.payments.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single payment by ID
 * GET /api/payments/{id}
 */
export function usePayment(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: paymentKeys.detail(id),
    queryFn: () => client.payments.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch a payment method by ID
 * GET /api/payment-methods/{id}
 */
export function usePaymentMethod(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: paymentKeys.method(id),
    queryFn: () => client.payments.getPaymentMethod(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}
