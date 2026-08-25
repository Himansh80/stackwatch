/**
 * Tier 13 Phase 3 — Theme tokens (Datadog-style dark palette).
 * Single source of truth for colors + spacing + typography.
 */

export const colors = {
  bg: '#0f172a',            // slate-900
  surface: '#1e293b',        // slate-800
  surfaceAlt: '#334155',     // slate-700
  border: '#475569',         // slate-600
  text: '#f1f5f9',           // slate-100
  textMuted: '#94a3b8',      // slate-400
  primary: '#3b82f6',        // blue-500
  primaryHover: '#2563eb',   // blue-600
  success: '#10b981',        // emerald-500
  warning: '#f59e0b',        // amber-500
  danger: '#ef4444',         // red-500
  info: '#6366f1',           // indigo-500
  up: '#10b981',
  down: '#ef4444',
  stale: '#f59e0b',
  unknown: '#94a3b8',
} as const;

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
} as const;

export const radius = {
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
} as const;

export const typography = {
  h1: { fontSize: 28, fontWeight: '700' as const },
  h2: { fontSize: 20, fontWeight: '600' as const },
  h3: { fontSize: 16, fontWeight: '600' as const },
  body: { fontSize: 14, fontWeight: '400' as const },
  caption: { fontSize: 12, fontWeight: '400' as const, color: colors.textMuted },
  metric: { fontSize: 32, fontWeight: '700' as const },
} as const;

export type Theme = {
  colors: typeof colors;
  spacing: typeof spacing;
  radius: typeof radius;
  typography: typeof typography;
};

export const darkTheme: Theme = { colors, spacing, radius, typography };
