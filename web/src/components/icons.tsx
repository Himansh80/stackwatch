/**
 * Shared SVG icon components used across the dashboard.
 *
 * All icons share the same defaults: 18×18, 1.7 stroke width, currentColor,
 * round caps + joins. Color is controlled by parent via CSS.
 *
 * Icons are named after WHAT they depict, not the metric they decorate.
 * Adding an icon? Add it here + reference it in nav-config.ts.
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
    stroke: 'currentColor',
    strokeWidth: 1.7,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  }
}

// ────────────────────────────────────────────────────────────
// Generic UI icons (used by shell, topbar, empty states)
// ────────────────────────────────────────────────────────────

export function HomeIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M3 10 L12 3 L21 10 V20 a1 1 0 0 1 -1 1 H14 V14 H10 V21 H4 a1 1 0 0 1 -1 -1 Z" />
    </svg>
  );
}

export function UserIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21 v-1 a7 7 0 0 1 16 0 v1" />
    </svg>
  );
}

export function CogIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  );
}

export function MenuIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M4 6 H20 M4 12 H20 M4 18 H20" />
    </svg>
  );
}

export function CloseIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M6 6 L18 18 M18 6 L6 18" />
    </svg>
  );
}

export function SearchIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
  );
}

export function BellIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M6 8 a6 6 0 0 1 12 0 c0 7 3 9 3 9 H3 s3-2 3-9" />
      <path d="M10 21 a2 2 0 0 0 4 0" />
    </svg>
  );
}

export function RefreshIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M3 12 a9 9 0 0 1 15-6.7 L21 8" />
      <path d="M21 3 v5 h-5" />
      <path d="M21 12 a9 9 0 0 1-15 6.7 L3 16" />
      <path d="M3 21 v-5 h5" />
    </svg>
  );
}

export function LogOutIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M9 21 H5 a2 2 0 0 1-2-2 V5 a2 2 0 0 1 2-2 h4" />
      <path d="m16 17 5-5-5-5" />
      <path d="M21 12 H9" />
    </svg>
  );
}

// ────────────────────────────────────────────────────────────
// Domain icons (referenced by nav-config.ts)
// ────────────────────────────────────────────────────────────

export function CreditCardIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <rect x="2" y="5" width="20" height="14" rx="2" />
      <path d="M2 10 H22" />
      <path d="M6 15 H10" />
    </svg>
  );
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

export function DatabaseIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <ellipse cx="12" cy="5" rx="9" ry="3" />
      <path d="M3 5 V19 a9 3 0 0 0 18 0 V5" />
      <path d="M3 12 a9 3 0 0 0 18 0" />
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

export function ActivityIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M3 12 H7 L10 4 L14 20 L17 12 H21" />
    </svg>
  );
}

export function FileTextIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M14 3 H6 a2 2 0 0 0-2 2 v14 a2 2 0 0 0 2 2 h12 a2 2 0 0 0 2-2 V9 Z" />
      <path d="M14 3 V9 H20" />
      <path d="M8 13 H16 M8 17 H14" />
    </svg>
  );
}

export function GlobeIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12 H21" />
      <path d="M12 3 a14 14 0 0 1 0 18 a14 14 0 0 1 0-18" />
    </svg>
  );
}

export function CheckCircleIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M22 11.1 V12 a10 10 0 1 1-5.9-9.1" />
      <path d="m22 4-10 10.01-3-3" />
    </svg>
  );
}

export function ShieldIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M12 3 L20 6 V12 c0 5-3.5 8-8 9-4.5-1-8-4-8-9 V6 Z" />
    </svg>
  );
}

export function CloudIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M17.5 19 a4.5 4.5 0 0 0 0-9 6 6 0 0 0-11.7-1 A4.5 4.5 0 0 0 6.5 19 Z" />
    </svg>
  );
}

export function GitBranchIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M6 3 V21" />
      <circle cx="6" cy="3" r="2" />
      <circle cx="6" cy="21" r="2" />
      <path d="M18 9 a4 4 0 0 0-4-4 H8" />
      <circle cx="18" cy="9" r="2" />
    </svg>
  );
}

export function SparklesIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M12 3 L13.5 9 L19 10.5 L13.5 12 L12 18 L10.5 12 L5 10.5 L10.5 9 Z" />
      <path d="M19 16 L19.7 18 L21.5 18.5 L19.7 19 L19 21 L18.3 19 L16.5 18.5 L18.3 18 Z" />
    </svg>
  );
}

export function AlertTriangleIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M10.3 3.86 1.82 18 a2 2 0 0 0 1.71 3 H20.47 a2 2 0 0 0 1.71-3 L13.71 3.86 a2 2 0 0 0-3.42 0 Z" />
      <path d="M12 9 V13 M12 17 H12.01" />
    </svg>
  );
}

export function BookOpenIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M2 4 H7 a4 4 0 0 1 4 4 V21 a3 3 0 0 0-3-3 H2 Z" />
      <path d="M22 4 H17 a4 4 0 0 0-4 4 V21 a3 3 0 0 1 3-3 H22 Z" />
    </svg>
  );
}

export function UsersIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="M16 21 v-2 a4 4 0 0 0-4-4 H6 a4 4 0 0 0-4 4 v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21 v-2 a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13 a4 4 0 0 1 0 7.75" />
    </svg>
  );
}

export function BriefcaseIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <rect x="2" y="7" width="20" height="14" rx="2" />
      <path d="M16 21 V5 a2 2 0 0 0-2-2 H10 a2 2 0 0 0-2 2 V21" />
    </svg>
  );
}

export function LayersIcon(props: IconProps) {
  return (
    <svg {...svgDefaults(props.size)} className={props.className} aria-hidden={props['aria-hidden']}>
      <path d="m12 2-10 5 10 5 10-5 Z" />
      <path d="m2 17 10 5 10-5" />
      <path d="m2 12 10 5 10-5" />
    </svg>
  );
}

// ────────────────────────────────────────────────────────────
// Domain-metric icons (used inside cards)
// ────────────────────────────────────────────────────────────

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

/**
 * Icon registry — maps the string names in nav-config.ts to components.
 * Adding an icon? Add it to this map.
 */
export const ICONS = {
  home: HomeIcon,
  user: UserIcon,
  cog: CogIcon,
  menu: MenuIcon,
  close: CloseIcon,
  search: SearchIcon,
  bell: BellIcon,
  refresh: RefreshIcon,
  'log-out': LogOutIcon,
  'credit-card': CreditCardIcon,
  server: ServerIcon,
  database: DatabaseIcon,
  network: NetworkIcon,
  activity: ActivityIcon,
  'file-text': FileTextIcon,
  globe: GlobeIcon,
  'check-circle': CheckCircleIcon,
  shield: ShieldIcon,
  cloud: CloudIcon,
  'git-branch': GitBranchIcon,
  sparkles: SparklesIcon,
  'alert-triangle': AlertTriangleIcon,
  'book-open': BookOpenIcon,
  users: UsersIcon,
  briefcase: BriefcaseIcon,
  layers: LayersIcon,
  play: PlayIcon,
  heart: HeartIcon,
} as const;

export type IconName = keyof typeof ICONS;

export function getIcon(name: string) {
  return ICONS[name as IconName] ?? ServerIcon;
}