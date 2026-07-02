import { create } from "zustand";
import type { TokenPair, User } from "../types/api";

const STORAGE_KEY = "sky-panel:auth";
// When an admin "views as" a user, their own session is stashed here so it can
// be restored on exit. Its presence is what marks the session as impersonating.
const ADMIN_BACKUP_KEY = "sky-panel:admin-session";

interface StoredAuth {
  accessToken: string;
  refreshToken: string;
  user: User;
}

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  impersonating: boolean;
  setSession: (tokens: TokenPair) => void;
  updateUser: (user: User) => void;
  beginImpersonation: (tokens: TokenPair) => void;
  endImpersonation: () => void;
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

function loadBackup(): StoredAuth | null {
  try {
    const raw = localStorage.getItem(ADMIN_BACKUP_KEY);
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
  impersonating: loadBackup() !== null,

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

  // Switch into a target user's session, stashing the admin's own session so it
  // can be restored. Only backs up if not already impersonating (so nested
  // "view as" can't clobber the real admin session).
  beginImpersonation: (tokens) => {
    const { accessToken, refreshToken, user, impersonating } = get();
    if (!impersonating && accessToken && refreshToken && user) {
      try {
        localStorage.setItem(ADMIN_BACKUP_KEY, JSON.stringify({ accessToken, refreshToken, user }));
      } catch {
        // if we can't persist the backup, don't strand the admin
        return;
      }
    }
    set({ accessToken: tokens.access_token, refreshToken: tokens.refresh_token, user: tokens.user, impersonating: true });
    persist({ accessToken: tokens.access_token, refreshToken: tokens.refresh_token, user: tokens.user });
  },

  endImpersonation: () => {
    const backup = loadBackup();
    localStorage.removeItem(ADMIN_BACKUP_KEY);
    if (backup) {
      set({ accessToken: backup.accessToken, refreshToken: backup.refreshToken, user: backup.user, impersonating: false });
      persist(backup);
    } else {
      set({ impersonating: false });
    }
  },

  clear: () => {
    localStorage.removeItem(ADMIN_BACKUP_KEY);
    set({ accessToken: null, refreshToken: null, user: null, impersonating: false });
    persist(null);
  },
}));
