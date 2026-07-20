import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for settings
export const settingsKeys = {
  all: ['settings'] as const,
  lists: (parentId: string) => [...settingsKeys.all, 'list', parentId] as const,
  details: () => [...settingsKeys.all, 'detail'] as const,
  detail: (parentId: string, id: string) => [...settingsKeys.details(), parentId, id] as const,
  gateways: () => [...settingsKeys.all, 'gateways'] as const,
  sessions: () => [...settingsKeys.all, 'sessions'] as const,
};

/**
 * Hook to fetch settings for an organization
 */
export function useSettings(parentId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: settingsKeys.lists(parentId),
    queryFn: () => client.settings.list(),
    enabled: !!parentId && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch a specific setting
 */
export function useSetting(parentId: string, id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: settingsKeys.detail(parentId, id),
    queryFn: () => client.settings.get(parentId, id),
    enabled: !!parentId && !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new setting
 */
export function useCreateSetting(parentId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.settings.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.lists(parentId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update a setting
 */
export function useUpdateSetting(parentId: string, id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.settings.update(parentId, id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.detail(parentId, id) });
      queryClient.invalidateQueries({ queryKey: settingsKeys.lists(parentId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a setting
 */
export function useDeleteSetting(parentId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.settings.delete(parentId, id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.detail(parentId, id) });
      queryClient.invalidateQueries({ queryKey: settingsKeys.lists(parentId) });
      options?.onSuccess?.(_);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to create a PSP configuration
 */
export function useCreatePspConfiguration(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.gateways.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.gateways() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}


// Convenience hooks for common settings patterns
export const useGeneralSettings = (orgId: string, options?: QueryOptions) => 
  useSettings(orgId, options);

export const useUpdateGeneralSettings = (orgId: string, settingId: string, options?: QueryOptions) => 
  useUpdateSetting(orgId, settingId, options);

export const useBillingSettings = (orgId: string, options?: QueryOptions) => 
  useSettings(orgId, options);

export const useUpdateBillingSettings = (orgId: string, settingId: string, options?: QueryOptions) => 
  useUpdateSetting(orgId, settingId, options);

export const useApiKeys = (orgId: string, options?: QueryOptions) => 
  useSettings(orgId, options);

export const useCreateApiKey = (orgId: string, options?: QueryOptions) => 
  useCreateSetting(orgId, options);

export const useDeleteApiKey = (orgId: string, options?: QueryOptions) => 
  useDeleteSetting(orgId, options);