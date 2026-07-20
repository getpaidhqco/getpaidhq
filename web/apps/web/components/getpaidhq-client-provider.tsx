"use client";

import {GetPaidHQProvider} from "@getpaidhq/react-sdk";
import {ReactNode, useCallback} from "react";
import {useAuth} from "@getpaidhq/auth";
import {useQueryClient} from "@tanstack/react-query";

interface GetPaidHQClientProviderProps {
  children: ReactNode;
  apiKey?: string;
  baseUrl?: string;
}

export function GetPaidHQClientProvider({
                                          children,
                                          apiKey,
                                          baseUrl
                                        }: GetPaidHQClientProviderProps) {
  const {getToken} = useAuth()
  // Always hand the SDK a getToken FUNCTION, even during SSR where Clerk's getToken is
  // undefined. The SDK client throws at construction if neither apiKey nor getToken is
  // set, and react-sdk hooks construct the client at render time — so a missing getToken
  // would 500 the server render of any page that mounts an SDK hook. During SSR this
  // resolves to null (no requests fire then); after hydration it returns the real token.
  const safeGetToken = useCallback(
    async () => (getToken ? await getToken() : null),
    [getToken]
  )
  // Share the app's single QueryClient (from QueryProvider) with the SDK provider.
  // Otherwise the SDK creates its own client, so SDK mutation hooks invalidate a
  // different cache than the one DataTable's useQuery reads — and tables never refresh.
  const queryClient = useQueryClient()
  return (
    <GetPaidHQProvider
      client={queryClient}
      getToken={safeGetToken}
      apiKey={apiKey}
      baseUrl={baseUrl}
    >
      {children}
    </GetPaidHQProvider>
  );
}