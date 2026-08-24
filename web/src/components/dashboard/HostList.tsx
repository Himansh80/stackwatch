import { Link } from 'react-router-dom';
import { ProxmoxHost } from '../../lib/proxmox';
import StatusPill from '../shared/StatusPill';

/**
 * HostList — the Operations panel's body. Renders one row per
 * connected Proxmox host, or a friendly empty state if none.
 */
export default function HostList({ hosts }: { hosts: ProxmoxHost[] }) {
  if (!hosts.length) {
    return (
      <div className="dash-panel-empty">
        <div className="dash-empty-illustration" aria-hidden="true">
          <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="1.2">
            <rect x="14" y="20" width="92" height="48" rx="6" opacity=".4" />
            <circle cx="60" cy="44" r="14" opacity=".5" />
            <path d="M44 44 L52 52 L76 28" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>
        <strong>No infrastructure connected yet</strong>
        <span>Register a Proxmox host to see your environment here.</span>
        <Link to="/proxmox">Open Proxmox workspace →</Link>
      </div>
    );
  }
  return (
    <div className="dash-host-list">
      {hosts.map((host) => (
        <div className="dash-host-row" key={host.id}>
          <span className="dash-host-icon">⌁</span>
          <span className="dash-host-name">
            <strong>{host.name || 'Unnamed host'}</strong>
            <small>{host.base_url}</small>
          </span>
          <StatusPill status={host.status || 'unknown'} />
        </div>
      ))}
    </div>
  );
}
