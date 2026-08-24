import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, health, me } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import ProfileMenu from '../components/ProfileMenu';
import CommandPalette from '../components/CommandPalette';
import FilterBar from '../components/FilterBar';
import TimeWidget from '../components/TimeWidget';
import MetricCard from '../components/dashboard/MetricCard';
import SkeletonCard from '../components/dashboard/SkeletonCard';
import TrendChart from '../components/dashboard/TrendChart';
import HostList from '../components/dashboard/HostList';
import StatusPill from '../components/dashboard/StatusPill';
import WelcomeHeader from '../components/dashboard/WelcomeHeader';
import ErrorBar from '../components/dashboard/ErrorBar';
import AppSidebar from '../components/AppSidebar';
import { HeartIcon, NetworkIcon, PlayIcon, ServerIcon } from '../components/icons';
import { listFrom, objectFrom, ProxmoxHost, ProxmoxResource, pxGet, formatBytes, formatPercent } from '../lib/proxmox';
import { greetingFor, formatRelative } from '../lib/clock';

type Json = Record<string, unknown>;

interface Alert {
  id: string;
  state: string;
  severity: string;
  name: string;
}

interface Snapshot {
  hosts: ProxmoxHost[];
  nodes: Json[];
  resources: ProxmoxResource[];
  health: Json;
  user: Json;
  tenant: Json;
  alerts: Alert[];
}

const emptySnapshot: Snapshot = {
  hosts: [], nodes: [], resources: [], health: {}, user: {}, tenant: {}, alerts: [],
};

const text = (valueToRead: unknown, fallback = '—'): string => {
  if (valueToRead === null || valueToRead === undefined || valueToRead === '') return fallback;
  return String(valueToRead);
};
const value = (obj: unknown, ...path: string[]): unknown => {
  let cur: unknown = obj;
  for (const key of path) {
    if (cur && typeof cur === 'object' && key in (cur as Record<string, unknown>)) {
      cur = (cur as Record<string, unknown>)[key];
    } else {
      return undefined;
    }
  }
  return cur;
};

const isRunning = (row: ProxmoxResource): boolean => {
  const status = String(row.status || '').toLowerCase();
  return status === 'running' || status === 'online' || status === 'active' || status === '';
};

/**
 * Dashboard — top-level route at /dashboard.
 *
 * Owns data loading, top-level state (snapshot, error, history),
 * Cmd+K binding, and the page shell (sidebar + topbar). Every
 * visible section is delegated to a component in
 * `components/dashboard/` so this file stays a thin shell.
 */
