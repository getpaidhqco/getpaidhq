import { AuthProvider } from "@getpaidhq/auth-core/server";
import { User } from "@getpaidhq/auth-core/types";
import {
  auth as clerkAuth,
  clerkMiddleware,
  createRouteMatcher,
  currentUser,
} from "@clerk/nextjs/server";

const isPublicRoute = createRouteMatcher(["/sign-in(.*)"]);

function toFullName(firstName?: string | null, lastName?: string | null): string {
  const fn = firstName ?? "";
  const ln = lastName ?? "";
  return [fn, ln].filter(Boolean).join(" ");
}

const clerkAuthProvider: AuthProvider = {
  getMiddleware: clerkMiddleware(async (auth, request) => {
    if (!isPublicRoute(request)) {
      await auth.protect();
    }
  }),

  currentUser: async (): Promise<User> => {
    const { userId, orgId } = await clerkAuth();
    const user = await currentUser();

    if (!user || !userId) {
      throw new Error("not authenticated");
    }

    return {
      id: userId,
      orgId: orgId ?? undefined,
      email: user.emailAddresses?.[0]?.emailAddress ?? undefined,
      name: toFullName(user.firstName, user.lastName),
      avatar: user.imageUrl ?? undefined,
    };
  },

  auth: async (): Promise<User> => {
    const { userId, orgId } = await clerkAuth();
    const user = await currentUser();

    return {
      id: userId ?? "",
      orgId: orgId ?? undefined,
      email: user?.emailAddresses?.[0]?.emailAddress ?? undefined,
      name: toFullName(user?.firstName, user?.lastName),
      avatar: user?.imageUrl ?? undefined,
    };
  },

  getAuthHeader: async () => {
    const { getToken } = await clerkAuth();
    const token = await getToken();
    const header: Record<string, string> = {};
    if (token) header.Authorization = `Bearer ${token}`;
    return header;
  },

  getToken: async () => {
    const { getToken } = await clerkAuth();
    return getToken();
  },
};

export default clerkAuthProvider;
