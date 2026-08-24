import { Link } from 'react-router-dom';
import { ProxmoxHost } from '../../lib/proxmox';
import StatusPill from '../shared/StatusPill';
import { formatRelative } from '../../lib/clock';

/**
 * HostList — the Operations panel's body. Renders one row per
 * connected Proxmox host in Datadog hosts-table column layout:
 *
 *   Status (pills) | Hostname (mono) | IP (mono) | OS pill |
 *   Last seen (relative) | Actions (kebab)
 *
 * Row hover: `--surface-2` background + left-edge 2px accent stripe
 * in the row's status color. Falls back to `<EmptyState>`-shaped
 * placeholder when the host list is empty so the page never shows
 * raw "No data" copy.
 */
export default function HostList({ hosts }: { hosts: ProxmoxHost[] }) {
  if (!hosts.length) {
    return (
      <div className="dash-host-list-empty">
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
  const now = new Date();
  return (
    <div className="dash-host-list" role="table" aria-label="Connected infrastructure hosts">
      <div className="dash-host-row dash-host-row-head" role="row">
        <span role="columnheader">Status</span>
        <span role="columnheader">Hostname</span>
        <span role="columnheader">Address</span>
        <span role="columnheader">Last seen</span>
        <span role="columnheader" aria-label="Row actions" />
      </div>
      {hosts.map((host) => {
        const status = host.status || 'unknown';
        const tone = statusTone(status);
        const lastSeen = host.last_check_at ? formatRelative(new Date(host.last_check_at), now) : 'never';
        const ip = extractIp(host.base_url);
        return (
          <div
            className={`dash-host-row dash-host-row-${tone}`}
            key={host.id}
            role="row"
            data-tone={tone}
          >
            <span className="dash-host-status" role="cell">
              <StatusPill status={status} />
            </span>
            <span className="dash-host-name" role="cell">
              <strong>{host.name || 'Unnamed host'}</strong>
            </span>
            <span className="dash-host-ip" role="cell">
              {ip}
            </span>
            <span className="dash-host-seen" role="cell">
              {lastSeen}
            </span>
            <span className="dash-host-actions" role="cell">
              <button
                type="button"
                className="dash-host-kebab"
                aria-label={`Actions for ${host.name || host.id}`}
                title="Host actions"
              >
                ⋮
              </button>
            </span>
          </div>
        );
      })}
    </div>
  );
}

/** Map Proxmox host status string to the row-tone modifier used
 *  by `.dash-host-row-{tone}` for the left-edge accent stripe.
 *  Mirrors the StatusPill resolution but exposed as a string so
 *  CSS can key off it for the hover stripe color. */
function statusTone(status: string): 'good' | 'warn' | 'bad' {
  if (status === 'bad' || status === 'down' || status === 'offline' || status === 'crit') return 'bad';
  if (status === 'warn' || status === 'degraded' || status === 'stale') return 'warn';
  return 'good';
}

/** Pull the host portion out of `https://host:8006/api2/json`-style
 *  base URLs so the table can render a clean mono address without
 *  leaking the path. Falls back to the raw string when we can't
 *  parse a URL (e.g. dev-mode placeholder). */
function extractIp(baseUrl: string): string {
  if (!baseUrl) return '—';
  try {
    const parsed = new URL(baseUrl);
    return parsed.host || baseUrl;
  } catch {
    return baseUrl;
  }
}
