/**
 * SkeletonCard.tsx — shared card-shaped loading skeleton.
 *
 * Tier 20: addresses loading_state drifts. Used wherever a card-shaped
 * placeholder is needed while data fetches.
 *
 * Visual contract (from spec §3.8):
 *   - 3 variants: kpi (label + value blocks), panel (single block),
 *     list (3 horizontal blocks)
 *   - Pulse animation via CSS @keyframes (1.5s loop)
 *   - Tokens: --color-surface ↔ --color-surface-hover
 *
 * Accessibility:
 *   - role="status", aria-live="polite" (announced by screen readers)
 *   - aria-hidden on the visual blocks (decorative only)
 */
import { CSSProperties } from 'react';

export type SkeletonCardVariant = 'kpi' | 'panel' | 'list';

export interface SkeletonCardProps {
  variant?: SkeletonCardVariant;
  width?: string | number;
  height?: string | number;
  className?: string;
  style?: CSSProperties;
}

const DEFAULT_DIMENSIONS: Record<SkeletonCardVariant, { width: number; height: number }> = {
  kpi: { width: 240, height: 120 },
  panel: { width: 320, height: 160 },
  list: { width: 480, height: 56 },
};

export default function SkeletonCard({
  variant = 'panel',
  width,
  height,
  className = '',
  style,
}: SkeletonCardProps) {
  const dims = DEFAULT_DIMENSIONS[variant];
  const finalWidth = width ?? dims.width;
  const finalHeight = height ?? dims.height;

  const wrapperStyle: CSSProperties = {
    width: typeof finalWidth === 'number' ? `${finalWidth}px` : finalWidth,
    height: typeof finalHeight === 'number' ? `${finalHeight}px` : finalHeight,
    ...style,
  };

  const classes = [
    'skeleton-card',
    `skeleton-card-${variant}`,
    className,
  ]
    .filter(Boolean)
    .join(' ');

  if (variant === 'kpi') {
    return (
      <div
        className={classes}
        style={wrapperStyle}
        role="status"
        aria-live="polite"
        aria-label="Loading KPI"
      >
        <div className="skeleton-card-eyebrow" aria-hidden="true" />
        <div className="skeleton-card-value" aria-hidden="true" />
      </div>
    );
  }

  if (variant === 'list') {
    return (
      <div
        className={classes}
        style={wrapperStyle}
        role="status"
        aria-live="polite"
        aria-label="Loading list"
      >
        <div className="skeleton-card-list-block" aria-hidden="true" />
        <div className="skeleton-card-list-block" aria-hidden="true" />
        <div className="skeleton-card-list-block" aria-hidden="true" />
      </div>
    );
  }

  // panel (default)
  return (
    <div
      className={classes}
      style={wrapperStyle}
      role="status"
      aria-live="polite"
      aria-label="Loading"
    >
      <div className="skeleton-card-block" aria-hidden="true" />
    </div>
  );
}