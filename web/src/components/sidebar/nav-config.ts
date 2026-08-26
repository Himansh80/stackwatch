/*
 * nav-config.ts — single source of truth for sidebar + mobile drawer.
 * Drives SidebarItem.tsx, SidebarSection.tsx, and (later) command palette.
 *
 * Adding/removing a route?
 *   1. Add/remove entry here
 *   2. Add/remove the route in App.tsx
 *   Both files share the same path constants — keep in sync.
 */

export interface NavItem {
  path: string;
  label: string;
  icon: string; // SVG icon name from components/icons.tsx
  external?: boolean; // opens in new tab (e.g. docs)
}

export interface NavSection {
  label: string; // section heading, uppercase
  items: NavItem[];
}

export const NAV_SECTIONS: NavSection[] = [
  {
    label: 'Workspace',
    items: [
      { path: '/dashboard', label: 'Overview', icon: 'home' },
      { path: '/billing', label: 'Billing', icon: 'credit-card' },
      { path: '/homelab', label: 'Homelab', icon: 'home' },
      { path: '/profile', label: 'Profile', icon: 'user' },
      { path: '/settings', label: 'Settings', icon: 'cog' },
    ],
  },
  {
    label: 'Infrastructure',
    items: [
      { path: '/proxmox-vms', label: 'Proxmox', icon: 'server' },
      { path: '/truenas', label: 'TrueNAS', icon: 'database' },
    ],
  },
  {
    label: 'Observability',
    items: [
      { path: '/apm', label: 'APM', icon: 'activity' },
      { path: '/logs', label: 'Logs', icon: 'file-text' },
      { path: '/rum', label: 'RUM', icon: 'globe' },
      { path: '/synthetics', label: 'Synthetics', icon: 'check-circle' },
      { path: '/security', label: 'Security', icon: 'shield' },
      { path: '/cspm', label: 'CSPM', icon: 'cloud' },
      { path: '/cicd', label: 'CI/CD', icon: 'git-branch' },
      { path: '/database', label: 'Database', icon: 'database' },
      { path: '/intelligence', label: 'Intelligence', icon: 'sparkles' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { path: '/incidents', label: 'Incidents', icon: 'alert-triangle' },
      { path: '/notebooks', label: 'Notebooks', icon: 'book-open' },
      { path: '/shared', label: 'Collaboration', icon: 'users' },
      { path: '/enterprise', label: 'Enterprise', icon: 'briefcase' },
      { path: '/platform', label: 'Platform', icon: 'layers' },
    ],
  },
];

// Flat list for command palette search (computed once)
export const ALL_NAV_ITEMS: NavItem[] = NAV_SECTIONS.flatMap((s) => s.items);

// Find section for a given path (used for breadcrumb + active section)
export function findSection(path: string): NavSection | undefined {
  return NAV_SECTIONS.find((s) =>
    s.items.some((i) => i.path === path || path.startsWith(i.path + '/'))
  );
}