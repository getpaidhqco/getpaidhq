export type AuthHeader = Record<string, string>;

// Define an interface for authenticated users
export interface User {
  id: string;
  // Add other common user properties that all auth providers should populate
  orgId?: string;
  email?: string;
  name?: string;
  avatar?: string;
}

// Define an interface for the auth token provider
export interface AuthHeadersProvider {
  getToken: () => Promise<string | null>;
  getAuthHeaders: () => Promise<AuthHeader>;
}

// Define an interface for the login component
export interface LoginComponentProps {
  // Add any props that might be needed for the login component
  afterSignInUrl?: string
}

// Define an interface for the login component
export interface OrgSwitcherComponentProps {
  // Add any props that might be needed for the login component
  afterChangeUrl?: string
}

// Define an interface for the user profile component
export interface UserProfileComponentProps {
  // Add any props that might be needed for the user profile component
  appearance?: Record<string, unknown>;
}

// Define an interface for the auth UI provider
export interface AuthUIProvider {
  LoginComponent: React.ComponentType<LoginComponentProps>;
}
