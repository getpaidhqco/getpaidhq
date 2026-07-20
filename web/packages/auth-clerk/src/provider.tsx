"use client";

import React, {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import {
  ClerkProvider,
  useAuth as useClerkAuth,
  useOrganizationList,
  useSession,
  useUser,
} from "@clerk/nextjs";
import { AuthHeader, User } from "@getpaidhq/auth-core/types";
import { useClerkAuthTokenProvider } from "./hooks";

interface AuthContextType {
  orgId?: string;
  isAuthenticated: boolean;
  setActiveOrg: (id: string) => Promise<unknown>;
  reloadSession: () => Promise<unknown>;
  getToken: () => Promise<string | null>;
  getAuthHeaders: () => Promise<AuthHeader>;
  login: () => void;
  logout: () => Promise<void>;
  currentUser: User;
}

const ClerkAuthContext = createContext<AuthContextType | undefined>(undefined);

function toFullName(firstName?: string | null, lastName?: string | null): string {
  const fn = firstName ?? "";
  const ln = lastName ?? "";
  return [fn, ln].filter(Boolean).join(" ");
}

const ClerkAuthInnerProvider = ({ children }: { children: ReactNode }) => {
  const { isSignedIn, signOut, orgId } = useClerkAuth();
  const { session } = useSession();
  const { setActive } = useOrganizationList();
  const { user } = useUser();

  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    setIsAuthenticated(!!isSignedIn);
  }, [isSignedIn]);

  const login = useCallback(() => {
    window.location.href = "/sign-in";
  }, []);

  const logout = useCallback(async () => {
    if (signOut) {
      await signOut();
      setIsAuthenticated(false);
    }
  }, [signOut]);

  const reloadSession = useCallback(async () => {
    return session?.reload();
  }, [session]);

  const setActiveOrg = useCallback(
    async (id: string) => {
      return setActive?.({ organization: id });
    },
    [setActive],
  );

  const { getAuthHeaders, getToken } = useClerkAuthTokenProvider();

  const currentUser: User = {
    id: user?.id ?? "",
    orgId: orgId ?? undefined,
    email: user?.emailAddresses?.[0]?.emailAddress ?? undefined,
    name: toFullName(user?.firstName, user?.lastName),
    avatar: user?.imageUrl ?? undefined,
  };

  return (
    <ClerkAuthContext.Provider
      value={{
        orgId: orgId ?? undefined,
        isAuthenticated,
        reloadSession,
        setActiveOrg,
        getToken,
        login,
        getAuthHeaders,
        logout,
        currentUser,
      }}
    >
      {children}
    </ClerkAuthContext.Provider>
  );
};

export const ClerkAuthProvider = ({ children }: { children: ReactNode }) => {
  return (
    <ClerkProvider>
      <ClerkAuthInnerProvider>{children}</ClerkAuthInnerProvider>
    </ClerkProvider>
  );
};

export const useClerkContextAuth = () => {
  const context = useContext(ClerkAuthContext);
  if (context === undefined) {
    throw new Error("useClerkContextAuth must be used within a ClerkAuthProvider");
  }
  return context;
};

export { ClerkAuthContext };
