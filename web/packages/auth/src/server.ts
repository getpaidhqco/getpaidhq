import { AuthProvider } from "@getpaidhq/auth-core";
import clerk from "@getpaidhq/auth-clerk/server";
import apikey from "@getpaidhq/auth-apikey/server";

// Utility function to load the appropriate auth provider
export const loadAuthProvider = (): AuthProvider => {
  const providerName = process.env.AUTH_PROVIDER ?? 'apiKey';

  switch (providerName) {
    case 'clerk':
      return clerk;
    case 'apiKey':
      return apikey;
    default:
      throw new Error(`Unknown auth provider: ${providerName}`);
  }
};

// Re-export the AuthProvider interface for convenience
export type { AuthProvider } from "@getpaidhq/auth-core";