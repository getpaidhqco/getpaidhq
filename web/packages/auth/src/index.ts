// Main entry point for the auth package - CLIENT SIDE ONLY
// Server exports are available at @getpaidhq/auth/server

// Re-export commonly used types
export type { 
  AuthHeader,
  User,
  AuthHeadersProvider,
  AuthUIProvider,
  LoginComponentProps,
  OrgSwitcherComponentProps,
  UserProfileComponentProps,
  AuthProvider
} from "@getpaidhq/auth-core";

// Re-export auth provider hooks and components
export { 
  FrontendAuthProvider,
  useAuth 
} from "./auth-provider";

// Re-export client components
export {
  LoginComponent,
  UserProfileComponent,
  OrgSwitcherComponent
} from "./client";