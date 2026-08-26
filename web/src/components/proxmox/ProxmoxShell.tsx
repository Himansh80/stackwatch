/**
 * Tier 14 (Phase 14.4+) — Shared shell wrapper for all Proxmox pages.
 *
 * The OLD ProxmoxWorkspace has its own internal section tabs (Overview,
 * Compute, Storage, ...) — those don't make sense for the new dedicated
 * routes (/proxmox-vms, /proxmox-lxc, /proxmox-vms/new, ...). So we
 * extract the chrome (topbar + sidebar with sectionless nav links) into
 * this wrapper and let every new page render its own content.
 */
import { ReactNode } from 'react';
import { Link } from 'react-router-dom';

export interface ProxmoxShellProps {
  title: string;
  subtitle?: string;
  eyebrow?: string;
  actions?: ReactNode;
  children: ReactNode;
  showSectionNav?: boolean;
}

const sectionLinks: Array<{ href: string; label: string; icon: string }> = [
  { href: '/proxmox', label: 'Overview (legacy)', icon: '⌂' },
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
  function signOut() {
    localStorage.removeItem('stackwatch.token');
    window.location.href = '/login';
  }

  return (
    <div className="sw-shell">
      <header className="sw-topbar">
        <div>
          <div className="sw-brand">
            <span className="sw-brand-mark">S</span>
            <span>StackWatch</span>
          </div>
          <span className="sw-product-label">Infrastructure control plane</span>
        </div>
        <div />
        <div className="sw-top-actions">
          <button
            type="button"
            className="sw-button sw-button-quiet"
            onClick={signOut}
          >
            Sign out
          </button>
        </div>
      </header>
      <div className="sw-layout">
        <nav className="sw-sidebar">
          <div className="sw-side-title">CONTROL PLANE</div>
          {showSectionNav &&
            sectionLinks.map((item) => (
              <Link key={item.href} to={item.href} className="sw-nav-item">
                <span>{item.icon}</span>
                <span>{item.label}</span>
              </Link>
            ))}
          <div className="sw-side-note">
            <strong>Tier 14</strong>
            <span>Proxmox full Web UI replacement</span>
            <small>Built per MASTER_BUILD_PLAN §14.</small>
          </div>
        </nav>
        <main className="sw-main">
          <div className="sw-page-head">
            <div>
              <span className="sw-eyebrow">{eyebrow}</span>
              <h1>{title}</h1>
              {subtitle && <p>{subtitle}</p>}
            </div>
            {actions && <div className="sw-page-actions">{actions}</div>}
          </div>
          {children}
        </main>
      </div>
    </div>
  );
}