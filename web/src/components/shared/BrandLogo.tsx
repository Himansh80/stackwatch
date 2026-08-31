/**
 * BrandLogo.tsx — shared SVG brand logo.
 *
 * Tier 20: replaces letter "S" mark in sidebar with proper SVG (U3).
 *
 * Visual contract:
 *   - Inline SVG (no external file fetch)
 *   - Uses currentColor for fill (inherits from parent text color)
 *   - 2 variants: 'full' (mark + wordmark) and 'mark' (mark only)
 *   - Sizes via the `size` prop (default 32)
 */
import { CSSProperties } from 'react';

export type BrandVariant = 'full' | 'mark';

export interface BrandLogoProps {
  variant?: BrandVariant;
  size?: number;
  className?: string;
  style?: CSSProperties;
  title?: string;
}

/**
 * BrandLogo — the StackWatch brand mark + wordmark.
 *
 * Design: stacked rectangular blocks (Datadog-style) with subtle gradient,
 * "S" monogram inside the mark. Cyan (var(--color-primary)) accent.
 */
export default function BrandLogo({
  variant = 'mark',
  size = 32,
  className = '',
  style,
  title = 'StackWatch',
}: BrandLogoProps) {
  const markSize = size;
  const classes = ['brand-logo', `brand-logo-${variant}`, className]
    .filter(Boolean)
    .join(' ');

  // Mark: stacked rounded squares forming an S-shape
  const mark = (
    <svg
      width={markSize}
      height={markSize}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden={variant === 'mark' ? 'true' : undefined}
      role={variant === 'mark' ? 'img' : undefined}
    >
      <title>{variant === 'mark' ? title : undefined}</title>
      <rect x="2" y="2" width="20" height="12" rx="3" fill="currentColor" opacity="0.9" />
      <rect x="10" y="14" width="20" height="12" rx="3" fill="currentColor" opacity="0.55" />
    </svg>
  );

  if (variant === 'mark') {
    return (
      <span className={classes} style={{ color: 'var(--color-primary)', ...style }}>
        {mark}
      </span>
    );
  }

  // Full: mark + wordmark
  return (
    <span className={classes} style={style}>
      {mark}
      <span className="brand-logo-wordmark">
        <strong>StackWatch</strong>
        <small>Infrastructure control plane</small>
      </span>
    </span>
  );
}