"use client"
import React, {ReactNode} from 'react';
import {QueryClient, QueryClientProvider} from '@tanstack/react-query';

export interface GetPaidHQProviderProps {
  children: ReactNode;
  client?: QueryClient;
  apiKey?: string;
  getToken?: () => Promise<string | null>;
  baseUrl?: string;
}

/**
 * GetPaidHQProvider sets up the QueryClient and provides it to the application.
 * It also configures the GetPaidHQ SDK with the provided API key and base URL.
 */
export function GetPaidHQProvider({
                                    children,
                                    client,
                                    apiKey,
                                    getToken,
                                    baseUrl = 'https://api.getpaidhq.co',
                                  }: GetPaidHQProviderProps) {
  // Create a client if one is not provided
  const queryClient = client ?? new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5, // 5 minutes
        refetchOnWindowFocus: false,
        retry: 1,
      },
    },
  });



  // Store API configuration in React context for hooks to access
  const contextValue = React.useMemo(() => ({
    apiKey,
    getToken,
    baseUrl,
  }), [apiKey, getToken, baseUrl]);

  return (
    <GetPaidHQContext.Provider value={contextValue}>
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    </GetPaidHQContext.Provider>
  );
}

// Create a context to store API configuration
interface GetPaidHQContextValue {
  getToken?: () => Promise<string | null>;
  apiKey?: string;
  baseUrl: string;
}

const GetPaidHQContext = React.createContext<GetPaidHQContextValue | undefined>(undefined);

/**
 * Hook to access the GetPaidHQ context
 */
export function useGetPaidHQContext() {
  const context = React.useContext(GetPaidHQContext);
  if (context === undefined) {
    throw new Error('useGetPaidHQContext must be used within a GetPaidHQProvider');
  }
  return context;
}
