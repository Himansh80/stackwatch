import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, health, me } from '../lib/api';
import SkeletonCard from '../components/dashboard/SkeletonCard';
import TrendChart from '../components/dashboard/TrendChart';
import HostList from '../components/dashboard/HostList';
import KpiCard from '../components/shared/KpiCard';
import StatusPill from '../components/shared/StatusPill';
import WelcomeHeader from '../components/dashboard/WelcomeHeader';
import ErrorBar from '../components/dashboard/ErrorBar';
import { motion, kpiStagger } from '../lib/motion';
import EmptyState from '../components/shared/EmptyState';
import { HeartIcon, NetworkIcon, PlayIcon, ServerIcon } from '../components/icons';
import { listFrom, objectFrom, ProxmoxHost, ProxmoxResource, formatBytes, formatPercent } from '../lib/proxmox';
import { formatRelative } from '../lib/clock';

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
 * NOTE (2026-08-26): Previously rendered its own sidebar + topbar +
 * CommandPalette. Those now live in AppShell (which wraps every
 * authenticated route via App.tsx). Dashboard just renders page content.
 */
export default function Dashboard() {
  const [snapshot, setSnapshot] = useState<Snapshot>(emptySnapshot);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [now, setNow] = useState<Date>(() => new Date());
  const abortRef = useRef<AbortController | null>(null);
  const [history, setHistory] = useState<{ hosts: number[]; nodes: number[]; running: number[]; cpu: number[] }>({
    hosts: [], nodes: [], running: [], cpu: [],
  });
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
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const [healthRes, meRes] = await Promise.all([
        health().catch(() => null),
        me().catch(() => null),
      ]);
      const [clusterResources, clusterStatus, hostsRes] = await Promise.all([
              api('GET', '/api/v1/proxmox/cluster/resources').catch(() => null),
              api('GET', '/api/v1/proxmox/cluster/status').catch(() => null),
              api<ProxmoxHost[]>('GET', '/api/v1/proxmox/hosts').catch(() => null),
            ]);
      const resources = listFrom(objectFrom(clusterResources)?.data, 'resources') as ProxmoxResource[];
      const nodes = listFrom(clusterStatus?.data, 'nodes') as Json[];
      const hosts = (hostsRes as ProxmoxHost[] | null) || [];
      const firing = (Array.isArray(objectFrom(healthRes)?.alerts)
        ? (objectFrom(healthRes).alerts as Alert[])
        : []) as Alert[];
      setSnapshot({
        hosts,
        nodes,
        resources,
        health: objectFrom(healthRes),
        user: objectFrom(meRes),
        tenant: objectFrom(objectFrom(meRes))?.tenant as Json,
        alerts: firing,
      });
      setHistory((prev) => ({
        hosts: appendBounded(prev.hosts, hosts.length, 24),
        nodes: appendBounded(prev.nodes, nodes.length, 24),
        running: appendBounded(prev.running, resources.filter(isRunning).length, 24),
        cpu: appendBounded(
          prev.cpu,
          averageCpu(resources),
          24,
        ),
      }));
      setLastUpdated(new Date());
    } catch (err) {
      if (!controller.signal.aborted) {
        setError((err as Error).message || 'Failed to load dashboard');
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadDashboard();
    const timer = window.setInterval(loadDashboard, 30000);
    const clockTimer = window.setInterval(() => setNow(new Date()), 1000);
    return () => {
      window.clearInterval(timer);
      window.clearInterval(clockTimer);
      abortRef.current?.abort();
    };
  }, []);

  const running = useMemo(() => snapshot.resources.filter(isRunning).length, [snapshot.resources]);
    const stopped = useMemo(
      () => snapshot.resources.filter((row) => text(row.status, '').toLowerCase() === 'stopped').length,
      [snapshot.resources],
    );
    const apiHealthStatus = text(value(snapshot.health, 'ok') === true ? 'ok' : value(snapshot.health, 'ok') === false ? 'down' : '');
    const userName = text(value(snapshot.user, 'full_name'), text(value(snapshot.user, 'email'), 'Operator'));
    const filteredResources = useMemo(() => {
      const q = search.trim().toLowerCase();
      if (!q) return snapshot.resources;
      return snapshot.resources.filter((r) =>
        [text(r.name), text(r.type), text(r.status), text(r.node)]
          .join(' ')
          .toLowerCase()
          .includes(q),
      );
    }, [snapshot.resources, search]);

  return (
    <div className="dash-page">
      {error && <ErrorBar error={error} onRetry={() => void loadDashboard()} />}
      <WelcomeHeader userName={userName} lastUpdated={lastUpdated} formatRelative={formatRelative} now={now} />
      <motion.section
        className="dash-metric-grid"
        variants={kpiStagger}
        initial="hidden"
        animate="show"
      >
        {loading && !snapshot.hosts.length ? (
          <>
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </>
        ) : (
          <>
            <KpiCard label="Connected hosts" value={snapshot.hosts.length} delta="Registered control planes" accent="cyan" icon={<ServerIcon />} sparkline={history.hosts} />
            <KpiCard label="Compute nodes" value={snapshot.nodes.length} delta="Across your Proxmox fabric" accent="indigo" icon={<NetworkIcon />} sparkline={history.nodes} />
            <KpiCard label="Running workloads" value={running} delta={`${stopped} stopped`} accent="green" icon={<PlayIcon />} sparkline={history.running} />
            <KpiCard
              label="API health"
              value={apiHealthStatus || '—'}
              delta={text(value(snapshot.health, 'version'), 'StackWatch API')}
              status={apiHealthStatus === 'ok' ? 'up' : 'down'}
              icon={<HeartIcon />}
              sparkline={history.cpu.map((c) => Math.min(100, c))}
            />
          </>
        )}
      </motion.section>
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
            <input
              className="dash-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search workloads by name, type, or status…"
              aria-label="Search workloads"
            />
            <Link className="dash-text-link" to="/proxmox-vms">Open full workspace →</Link>
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
          <EmptyState
            illustration={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7" />
                <path d="m20 20-3.5-3.5" />
              </svg>
            }
            headline={`No workloads match “${search}”`}
            subhead={`Try a different search term, or clear the search to see all ${snapshot.resources.length} workload${snapshot.resources.length === 1 ? '' : 's'}.`}
            cta={{ label: 'Clear search', onClick: () => setSearch('') }}
          />
        ) : (
          <EmptyState
            illustration={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <rect x="3" y="5" width="18" height="12" rx="2" />
                <path d="M8 21h8M12 17v4" />
              </svg>
            }
            headline="No workloads reported"
            subhead="The dashboard will populate as soon as the connected host returns resource inventory."
          />
        )}
      </section>
    </div>
  );
}

function appendBounded(values: number[], next: number, max: number): number[] {
  const out = [...values, next];
  if (out.length > max) out.splice(0, out.length - max);
  return out;
}

function averageCpu(resources: ProxmoxResource[]): number {
  let total = 0;
  let count = 0;
  for (const r of resources) {
    if (typeof r.cpu === 'number') { total += r.cpu; count += 1; }
  }
  return count === 0 ? 0 : Math.round((total / count) * 10) / 10;
}