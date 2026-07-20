"use client"
import {createContext, ReactNode, useCallback, useContext} from "react";
import {useMutation, useQuery} from "@tanstack/react-query";
import type {SubscriptionResponse} from "@getpaidhq/sdk";
import {useAuth} from "@getpaidhq/auth";
import {AuthHeader} from "@getpaidhq/auth";

type ApiError = {
  message: string
  code: string
  errors: { [key: string]: string[] }
};

export type SubscriptionPauseOptions = { reason?: string };
export type SubscriptionCancelOptions = { reason?: string };
export type SubscriptionResumeOptions = {
  resume_behavior: "start_new_billing_period" | "continue_existing_billing_period"
};

type FetchError = Error & { body?: ApiError };

async function jsonOrThrow(rsp: Response) {
  if (!rsp.ok) {
    const body = await rsp.json().catch(() => undefined);
    throw Object.assign(new Error(`${rsp.status} ${rsp.statusText}`), { body }) as FetchError;
  }
  return rsp.json();
}

interface SubscriptionContextType {
  subscription: SubscriptionResponse;
  isLoading: boolean;
  error: Error | null;
  refreshData: () => void;
  pause: (id: string, options: SubscriptionPauseOptions) => Promise<any>;
  resume: (id: string, options: SubscriptionResumeOptions) => Promise<any>;
  cancel: (id: string, options: SubscriptionCancelOptions) => Promise<any>;
}

const SubscriptionContext = createContext<SubscriptionContextType | undefined>(undefined);

export function SubscriptionProvider({
                                       children,
                                       subscription
                                     }: {
  children: ReactNode;
  subscription: SubscriptionResponse;
}) {
  const {getAuthHeaders} = useAuth();
  const subscriptionId = subscription.id;

  const fetchSubscription = async (authHeaders: AuthHeader): Promise<SubscriptionResponse> => {
    const response = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL}/api/subscriptions/${subscriptionId}`,
      {headers: authHeaders}
    );
    if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
    return response.json();
  };

  const {data: fetchedSubscription, isLoading, error, refetch} = useQuery({
    queryKey: ['subscription', subscriptionId],
    queryFn: async () => {
      const headers = await getAuthHeaders();
      return fetchSubscription(headers);
    },
    enabled: !!subscriptionId,
    initialData: subscription,
  });

  const subscriptionData = fetchedSubscription || subscription;

  const refreshData = useCallback(() => {
    refetch();
  }, [refetch]);

  const putAction = useCallback(async <T,>(id: string, action: string, options: T) => {
    const headers = await getAuthHeaders();
    const rsp = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL}/api/subscriptions/${id}/${action}`,
      {
        method: "PUT",
        headers: { ...headers, "Content-Type": "application/json" },
        body: JSON.stringify(options),
      }
    );
    return jsonOrThrow(rsp);
  }, [getAuthHeaders]);

  const pauseMutation = useMutation({
    mutationFn: ({id, options}: { id: string, options: SubscriptionPauseOptions }) =>
      putAction(id, "pause", options),
    onSuccess: () => refreshData(),
  });

  const resumeMutation = useMutation({
    mutationFn: ({id, options}: { id: string, options: SubscriptionResumeOptions }) =>
      putAction(id, "resume", options),
    onSuccess: () => refreshData(),
  });

  const cancelMutation = useMutation({
    mutationFn: ({id, options}: { id: string, options: SubscriptionCancelOptions }) =>
      putAction(id, "cancel", options),
    onSuccess: () => refreshData(),
  });

  const pause = useCallback((id: string, options: SubscriptionPauseOptions) => {
    return new Promise((resolve, reject) => {
      pauseMutation.mutate({id, options}, {
        onSuccess: (data) => resolve(data),
        onError: (error: Error) => reject((error as FetchError).body),
      });
    });
  }, [pauseMutation]);

  const resume = useCallback((id: string, options: SubscriptionResumeOptions) => {
    return new Promise((resolve, reject) => {
      resumeMutation.mutate({id, options}, {
        onSuccess: (data) => resolve(data),
        onError: (error: Error) => reject((error as FetchError).body),
      });
    });
  }, [resumeMutation]);

  const cancel = useCallback((id: string, options: SubscriptionCancelOptions) => {
    return new Promise((resolve, reject) => {
      cancelMutation.mutate({id, options}, {
        onSuccess: (data) => resolve(data),
        onError: (error: Error) => reject((error as FetchError).body),
      });
    });
  }, [cancelMutation]);

  const contextValue: SubscriptionContextType = {
    subscription: subscriptionData,
    isLoading,
    error,
    refreshData,
    pause,
    resume,
    cancel
  };

  return (
    <SubscriptionContext.Provider value={contextValue}>
      {children}
    </SubscriptionContext.Provider>
  );
}

export function useSubscription() {
  const context = useContext(SubscriptionContext);
  if (context === undefined) {
    throw new Error("useSubscription must be used within a SubscriptionProvider");
  }
  return context;
}
