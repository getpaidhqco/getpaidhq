"use client"
import {useCallback} from "react";
import {AuthHeadersProvider} from "@getpaidhq/auth-core/types";

// API key implementation
export const useApiKeyAuthTokenProvider = (): AuthHeadersProvider => {
  const getApiKeyToken = useCallback(async () => {
    // Fetch the API key from local storage or any other source
    return "sk_23456789";
  }, []);

  return {
    getAuthHeaders: async () => ({
      "x-api-key": await getApiKeyToken()
    })
  };
};
