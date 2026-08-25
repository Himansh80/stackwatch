/**
 * Tier 13 Phase 3 — Theme context.
 * For Phase 1: dark theme only (system preference would be Phase 2).
 */
import React, { createContext, useContext } from 'react';
import { darkTheme, Theme } from '../lib/theme';

const ThemeCtx = createContext<Theme | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return <ThemeCtx.Provider value={darkTheme}>{children}</ThemeCtx.Provider>;
}

export function useTheme(): Theme {
  const ctx = useContext(ThemeCtx);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}
