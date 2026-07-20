import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for organizations
export const organizationKeys = {
  all: ['organizations'] as const,
  lists: () => [...organizationKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...organizationKeys.lists(), filters] as const,
  details: () => [...organizationKeys.all, 'detail'] as const,
  detail: (id: string) => [...organizationKeys.details(), id] as const,
};

/**
 * Hook to create a new organization
 * POST /api/organizations
 */
export function useCreateOrganization(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.organizations.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
