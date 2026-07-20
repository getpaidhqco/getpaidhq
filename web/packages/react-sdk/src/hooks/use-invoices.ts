import { useQuery } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for invoices
export const invoiceKeys = {
  all: ['invoices'] as const,
  lists: () => [...invoiceKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...invoiceKeys.lists(), filters] as const,
  details: () => [...invoiceKeys.all, 'detail'] as const,
  detail: (id: string) => [...invoiceKeys.details(), id] as const,
};

/**
 * Hook to fetch a list of invoices
 * GET /api/invoices
 */
export function useInvoices(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: invoiceKeys.list(params || {}),
    queryFn: () => client.invoices.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single invoice by ID
 * GET /api/invoices/{id}
 */
export function useInvoice(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: invoiceKeys.detail(id),
    queryFn: () => client.invoices.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}
