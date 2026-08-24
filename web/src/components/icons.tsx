/**
 * Shared SVG icon components used across the dashboard.
 *
 * Why a dedicated file: the dashboard's MetricCard used inline SVG
 * strings hard-coded in JSX. Centralising them here:
 *   - gives every icon the same `currentColor` + `strokeWidth` defaults,
 *   - lets the parent control color via CSS (`color: var(--accent)`),
 *   - keeps icon dimensions consistent (default 18×18, override via prop).
 *
 * Icons are named after WHAT they depict (server, network) rather than
 * the metric they currently decorate. The MetricCard does the mapping.
 */

interface IconProps {
  size?: number;
  className?: string;
  'aria-hidden'?: boolean;
}

function svgDefaults(size = 18) {
  return {
    width: size,
    height: size,
    viewBox: '0 0 24 24',
    fill: 'none',
    strokeWidth: 1.7,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  };
}

export function ServerIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <rect x="3" y="4" width="18" height="6" rx="1.5" />
      <rect x="3" y="14" width="18" height="6" rx="1.5" />
      <circle cx="7" cy="7" r="0.8" fill="currentColor" stroke="none" />
      <circle cx="7" cy="17" r="0.8" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function NetworkIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M12 3 L21 8 L21 16 L12 21 L3 16 L3 8 Z" />
      <path d="M12 12 L21 8 M12 12 L3 8 M12 12 L12 21" />
    </svg>
  );
}

export function PlayIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <polygon points="6,4 20,12 6,20" />
    </svg>
  );
}

export function HeartIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78L12 21l8.84-8.61a5.5 5.5 0 0 0 0-7.78z" />
    </svg>
  );
}
