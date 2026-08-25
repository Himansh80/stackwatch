/**
 * Tier 13 Phase 3 — Auth context.
 * Holds the JWT in memory + AsyncStorage (when available).
 * On login: stores token, sets Authorization header on apiFetch.
 * On logout: clears token, navigates to Login.
 */
import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { mobile, setAuthToken } from '../lib/api';

export interface User {
  id: string;
  email: string;
  full_name: string;
}

export interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthCtx = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setTokenState] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const login = useCallback(async (email: string, password: string) => {
    setLoading(true);
    try {
      const resp = await mobile.login(email, password);
      setToken(resp.token);
      setAuthToken(resp.token);
      setTokenState(resp.token);
      setUser(resp.user);
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    setAuthToken(null);
    setTokenState(null);
    setUser(null);
  }, []);

  // On unmount clear auth header.
  useEffect(() => () => setAuthToken(null), []);

  return (
    <AuthCtx.Provider value={{ user, token, loading, login, logout }}>
      {children}
    </AuthCtx.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
