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
  active: 'dashboard' | 'billing' | 'profile' | 'settings' | 'proxmox' | 'truenas' | 'apm' | 'logs' | 'rum' | 'synthetics' | 'security' | 'cspm' | 'cicd' | 'database' | 'incidents' | 'notebooks';
  onLogout: () => void;
  apiVersion?: string;
  firingAlerts?: number;
  /**
   * Which nav items to show. Defaults to all. Pages that want a
   * stripped-down sidebar pass a subset. E.g. the Dashboard page
   * removes Profile + Settings since they're reachable from the
   * avatar menu.
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
  { key: 'profile', to: '/profile', icon: '◉', label: 'Profile', section: 'workspace' },
  { key: 'settings', to: '/settings', icon: '⚙', label: 'Settings', section: 'workspace' },
  { key: 'proxmox', to: '/proxmox', icon: '◈', label: 'Proxmox', section: 'infrastructure' },
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
];

export default function AppSidebar({
  active,
  onLogout,
  apiVersion,
  firingAlerts,
  show,
}: SidebarProps) {
  // Default: show every nav item. Pass `show` to filter.
  const visible = show
    ? ALL_NAV_ITEMS.filter((i) => show.includes(i.key))
    : ALL_NAV_ITEMS;
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