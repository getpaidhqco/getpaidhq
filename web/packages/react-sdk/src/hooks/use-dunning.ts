import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for dunning
export const dunningKeys = {
  all: ['dunning'] as const,
  campaigns: () => [...dunningKeys.all, 'campaigns'] as const,
  campaignsList: (filters: Record<string, any>) => [...dunningKeys.campaigns(), 'list', filters] as const,
  campaignDetail: (id: string) => [...dunningKeys.campaigns(), 'detail', id] as const,
  campaignAttempts: (id: string) => [...dunningKeys.campaignDetail(id), 'attempts'] as const,
  campaignCommunications: (id: string) => [...dunningKeys.campaignDetail(id), 'communications'] as const,
  configurations: () => [...dunningKeys.all, 'configurations'] as const,
  configurationDetail: (id: string) => [...dunningKeys.configurations(), 'detail', id] as const,
  paymentTokens: () => [...dunningKeys.all, 'payment-tokens'] as const,
};

/**
 * Hook to fetch dunning campaigns
 */
export function useDunningCampaigns(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.campaignsList(params || {}),
    queryFn: () => client.dunning.listCampaigns(params),
    ...options,
  });
}

/**
 * Hook to fetch a single dunning campaign by ID
 */
export function useDunningCampaign(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.campaignDetail(id),
    queryFn: () => client.dunning.getCampaign(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch campaign attempts
 */
export function useCampaignAttempts(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.campaignAttempts(id),
    queryFn: () => client.dunning.listCampaignAttempts(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch campaign communications
 */
export function useCampaignCommunications(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.campaignCommunications(id),
    queryFn: () => client.dunning.listCampaignCommunications(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to fetch dunning configurations
 */
export function useDunningConfigurations(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.configurations(),
    queryFn: () => client.dunning.listConfigurations(),
    ...options,
  });
}

/**
 * Hook to fetch a single dunning configuration by ID
 */
export function useDunningConfiguration(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: dunningKeys.configurationDetail(id),
    queryFn: () => client.dunning.getConfiguration(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to update a dunning campaign
 */
export function useUpdateDunningCampaign(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.dunning.updateCampaign(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: dunningKeys.campaignDetail(id) });
      queryClient.invalidateQueries({ queryKey: dunningKeys.campaigns() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to trigger a manual dunning attempt
 */
export function useTriggerManualAttempt(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.dunning.triggerManualAttempt(id),
    onSuccess: (data, id) => {
      queryClient.invalidateQueries({ queryKey: dunningKeys.campaignAttempts(id) });
      queryClient.invalidateQueries({ queryKey: dunningKeys.campaignDetail(id) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to create a dunning configuration
 */
export function useCreateDunningConfiguration(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.dunning.createConfiguration(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: dunningKeys.configurations() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update a dunning configuration
 */
export function useUpdateDunningConfiguration(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.dunning.updateConfiguration(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: dunningKeys.configurationDetail(id) });
      queryClient.invalidateQueries({ queryKey: dunningKeys.configurations() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to verify a payment token
 */
export function useVerifyPaymentToken(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useMutation({
    mutationFn: (token: string) => (client as any).paymentTokens?.verify?.({ token }) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to activate a payment token
 */
export function useActivatePaymentToken(options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useMutation({
    mutationFn: (params: { token: string; payment_method_token: string }) => 
      (client as any).paymentTokens?.activate?.(params) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to create a payment token for a subscription
 */
export function useCreateSubscriptionPaymentToken(subscriptionId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useMutation({
    mutationFn: () => (client as any).admin?.createSubscriptionPaymentToken?.(subscriptionId) || Promise.reject(new Error('Not supported')),
    onSuccess: (data) => {
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}