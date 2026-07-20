"use client"
import {useCallback} from "react";
import {AuthHeadersProvider} from "@getpaidhq/auth-core/types";
import {useAuth} from "@clerk/nextjs"


export const useClerkAuthTokenProvider = (): AuthHeadersProvider => {
  const {getToken} = useAuth();

  const getClerkToken = useCallback(async () => {
    try {
      return await getToken();
    } catch (err) {
      console.error('Error fetching Clerk token:', err);
      return null;
    }
  }, [getToken]);

  return {
    getToken,
    getAuthHeaders: async () => ({
      "Authorization": `Bearer ${await getClerkToken()}`
    })
  };
};

