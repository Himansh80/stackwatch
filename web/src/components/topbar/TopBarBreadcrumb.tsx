/**
 * TopBarBreadcrumb.tsx — derived from current URL.
 * Maps URL path → section/label using nav-config.ts.
 *
 * Example:
 *   /dashboard            → "Overview"
 *   /proxmox-vms/abc/100  → "Proxmox / VM 100"
 */
import { useLocation, matchPath } from 'react-router-dom';
import { ALL_NAV_ITEMS, NavItem } from '../sidebar/nav-config';

interface Crumb {
  label: string;
  path: string;
}

/**
 * Find the nav item whose path is the longest prefix of the current URL.
 */
function findActiveItem(currentPath: string): NavItem | undefined {
  return ALL_NAV_ITEMS
    .slice()
    .sort((a, b) => b.path.length - a.path.length)
    .find((item) => currentPath === item.path || currentPath.startsWith(item.path + '/'));
}

export default function TopBarBreadcrumb() {
  const location = useLocation();
  const active = findActiveItem(location.pathname);

  // Optional dynamic detail (e.g., VM ID from path)
  let detail: string | null = null;
  const vmMatch = matchPath('/proxmox-vms/:hostId/:node/:vmid', location.pathname);
  if (vmMatch) detail = `VM ${vmMatch.params.vmid}`;
  const lxcMatch = matchPath('/proxmox-lxc/:hostId/:node/:vmid', location.pathname);
  if (lxcMatch) detail = `LXC ${lxcMatch.params.vmid}`;

  const crumbs: Crumb[] = active ? [{ label: active.label, path: active.path }] : [];
  if (detail) crumbs.push({ label: detail, path: location.pathname });

  if (crumbs.length === 0) return null;

  return (
    <nav className="tb-breadcrumb" aria-label="Breadcrumb">
      {crumbs.map((c, i) => (
        <span key={c.path + i} className="tb-crumb">
          {i > 0 && <span className="tb-crumb-sep" aria-hidden="true">/</span>}
          <span className="tb-crumb-label">{c.label}</span>
        </span>
      ))}
    </nav>
  );
}