export default function Dashboard() {
  const logout = useLogout();
  const abortRef = useRef<AbortController | null>(null);
  const [snapshot, setSnapshot] = useState<Snapshot>(emptySnapshot);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [now, setNow] = useState<Date>(() => new Date());
  // Rolling history of metric values for sparklines. Capped at 30
  // samples (~30 minutes at the 60s refresh cadence) so the
  // sparkline gives useful shape without unbounded memory growth.
  const [history, setHistory] = useState<{ hosts: number[]; nodes: number[]; running: number[]; cpu: number[] }>({
    hosts: [], nodes: [], running: [], cpu: [],
  });
  // Command palette open state (Cmd+K / Ctrl+K).
  const [paletteOpen, setPaletteOpen] = useState(false);
  // Search query for the workloads inventory list.
  const [search, setSearch] = useState('');

  async function loadDashboard() {
    setError('');
    // Cancel any in-flight dashboard request before starting a new one.
    // Without this, a slow request from the previous cycle can land
    // after the user has already signed out, set state on a now-
    // unmounted component, and surface an 'unauthorized' error before
    // the route swap happens.
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    try {
      const [healthData, meData, hostsData, alertsData] = await Promise.all([
        health({ signal: ctrl.signal }),
        me({ signal: ctrl.signal }),
        api<{ hosts?: ProxmoxHost[] }>('GET', '/api/v1/proxmox/hosts', undefined, true, { signal: ctrl.signal }),
        // Alerts endpoint may not exist yet on this build — catch and
        // degrade to an empty array rather than failing the whole load.
        api<{ alerts?: Alert[] }>('GET', '/api/v1/alerts?state=open&limit=200', undefined, true, { signal: ctrl.signal }).catch(() => ({ alerts: [] })),
      ]);
      const hosts = hostsData.hosts || [];
      const primary = hosts[0];
      const [nodesPayload, resourcePayload] = primary
        ? await Promise.all([pxGet(primary.id, '/nodes'), pxGet(primary.id, '/cluster/resources')])
        : [{}, {}];
      const resources = listFrom(resourcePayload, 'resources', 'vms') as ProxmoxResource[];
      const runningCount = resources.filter(isRunning).length;
      const avgCpu = resources.length
        ? resources.map((r) => Number(r.cpu)).filter(Number.isFinite).reduce((a, b) => a + b, 0) / resources.length
        : 0;
      setSnapshot({
        hosts,
        nodes: listFrom(nodesPayload, 'nodes'),
        resources,
        health: objectFrom(healthData),
        user: objectFrom(meData.user),
        tenant: objectFrom(meData.tenant),
        alerts: alertsData.alerts || [],
      });
      setLastUpdated(new Date());
      setHistory((prev) => ({
        hosts: [...prev.hosts, hosts.length].slice(-30),
        nodes: [...prev.nodes, listFrom(nodesPayload, 'nodes').length].slice(-30),
        running: [...prev.running, runningCount].slice(-30),
        cpu: [...prev.cpu, avgCpu * 100].slice(-30),
      }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load the live dashboard.');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadDashboard();
    const timer = window.setInterval(() => void loadDashboard(), 30000);
    const clockTimer = window.setInterval(() => setNow(new Date()), 30 * 1000);
    return () => {
      window.clearInterval(timer);
      window.clearInterval(clockTimer);
      abortRef.current?.abort();
    };
  }, []);

  // Cmd+K / Ctrl+K — open the command palette. Bound globally so it
  // works from any focused input.
  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setPaletteOpen(true);
      }
      if (event.key === 'Escape' && paletteOpen) {
        setPaletteOpen(false);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [paletteOpen]);

  const running = useMemo(() => snapshot.resources.filter(isRunning).length, [snapshot.resources]);
  const stopped = useMemo(
    () => snapshot.resources.filter((row) => text(row.status, '').toLowerCase() === 'stopped').length,
    [snapshot.resources],
  );
  const tenantName = text(value(snapshot.tenant, 'name'), 'Workspace');
  const userName = text(value(snapshot.user, 'full_name'), text(value(snapshot.user, 'email'), 'Operator'));
  const firstName = (userName || '').split(' ')[0] || 'Operator';
  const firingAlerts = snapshot.alerts.filter((a) => a.state === 'open' || a.state === 'firing').length;
  const apiHealthStatus = String(value(snapshot.health, 'status') || '');
  const apiHealthTone = apiHealthStatus === 'ok' ? 'green' : apiHealthStatus ? 'amber' : 'amber';

  // Filter resources by the search query — matches name, type, vmid, status.
  const filteredResources = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return snapshot.resources;
    return snapshot.resources.filter((row) => {
      const haystack = `${text(row.name)} ${text(row.type)} ${text(row.vmid)} ${text(row.status)}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [snapshot.resources, search]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="dashboard"
        onLogout={logout}
        apiVersion={text(value(snapshot.health, 'version'), 'StackWatch API')}
        firingAlerts={firingAlerts}
        show={['dashboard', 'billing', 'proxmox', 'truenas']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Hello, {firstName}</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">{greetingFor(now)}.</strong>
              <span className="dash-greeting-live" title="You are live" aria-label="Live">
                <span className="dash-live-dot" aria-hidden />
                <span className="dash-greeting-live-text">Live</span>
              </span>
            </div>
          </div>
          <div className="dash-topbar-center">
            <button
              className="dash-topbar-search"
              onClick={() => setPaletteOpen(true)}
              title="Search & navigate (Cmd+K)"
              aria-label="Open command palette"
            >
              <span className="dash-topbar-search-icon" aria-hidden="true">⌕</span>
              <span className="dash-topbar-search-placeholder">Search & navigate…</span>
              <kbd className="dash-topbar-search-kbd">⌘</kbd>
              <kbd className="dash-topbar-search-kbd">K</kbd>
            </button>
          </div>
          <div className="dash-top-actions">
            <TimeWidget />
            <button
              className="dash-icon-button"
              onClick={() => window.location.reload()}
              aria-label="Refresh page"
              title="Refresh page"
            >↻</button>
            <ProfileMenu
              firstName={firstName}
              fullName={userName}
              tenantName={tenantName}
              initials={userName.charAt(0).toUpperCase()}
            />
          </div>
        </header>
        <div className="dash-content">
          {error && <ErrorBar error={error} onRetry={() => void loadDashboard()} />}
          <WelcomeHeader userName={userName} lastUpdated={lastUpdated} formatRelative={formatRelative} now={now} />
          <section className="dash-metric-grid">
            {loading && !snapshot.hosts.length ? (
              <>
                <SkeletonCard />
                <SkeletonCard />
                <SkeletonCard />
                <SkeletonCard />
              </>
            ) : (
              <>
                <MetricCard label="Connected hosts" value={String(snapshot.hosts.length)} hint="Registered control planes" tone="cyan" icon={<ServerIcon />} sparkline={history.hosts} />
                <MetricCard label="Compute nodes" value={String(snapshot.nodes.length)} hint="Across your Proxmox fabric" tone="indigo" icon={<NetworkIcon />} sparkline={history.nodes} />
                <MetricCard label="Running workloads" value={String(running)} hint={`${stopped} stopped`} tone="green" icon={<PlayIcon />} sparkline={history.running} />
                <MetricCard label="API health" value={apiHealthStatus || '—'} hint={text(value(snapshot.health, 'version'), 'StackWatch API')} tone={apiHealthTone} icon={<HeartIcon />} sparkline={history.cpu.map((c) => Math.min(100, c))} />
              </>
            )}
          </section>
          <section className="dash-grid-main">
            <article className="dash-panel dash-chart-panel">
              <div className="dash-panel-head">
                <div>
                  <span className="dash-eyebrow">Live telemetry</span>
                  <h3>Workload pressure</h3>
                </div>
                <span className="dash-panel-context">Current snapshot</span>
              </div>
              <TrendChart resources={snapshot.resources} />
            </article>
            <article className="dash-panel">
              <div className="dash-panel-head">
                <div>
                  <span className="dash-eyebrow">Operations</span>
                  <h3>Infrastructure status</h3>
                </div>
                <span className="dash-panel-context">{snapshot.hosts.length} host{snapshot.hosts.length === 1 ? '' : 's'}</span>
              </div>
              <HostList hosts={snapshot.hosts} />
            </article>
          </section>
          <section className="dash-panel">
            <div className="dash-panel-head">
              <div>
                <span className="dash-eyebrow">Compute inventory</span>
                <h3>Workloads and resources</h3>
              </div>
              <div className="dash-panel-controls">
                <FilterBar
                  search={search}
                  onSearchChange={setSearch}
                  placeholder="Search workloads by name, type, or status..."
                  ariaLabel="Search workloads"
                />
                <Link className="dash-text-link" to="/proxmox">Open full workspace →</Link>
              </div>
            </div>
            {filteredResources.length ? (
              <div className="dash-resource-grid">
                {filteredResources.slice(0, 12).map((resource, index) => (
                  <div className="dash-resource-card" key={String(resource.id || resource.vmid || index)}>
                    <div className="dash-resource-head">
                      <span className="dash-resource-type">{text(resource.type, 'resource')}</span>
                      <StatusPill status={resource.status ?? ''} />
                    </div>
                    <strong>{text(resource.name, `Workload ${text(resource.vmid, String(index + 1))}`)}</strong>
                    <div className="dash-resource-meta">
                      <span>
                        CPU <b>{typeof resource.cpu === 'number' ? formatPercent(resource.cpu) : '—'}</b>
                      </span>
                      <span>
                        RAM <b>{formatBytes(resource.mem)}</b>
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ) : search ? (
              <div className="dash-empty-inline">
                <span>⌕</span>
                <div>
                  <strong>No workloads match &ldquo;{search}&rdquo;</strong>
                  <p>
                    Try a different search term, or clear the search to see all {snapshot.resources.length} workload{snapshot.resources.length === 1 ? '' : 's'}.
                  </p>
                </div>
              </div>
            ) : (
              <div className="dash-empty-inline">
                <span>◇</span>
                <div>
                  <strong>No workloads reported</strong>
                  <p>The dashboard will populate as soon as the connected host returns resource inventory.</p>
                </div>
              </div>
            )}
          </section>
        </div>
      </main>
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  );
}
