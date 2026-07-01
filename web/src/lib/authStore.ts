import { create } from "zustand";
import type { TokenPair, User } from "../types/api";

const STORAGE_KEY = "sky-panel:auth";

interface StoredAuth {
  accessToken: string;
  refreshToken: string;
  user: User;
}

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  setSession: (tokens: TokenPair) => void;
  updateUser: (user: User) => void;
  clear: () => void;
}

function loadStored(): StoredAuth | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function persist(auth: StoredAuth | null) {
  if (auth) localStorage.setItem(STORAGE_KEY, JSON.stringify(auth));
  else localStorage.removeItem(STORAGE_KEY);
}

const initial = loadStored();

export const useAuthStore = create<AuthState>((set, get) => ({
  accessToken: initial?.accessToken ?? null,
  refreshToken: initial?.refreshToken ?? null,
  user: initial?.user ?? null,

  setSession: (tokens) => {
    set({ accessToken: tokens.access_token, refreshToken: tokens.refresh_token, user: tokens.user });
    persist({ accessToken: tokens.access_token, refreshToken: tokens.refresh_token, user: tokens.user });
  },

  updateUser: (user) => {
    set({ user });
    const { accessToken, refreshToken } = get();
    if (accessToken && refreshToken) {
      persist({ accessToken, refreshToken, user });
    }
  },

  clear: () => {
    set({ accessToken: null, refreshToken: null, user: null });
    persist(null);
  },
}));
