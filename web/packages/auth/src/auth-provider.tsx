"use client"
import React, {ReactNode} from 'react';
import {ClerkAuthProvider, useClerkContextAuth} from '@getpaidhq/auth-clerk/provider';
import {ApiKeyAuthProvider} from '@getpaidhq/auth-apikey/provider';

// You can select based on env vars, config files, etc.
const provider = process.env.NEXT_PUBLIC_AUTH_PROVIDER as string;

// Dynamic auth provider wrapper
export const FrontendAuthProvider = ({children}: { children: ReactNode }) => {

  switch (provider) {
    case 'clerk':
      return <ClerkAuthProvider>{children}</ClerkAuthProvider>;
    case 'apiKey':
      return <ApiKeyAuthProvider>{children}</ApiKeyAuthProvider>;
    default:
      console.warn(`Unknown auth provider: ${provider}, falling back to base provider`);
      return <ApiKeyAuthProvider>{children}</ApiKeyAuthProvider>;
  }
};

export const useAuth = () => {
  // Use the appropriate context based on the provider
  switch (provider) {
    case 'clerk': {
      // Dynamically use the Clerk auth context
      return useClerkContextAuth();
    }
    case 'apiKey': {
      // Dynamically use the API Key auth context
      const {useApiKeyAuth} = require('@getpaidhq/auth-apikey/provider');
      return useApiKeyAuth();
    }
    default:
      throw new Error(`Unknown auth provider: ${provider}`);
  }
};