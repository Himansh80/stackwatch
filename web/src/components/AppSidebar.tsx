import { Link } from 'react-router-dom';
import { ReactNode } from 'react';

/**
 * AppSidebar — left rail used on all authenticated pages.
 *
 * Owns the brand mark, workspace + infrastructure nav, optional
 * "Health" alert pill (shown when there are firing alerts), and
 * the bottom "Control plane online" status + sign-out button.
 *
 * Each page passes its own active-nav key + alert count + logout
 * handler. The sidebar itself has no internal data fetches.
 */
interface SidebarProps {
  active: 'dashboard' | 'billing' | 'homelab' | 'profile' | 'settings' | 'proxmox' | 'truenas' | 'apm' | 'logs' | 'rum' | 'synthetics' | 'security' | 'cspm' | 'cicd' | 'database' | 'incidents' | 'notebooks' | 'shared' | 'intelligence' | 'enterprise' | 'platform';
  onLogout: () => void;
  apiVersion?: string;
  firingAlerts?: number;
  /**
   * Deprecated: the global sidebar always shows every nav item. Kept
   * on the interface for backward compat with the old call sites but
   * no longer filters the list. Remove once all page-level callers
   * are updated.
   * @deprecated
   */
  show?: ReadonlyArray<SidebarProps['active']>;
  userMenuSlot?: ReactNode;
}

const ALL_NAV_ITEMS: Array<{
  key: SidebarProps['active'];
  to: string;
  icon: string;
  label: string;
  section: 'workspace' | 'infrastructure' | 'observability' | 'operations';
}> = [
  { key: 'dashboard', to: '/dashboard', icon: '⌂', label: 'Overview', section: 'workspace' },
  { key: 'billing', to: '/billing', icon: '$', label: 'Billing', section: 'workspace' },
  { key: 'homelab', to: '/homelab', icon: '◉', label: 'Homelab', section: 'workspace' },
  { key: 'profile', to: '/profile', icon: '◉', label: 'Profile', section: 'workspace' },
  { key: 'settings', to: '/settings', icon: '⚙', label: 'Settings', section: 'workspace' },
  // Tier 14 (Phase 14.4+): the old /proxmox workspace has been
  // superseded by dedicated routes (/proxmox-vms, /proxmox-lxc, ...).
  // The sidebar now links straight into the new VM list page. The
  // legacy workspace stays routable at /proxmox for users who
  // bookmarked it but is no longer the Proxmox entry point.
  { key: 'proxmox', to: '/proxmox-vms', icon: '◈', label: 'Proxmox', section: 'infrastructure' },
  { key: 'truenas', to: '/truenas', icon: '▤', label: 'TrueNAS', section: 'infrastructure' },
  { key: 'apm', to: '/apm', icon: '◴', label: 'APM', section: 'observability' },
  { key: 'logs', to: '/logs', icon: '≡', label: 'Logs', section: 'observability' },
  { key: 'rum', to: '/rum', icon: '◎', label: 'RUM', section: 'observability' },
  { key: 'synthetics', to: '/synthetics', icon: '◔', label: 'Synthetics', section: 'observability' },
  { key: 'security', to: '/security', icon: '◍', label: 'Security', section: 'observability' },
  { key: 'cspm', to: '/cspm', icon: '◐', label: 'CSPM', section: 'observability' },
  { key: 'cicd', to: '/cicd', icon: '⇄', label: 'CI/CD', section: 'observability' },
  { key: 'database', to: '/database', icon: '◧', label: 'Database', section: 'observability' },
  { key: 'incidents', to: '/incidents', icon: '⚑', label: 'Incidents', section: 'operations' },
  { key: 'notebooks', to: '/notebooks', icon: '◰', label: 'Notebooks', section: 'operations' },
  { key: 'shared', to: '/shared', icon: '⌬', label: 'Collaboration', section: 'operations' },
  { key: 'intelligence', to: '/intelligence', icon: '✦', label: 'Intelligence', section: 'observability' },
  { key: 'enterprise', to: '/enterprise', icon: '◈', label: 'Enterprise', section: 'operations' },
  { key: 'platform', to: '/platform', icon: '◊', label: 'Platform', section: 'operations' },
];

