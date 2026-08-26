/**
 * Tier 14 (Phase 14.4+) — Shared shell wrapper for all Proxmox pages.
 *
 * Provides the px-shell layout: topbar + sidebar (with sectionless
 * nav) + main + page header. Uses ONLY the px-* design system
 * (defined in web/src/styles/proxmox.css) so the new Tier-14 pages
 * match the rest of the new design (Datadog style with KPI cards,
 * status pills, px-tables, etc.).
 *
 * The OLD /proxmox workspace (still routable, still works) uses the
 * older sw-* design system and is NOT wrapped in this component.
 *
 * Props:
 *   - title: page title (e.g. "VMs", "LXC Containers", "VM 100")
 *   - subtitle: short line under the title
 *   - actions: optional right-side action buttons (Add VM, Refresh)
 *   - showSectionNav: true (default) = show section nav in sidebar.
 *     false = suppress the nav (used on full-screen pages like
 *     console / wizards).
 */
import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';

export interface ProxmoxShellProps {
  title: string;
  subtitle?: string;
  eyebrow?: string;
  actions?: ReactNode;
  children: ReactNode;
  showSectionNav?: boolean;
}

const sectionLinks: Array<{ href: string; label: string; icon: string }> = [
  { href: '/proxmox-vms', label: 'VMs', icon: '▣' },
  { href: '/proxmox-vms/new', label: 'Create VM', icon: '＋' },
  { href: '/proxmox-lxc', label: 'LXC Containers', icon: '▤' },
  { href: '/proxmox-lxc/new', label: 'Create LXC', icon: '＋' },
  { href: '/proxmox/nodes', label: 'Nodes', icon: '◉' },
  { href: '/proxmox/storage', label: 'Storage', icon: '◫' },
  { href: '/proxmox/network', label: 'Network & Firewall', icon: '⌁' },
  { href: '/proxmox/users', label: 'Access & Tokens', icon: '◇' },
  { href: '/proxmox/tasks', label: 'Tasks, Pools & Backup', icon: '◴' },
  { href: '/proxmox/security', label: 'TLS & ACME', icon: '◈' },
  { href: '/proxmox/templates', label: 'Templates & Cloud-init', icon: '◇' },
  { href: '/proxmox/cluster', label: 'HA & Cluster', icon: '◎' },
  { href: '/proxmox/monitoring', label: 'Monitoring', icon: '⌁' },
];

export default function ProxmoxShell({
  title,
  subtitle,
  eyebrow = 'TIER 14 · PROXMOX VE',
  actions,
  children,
  showSectionNav = true,
}: ProxmoxShellProps) {
  const location = useLocation();

  function signOut() {
    localStorage.removeItem('stackwatch.token');
    window.location.href = '/login';
  }

  return (
    <div className="px-shell">
      <header className="px-topbar">
        <div>
          <Link to="/proxmox-vms" className="px-brand">
            <span className="px-brand-mark">S</span>
            <span>StackWatch</span>
          </Link>
          <span className="px-product-label">Infrastructure control plane</span>
        </div>
        <div />
        <div className="px-top-actions">
          <button
            type="button"
            className="px-btn px-topbar-btn"
            onClick={signOut}
          >
            Sign out
          </button>
        </div>
      </header>
      <nav className="px-sidebar">
        <div className="px-side-title">CONTROL PLANE</div>
        {showSectionNav &&
          sectionLinks.map((item) => {
            const isActive =
              location.pathname === item.href ||
              (item.href !== '/proxmox-vms' && location.pathname.startsWith(item.href));
            return (
              <Link
                key={item.href}
                to={item.href}
                className={`px-nav-item${isActive ? ' active' : ''}`}
              >
                <span>{item.icon}</span>
                <span>{item.label}</span>
              </Link>
            );
          })}
        <div className="px-side-note">
          <strong>Tier 14</strong>
          <span>Proxmox full Web UI replacement</span>
          <small>Built per MASTER_BUILD_PLAN §14.</small>
        </div>
      </nav>
      <main className="px-main">
        <div className="px-page-head">
          <div>
            <span className="px-eyebrow">{eyebrow}</span>
            <h1>{title}</h1>
            {subtitle && <p>{subtitle}</p>}
          </div>
          {actions && <div className="px-page-actions">{actions}</div>}
        </div>
        {children}
      </main>
    </div>
  );
}