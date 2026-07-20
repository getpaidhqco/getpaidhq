import {AuthHeadersProvider} from "./types";

// Main hook to use the appropriate auth token provider
const useAuthToken = (): AuthHeadersProvider => {
  // Dynamically import the appropriate auth token provider based on the environment variable
  switch (process.env.NEXT_PUBLIC_AUTH_PROVIDER) {
    case 'clerk':
      // We need to dynamically import to avoid circular dependencies
      // and to ensure Clerk's hooks are only used within ClerkProvider
      const { useClerkAuthTokenProvider } = require("@getpaidhq/auth-clerk/hooks");
      return useClerkAuthTokenProvider();
    case 'apiKey':
      const { useApiKeyAuthTokenProvider } = require("@getpaidhq/auth-apikey/hooks");
      return useApiKeyAuthTokenProvider();
    default:
      throw new Error('Unknown auth provider');
  }
};

export default useAuthToken;
