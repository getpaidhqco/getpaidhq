import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { CreateSessionRequest, CreateSessionResponse } from '@getpaidhq/sdk';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for sessions
export const sessionKeys = {
  all: ['sessions'] as const,
};

export type { CreateSessionRequest, CreateSessionResponse };

/**
 * Hook to create a checkout session
 * POST /api/sessions
 */
export function useCreateSession(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateSessionRequest) => client.sessions.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.all });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
