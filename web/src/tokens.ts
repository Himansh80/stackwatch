// Design tokens — TypeScript mirror of pcss/tokens.css for type safety.
// Always import from here in .tsx files instead of hardcoding hex codes or sizes.

export const color = {
  bg: 'var(--color-bg)',
  bgElevated: 'var(--color-bg-elevated)',
  surface: 'var(--color-surface)',
  surfaceElevated: 'var(--color-surface-elevated)',
  surfaceHover: 'var(--color-surface-hover)',

  textPrimary: 'var(--color-text)',
  textMuted: 'var(--color-text-muted)',
  textSubtle: 'var(--color-text-subtle)',
  textDisabled: 'var(--color-text-disabled)',
  textInverse: 'var(--color-text-inverse)',
  textLink: 'var(--color-text-link)',

  // Brand
  primary: 'var(--color-primary)',
  primaryHover: 'var(--color-primary-hover)',
  primaryActive: 'var(--color-primary-active)',
  primaryMuted: 'var(--color-primary-muted)',

  // Semantic states
  success: 'var(--color-success)',
  successHover: 'var(--color-success-hover)',
  successMuted: 'var(--color-success-muted)',
  warning: 'var(--color-warning)',
  warningHover: 'var(--color-warning-hover)',
  warningMuted: 'var(--color-warning-muted)',
  error: 'var(--color-error)',
  errorHover: 'var(--color-error-hover)',
  errorMuted: 'var(--color-error-muted)',
  info: 'var(--color-info)',
  infoMuted: 'var(--color-info-muted)',

  // Border
  border: 'var(--color-border)',
  borderStrong: 'var(--color-border-strong)',
  borderFocus: 'var(--color-border-focus)',
};

export const font = {
  family: 'var(--font-family)',
  mono: 'var(--font-family-mono)',
  sm: 'var(--font-size-sm)',
  base: 'var(--font-size-md)',
  lg: 'var(--font-size-lg)',
  xl: 'var(--font-size-xl)',
  '2xl': 'var(--font-size-2xl)',
};

export const spacing = {
  xs: 'var(--space-xs)',
  sm: 'var(--space-sm)',
  md: 'var(--space-md)',
  lg: 'var(--space-lg)',
  xl: 'var(--space-xl)',
  '2xl': 'var(--space-2xl)',
  '3xl': 'var(--space-3xl)',
};

export const radius = {
  sm: 'var(--radius-sm)',
  md: 'var(--radius-md)',
  lg: 'var(--radius-lg)',
  pill: 'var(--radius-pill)',
  full: 'var(--radius-full)',
};

export const shadow = {
  sm: 'var(--shadow-sm)',
  md: 'var(--shadow-md)',
  lg: 'var(--shadow-lg)',
  xl: 'var(--shadow-xl)',
};
