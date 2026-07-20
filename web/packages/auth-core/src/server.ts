import { AuthHeader, User } from "./types";

/**
 * AuthProvider abstracts the server-side auth implementation (Clerk, API key, …).
 *
 * - `auth(req)`: resolves the current user for the incoming request. Used by
 *   the proxy.ts fallback and by RSC code that needs the user without throwing
 *   on missing session.
 * - `currentUser()`: same as `auth` but contractually throws if not signed in.
 * - `getAuthHeader()` / `getToken()`: lift the session credential out for
 *   forwarding to backend APIs.
 * - `getMiddleware`: an optional, fully-formed Next 16 proxy function
 *   (returned by e.g. `clerkMiddleware(...)`). When present, the app's
 *   `proxy.ts` delegates to it directly instead of running its own fallback.
 */
export interface AuthProvider {
  currentUser: () => Promise<User>;
  auth: (req?: unknown) => Promise<User>;
  getAuthHeader: () => Promise<AuthHeader>;
  getToken: () => Promise<string | null>;
  /**
   * Next 16 proxy function — loosely typed so each provider can plug in the
   * function returned by their own SDK (eg. `clerkMiddleware(...)` from
   * `@clerk/nextjs/server`) without leaking that SDK's types here.
   */
  getMiddleware?: (req: any, event?: any) => any;
}
