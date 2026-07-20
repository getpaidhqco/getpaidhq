"use client"

import React, {createContext, ReactNode, useContext, useState} from 'react';
import {useApiKeyAuthTokenProvider} from './hooks';
import {User} from '@getpaidhq/auth-core/types';

// Define the auth context type
interface AuthContextType {
  isAuthenticated: boolean;
  getAuthHeaders: any;
  login: () => void;
  logout: () => void;
  currentUser: User;
}

// Create a context for API Key authentication
const ApiKeyAuthContext = createContext<AuthContextType | undefined>(undefined);

// API Key Auth Provider component
export const ApiKeyAuthProvider = ({children}: { children: ReactNode }) => {
  // Check if there's an API key in localStorage to determine initial auth state
  const [isAuthenticated, setIsAuthenticated] = useState(() => {
    if (typeof window !== 'undefined') {
      return !!localStorage.getItem('apiKey');
    }
    return false;
  });

  // Login function - in a real implementation, this might validate the API key
  const login = () => setIsAuthenticated(true);

  // Logout function - removes the API key from localStorage
  const logout = () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('apiKey');
    }
    setIsAuthenticated(false);
  };

  const {getAuthHeaders} = useApiKeyAuthTokenProvider();

  // Create empty currentUser object for API Key auth
  const currentUser: User = {
    id: '',
    orgId: '',
    email: '',
    name: '',
    avatar: '',
  };

  return (
    <ApiKeyAuthContext.Provider value={{
      isAuthenticated,
      login,
      getAuthHeaders,
      logout,
      currentUser
    }}>
      {children}
    </ApiKeyAuthContext.Provider>
  );
};

// Hook to use the API Key auth context
export const useApiKeyAuth = () => {
  const context = useContext(ApiKeyAuthContext);
  if (context === undefined) {
    throw new Error('useApiKeyAuth must be used within an ApiKeyAuthProvider');
  }
  return context;
};

// Export the context for potential direct access
export {ApiKeyAuthContext};
