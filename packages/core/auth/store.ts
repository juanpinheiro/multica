import { create } from "zustand";
import type { User } from "../types";
import type { ApiClient } from "../api/client";

export interface AuthStoreOptions {
  api: ApiClient;
}

export interface AuthState {
  user: User | null;
  isLoading: boolean;

  initialize: () => Promise<void>;
  setUser: (user: User) => void;
  refreshMe: () => Promise<void>;
}

export function createAuthStore(options: AuthStoreOptions) {
  const { api } = options;

  return create<AuthState>((set) => ({
    user: null,
    isLoading: true,

    initialize: async () => {
      try {
        const user = await api.getMe();
        set({ user, isLoading: false });
      } catch {
        set({ user: null, isLoading: false });
      }
    },

    setUser: (user: User) => {
      set({ user });
    },

    refreshMe: async () => {
      const user = await api.getMe();
      set({ user });
    },
  }));
}
