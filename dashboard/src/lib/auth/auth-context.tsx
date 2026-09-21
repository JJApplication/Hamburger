"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

import { login as loginRequest, logout as logoutRequest, fetchCurrentUser, type AuthUser } from "@/lib/api/gateway";

interface AuthContextValue {
  user: AuthUser | null;
  token: string | null;
  ready: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);
const TOKEN_KEY = "hamburger.dashboard.token";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const expire = () => {
      window.sessionStorage.removeItem(TOKEN_KEY);
      setToken(null);
      setUser(null);
    };
    window.addEventListener("hamburger-auth-expired", expire);
    const stored = window.sessionStorage.getItem(TOKEN_KEY);
    if (!stored) { setReady(true); return () => window.removeEventListener("hamburger-auth-expired", expire); }
    setToken(stored);
    void fetchCurrentUser(stored)
      .then((value) => setUser(value))
      .catch(expire)
      .finally(() => setReady(true));
    return () => window.removeEventListener("hamburger-auth-expired", expire);
  }, []);

  const value = useMemo<AuthContextValue>(() => ({
    user,
    token,
    ready,
    login: async (username, password) => {
      const result = await loginRequest(username, password);
      window.sessionStorage.setItem(TOKEN_KEY, result.token);
      setToken(result.token);
      setUser(result.user);
    },
    logout: async () => {
      const active = token;
      try { if (active) await logoutRequest(active); } finally {
        window.sessionStorage.removeItem(TOKEN_KEY);
        setToken(null);
        setUser(null);
      }
    },
  }), [ready, token, user]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used within AuthProvider");
  return value;
}

export { TOKEN_KEY };