export default function AppSidebar({
  active,
  onLogout,
  apiVersion,
  firingAlerts,
}: SidebarProps) {
  // The global sidebar always shows every nav item. Tier-14 Proxmox
  // pages render their own (px-) sidebar via ProxmoxShell, so the global
  // one should never shrink on a per-page basis. The legacy `show` prop
  // is ignored — kept on the interface for backward compat with old call
  // sites that still pass it.
  const _unusedShow: ReadonlyArray<SidebarProps['active']> | undefined = undefined;
  void _unusedShow;
  const visible = ALL_NAV_ITEMS;
  const workspaceItems = visible.filter((i) => i.section === 'workspace');
  const infraItems = visible.filter((i) => i.section === 'infrastructure');
  const observabilityItems = visible.filter((i) => i.section === 'observability');
  const operationsItems = visible.filter((i) => i.section === 'operations');

  return (
    <aside className="dash-sidebar">
      <Link className="dash-brand" to="/dashboard">
        <span className="dash-brand-mark">S</span>
        <span>
          <strong>StackWatch</strong>
          <small>Infrastructure control plane</small>
        </span>
      </Link>
      <div className="dash-nav-section">
        <span className="dash-nav-heading">Workspace</span>
        {workspaceItems.map((item) => (
          <Link
            key={item.key}
            className={`dash-nav-item ${active === item.key ? 'dash-nav-active' : ''}`}
            to={item.to}
          >
            <span className="dash-nav-icon">{item.icon}</span>
            <span className="dash-nav-label">{item.label}</span>
          </Link>
        ))}
      </div>
      <div className="dash-nav-section">
        <span className="dash-nav-heading">Infrastructure</span>
        {infraItems.map((item) => (
          <Link
            key={item.key}
            className={`dash-nav-item ${active === item.key ? 'dash-nav-active' : ''}`}
            to={item.to}
          >
            <span className="dash-nav-icon">{item.icon}</span>
            <span className="dash-nav-label">{item.label}</span>
          </Link>
        ))}
      </div>
      {(firingAlerts ?? 0) > 0 ? (
        <div className="dash-nav-section">
          <span className="dash-nav-heading">Health</span>
          <div
            className="dash-nav-alert-pill"
            title={`${firingAlerts} firing alert${firingAlerts === 1 ? '' : 's'}`}
          >
            <span className="dash-status-dot dash-status-bad" />
            {firingAlerts} firing
          </div>
        </div>
      ) : null}
      {observabilityItems.length > 0 ? (
        <div className="dash-nav-section">
          <span className="dash-nav-heading">Observability</span>
          {observabilityItems.map((item) => (
            <Link
              key={item.key}
              className={`dash-nav-item ${active === item.key ? 'dash-nav-active' : ''}`}
              to={item.to}
            >
              <span className="dash-nav-icon">{item.icon}</span>
              <span className="dash-nav-label">{item.label}</span>
            </Link>
          ))}
        </div>
      ) : null}
      {operationsItems.length > 0 ? (
        <div className="dash-nav-section">
          <span className="dash-nav-heading">Operations</span>
          {operationsItems.map((item) => (
            <Link
              key={item.key}
              className={`dash-nav-item ${active === item.key ? 'dash-nav-active' : ''}`}
              to={item.to}
            >
              <span className="dash-nav-icon">{item.icon}</span>
              <span className="dash-nav-label">{item.label}</span>
            </Link>
          ))}
        </div>
      ) : null}
            {/* Tier 14: Proxmox nav lives inside ProxmoxShell (the page's own
                sidebar), not here. The global AppSidebar just shows the
                single 'Proxmox' link to the VM list. */}
      <div className="dash-sidebar-bottom">
        <div className="dash-connection">
          <span className="dash-live-dot" />
          Control plane online
          <small>{apiVersion || 'StackWatch API'}</small>
        </div>
        <button className="dash-sidebar-logout" onClick={onLogout}>↪ Sign out</button>
      </div>
    </aside>
  );
}