import { AuthProvider } from "@getpaidhq/auth-core/server";
import { User } from "@getpaidhq/auth-core/types";

const apiKeyAuthProvider: AuthProvider = {
  auth: async (): Promise<User> => {
    return {
      id: "api-user",
    };
  },
  currentUser: async (): Promise<User> => {
    return {
      id: "api-user",
    };
  },
  getAuthHeader: async () => {
    return { "x-api-key": "sk_23456789" };
  },
  getToken: async () => {
    return "sk_23456789";
  },
};

export default apiKeyAuthProvider;
