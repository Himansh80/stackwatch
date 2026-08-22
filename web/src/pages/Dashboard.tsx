import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, health, me } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import ProfileMenu from '../components/ProfileMenu';
import CommandPalette from '../components/CommandPalette';
import { listFrom, objectFrom, ProxmoxHost, ProxmoxResource, pxGet, formatBytes, formatPercent } from '../lib/proxmox';

type Json = Record<string, unknown>;

type Snapshot = {
  hosts: ProxmoxHost[];
  nodes: Json[];
  resources: ProxmoxResource[];
  health: Json | null;
  user: Json | null;
  tenant: Json | null;
  alerts: { id: string; state: string; severity: string; name: string }[];
};

const emptySnapshot: Snapshot = { hosts: [], nodes: [], resources: [], health: null, user: null, tenant: null, alerts: [] };

function value(row: Json | null, key: string): unknown {
  return row?.[key];
}

function text(valueToRead: unknown, fallback = '—'): string {
  if (valueToRead === null || valueToRead === undefined || valueToRead === '') return fallback;
  return String(valueToRead);
}

function pad2(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

function greetingFor(date: Date): string {
  const hour = date.getHours();
  if (hour < 5) return 'Working late';
  if (hour < 12) return 'Good morning';
  if (hour < 17) return 'Good afternoon';
  if (hour < 22) return 'Good evening';
  return 'Working late';
}

function firstNameOf(full: string): string {
  const trimmed = full.trim();
  if (!trimmed || trimmed === '—') return 'there';
  return trimmed.split(/\s+/)[0]!;
}

function formatClock(date: Date): string {
  const rawHour = date.getHours();
  const hour12 = rawHour % 12 === 0 ? 12 : rawHour % 12;
  const meridiem = rawHour < 12 ? 'AM' : 'PM';
  return `${pad2(hour12)}:${pad2(date.getMinutes())} ${meridiem}`;
}

function formatDayLabel(date: Date): string {
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${days[date.getDay()]} · ${date.getDate()} ${months[date.getMonth()]} ${date.getFullYear()}`;
}

/**
 * Relative time formatter. Datadog-style: "3 sec ago", "4 min ago",
 * "1 hr ago", "yesterday". Falls back to absolute time after a day.
 */
function formatRelative(date: Date, now: Date): string {
  const seconds = Math.max(0, Math.floor((now.getTime() - date.getTime()) / 1000));
  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds} sec ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hr ago`;
  const days = Math.floor(hours / 24);
  if (days === 1) return 'yesterday';
  if (days < 7) return `${days} days ago`;
  return date.toLocaleDateString();
}

function isRunning(row: Json): boolean {
  return ['running', 'online', 'up', 'active'].includes(text(row.status, '').toLowerCase());
}

function statusTone(status: unknown): 'good' | 'warn' | 'bad' {
  const normalized = text(status, '').toLowerCase();
  if (['running', 'online', 'up', 'active', 'ok'].includes(normalized)) return 'good';
  if (['stopped', 'offline', 'down', 'error', 'failed'].includes(normalized)) return 'bad';
  return 'warn';
}

function StatusPill({ status }: { status: unknown }) {
  const tone = statusTone(status);
  return <span className={`dash-status dash-status-${tone}`}><span className="dash-status-dot" />{text(status, 'unknown')}</span>;
}

/**
 * Inline sparkline — a tiny SVG trend line used inside KPI cards.
 * Renders an N-point series as a polyline with optional area fill.
 * Sized for a 60×24 box so it sits cleanly at the right edge of a card.
 */
function Sparkline({ values, tone = 'cyan' }: { values: number[]; tone?: 'cyan' | 'indigo' | 'green' | 'amber' | 'red' }) {
  if (!values.length) {
    return <div className="dash-spark dash-spark-empty" aria-hidden="true" />;
  }
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const range = Math.max(max - min, 1);
  const w = 60;
  const h = 24;
  const points = values.map((number, index) => {
    const x = (index / Math.max(values.length - 1, 1)) * w;
    const y = h - ((number - min) / range) * h;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  }).join(' ');
  const colorMap: Record<string, string> = { cyan: '#38bdf8', indigo: '#818cf8', green: '#34d399', amber: '#fbbf24', red: '#f87171' };
  const stroke = colorMap[tone] || colorMap.cyan;
  return <svg className="dash-spark" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" aria-hidden="true">
    <polyline points={points} fill="none" stroke={stroke} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>;
}

/**
 * Skeleton loader card — used while data is loading on initial mount.
 * Mirrors the MetricCard layout so the page doesn't jump when data arrives.
 */
function SkeletonCard() {
  return <article className="dash-metric dash-metric-skeleton">
    <div className="dash-metric-top"><span className="dash-metric-icon dash-skel-line" style={{ width: 18, height: 18, display: 'inline-block' }} /><span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} /></div>
    <strong className="dash-skel-line" style={{ width: 50, height: 28 }} />
    <span className="dash-metric-hint dash-skel-line" style={{ width: 120, height: 10 }} />
  </article>;
}

function MetricCard({ label, value: metric, hint, tone, icon, sparkline }: { label: string; value: string; hint: string; tone: string; icon: string; sparkline?: number[] }) {
  return <article className={`dash-metric dash-metric-${tone}`}>
    <div className="dash-metric-top"><span className="dash-metric-icon">{icon}</span><span className="dash-metric-label">{label}</span></div>
    <div className="dash-metric-row">
      <strong>{metric}</strong>
      {sparkline && <Sparkline values={sparkline} tone={tone as 'cyan' | 'indigo' | 'green' | 'amber' | 'red'} />}
    </div>
    <span className="dash-metric-hint">{hint}</span>
  </article>;
}

function TrendChart({ resources }: { resources: ProxmoxResource[] }) {
  const values = resources.map((row) => Number(row.cpu)).filter((number) => Number.isFinite(number));
  if (!values.length) {
    return <div className="dash-chart-empty">
      <div className="dash-empty-illustration" aria-hidden="true">
        <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="1.2">
          <rect x="6" y="14" width="108" height="56" rx="6" opacity=".35" />
          <path d="M14 60 L34 44 L54 50 L74 32 L94 38 L106 28" strokeLinecap="round" strokeLinejoin="round" />
          <circle cx="34" cy="44" r="2" /><circle cx="54" cy="50" r="2" /><circle cx="74" cy="32" r="2" /><circle cx="94" cy="38" r="2" />
        </svg>
      </div>
      <strong>Waiting for live resource data</strong>
      <small>Connect a Proxmox host to populate this view.</small>
    </div>;
  }
  const max = Math.max(...values, 1);
  const points = values.map((number, index) => {
    const x = 18 + (index / Math.max(values.length - 1, 1)) * 464;
    const y = 158 - (number / max) * 124;
    return `${x},${y}`;
  }).join(' ');
  return <div className="dash-chart-wrap">
    <svg viewBox="0 0 500 190" role="img" aria-label="Current workload CPU values">
      <defs><linearGradient id="dashArea" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stopColor="#38bdf8" stopOpacity=".34" /><stop offset="1" stopColor="#38bdf8" stopOpacity="0" /></linearGradient></defs>
      <line x1="18" y1="34" x2="482" y2="34" className="dash-grid-line" /><line x1="18" y1="96" x2="482" y2="96" className="dash-grid-line" /><line x1="18" y1="158" x2="482" y2="158" className="dash-grid-line" />
      <polygon points={`18,158 ${points} 482,158`} fill="url(#dashArea)" />
      <polyline points={points} fill="none" stroke="#38bdf8" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
      {values.map((number, index) => { const x = 18 + (index / Math.max(values.length - 1, 1)) * 464; const y = 158 - (number / max) * 124; return <circle key={`${number}-${index}`} cx={x} cy={y} r="3.5" fill="#0b1220" stroke="#67e8f9" strokeWidth="2" />; })}
    </svg>
    <div className="dash-chart-axis"><span>0%</span><span>Current workload CPU by resource</span><span>100%</span></div>
  </div>;
}

export default function Dashboard() {
  const logout = useLogout();
  const abortRef = useRef<AbortController | null>(null);
  const [snapshot, setSnapshot] = useState<Snapshot>(emptySnapshot);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [now, setNow] = useState<Date>(() => new Date());
  // Rolling history of metric values for sparklines. We keep the last
  // 30 samples (≈ 30 minutes at the 60s refresh cadence) so the sparkline
  // gives a useful shape without unbounded memory growth.
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
        api<{ alerts?: { id: string; state: string; severity: string; name: string }[] }>('GET', '/api/v1/alerts?state=open&limit=200', undefined, true, { signal: ctrl.signal }).catch(() => ({ alerts: [] })),
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
      // Append this cycle's values to the rolling history. We cap at
      // 30 samples so the sparkline has a stable shape (~30 minutes
      // of context at the 60s auto-refresh rate).
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
      // Cancel any pending request so it can't fire after the user signs
      // out and surface a stale 401 against the now-unmounted Dashboard.
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
  const stopped = useMemo(() => snapshot.resources.filter((row) => text(row.status, '').toLowerCase() === 'stopped').length, [snapshot.resources]);
  const tenantName = text(value(snapshot.tenant, 'name'), 'Workspace');
  const userName = text(value(snapshot.user, 'full_name'), text(value(snapshot.user, 'email'), 'Operator'));
  const firingAlerts = snapshot.alerts.filter((a) => a.state === 'open' || a.state === 'firing').length;

  // Filter resources by the search query — matches name, type, vmid, status.
  const filteredResources = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return snapshot.resources;
    return snapshot.resources.filter((row) => {
      const haystack = `${text(row.name)} ${text(row.type)} ${text(row.vmid)} ${text(row.status)}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [snapshot.resources, search]);

  return <div className="dash-app">
    <aside className="dash-sidebar">
      <Link className="dash-brand" to="/dashboard"><span className="dash-brand-mark">S</span><span><strong>StackWatch</strong><small>Infrastructure control plane</small></span></Link>
      <div className="dash-nav-section"><span className="dash-nav-heading">Workspace</span>
        <Link className="dash-nav-item dash-nav-active" to="/dashboard"><span className="dash-nav-icon">⌂</span><span className="dash-nav-label">Overview</span></Link>
        <Link className="dash-nav-item" to="/profile"><span className="dash-nav-icon">◉</span><span className="dash-nav-label">Profile</span></Link>
        <Link className="dash-nav-item" to="/billing"><span className="dash-nav-icon">$</span><span className="dash-nav-label">Billing</span></Link>
        <Link className="dash-nav-item" to="/settings"><span className="dash-nav-icon">⚙</span><span className="dash-nav-label">Settings</span></Link>
      </div>
      <div className="dash-nav-section"><span className="dash-nav-heading">Infrastructure</span>
        <Link className="dash-nav-item" to="/proxmox"><span className="dash-nav-icon">◈</span><span className="dash-nav-label">Proxmox</span></Link>
        <Link className="dash-nav-item" to="/truenas"><span className="dash-nav-icon">▤</span><span className="dash-nav-label">TrueNAS</span></Link>
        <Link className="dash-nav-item" to="/dashboard"><span className="dash-nav-icon">⌘</span><span className="dash-nav-label">Terminal</span></Link>
        {firingAlerts > 0 && <div className="dash-nav-section"><span className="dash-nav-heading">Health</span>
          <div className="dash-nav-alert-pill" title={`${firingAlerts} firing alert${firingAlerts === 1 ? '' : 's'}`}>
            <span className="dash-status-dot dash-status-bad" />{firingAlerts} firing
          </div>
        </div>}
      </div>
      <div className="dash-sidebar-bottom"><div className="dash-connection"><span className="dash-live-dot" />Control plane online<small>{text(value(snapshot.health, 'version'), 'StackWatch API')}</small></div><button className="dash-sidebar-logout" onClick={logout}>↪ Sign out</button></div>
    </aside>
    <main className="dash-main">
      <header className="dash-topbar"><div className="dash-greeting"><span className="dash-greeting-eyebrow">Hello, {firstNameOf(userName)}</span><div className="dash-greeting-row"><strong className="dash-greeting-text">{greetingFor(now)}, {firstNameOf(userName)}.</strong><span className="dash-greeting-clock"><span className="dash-greeting-clock-time">{formatClock(now)}</span><span className="dash-greeting-clock-dot" /><span className="dash-greeting-clock-day">{formatDayLabel(now)}</span></span></div></div><div className="dash-top-actions">
        <button className="dash-topbar-hint" onClick={() => setPaletteOpen(true)} title="Search & navigate (Cmd+K)"><kbd>⌘</kbd><kbd>K</kbd><span>Search</span></button>
        <button className="dash-icon-button" onClick={() => window.location.reload()} aria-label="Refresh page" title="Refresh page">↻</button>
        <ProfileMenu firstName={firstNameOf(userName)} fullName={userName} tenantName={tenantName} initials={userName.charAt(0).toUpperCase()} />
      </div></header>
      <div className="dash-content">
        {error && <div className="dash-error"><strong>Live data unavailable</strong><span>{error}</span><button onClick={() => void loadDashboard()}>Retry</button></div>}
        <section className="dash-welcome"><div><span className="dash-eyebrow">Infrastructure overview</span><h2>Good to see you, {userName.split(' ')[0]}.</h2><p>One place to see the health of your infrastructure and move from signal to action.</p></div><div className="dash-welcome-meta"><span className="dash-live-dot" />Live sync<div title={lastUpdated?.toLocaleString()}>{lastUpdated ? `Updated ${formatRelative(lastUpdated, now)}` : 'Syncing now'}</div></div></section>
        <section className="dash-metric-grid">
          {loading && !snapshot.hosts.length ? <>
            <SkeletonCard /><SkeletonCard /><SkeletonCard /><SkeletonCard />
          </> : <>
            <MetricCard label="Connected hosts" value={String(snapshot.hosts.length)} hint="Registered control planes" tone="cyan" icon="◫" sparkline={history.hosts} />
            <MetricCard label="Compute nodes" value={String(snapshot.nodes.length)} hint="Across your Proxmox fabric" tone="indigo" icon="◇" sparkline={history.nodes} />
            <MetricCard label="Running workloads" value={String(running)} hint={`${stopped} stopped`} tone="green" icon="▶" sparkline={history.running} />
            <MetricCard label="API health" value={text(value(snapshot.health, 'status'), 'unknown')} hint={text(value(snapshot.health, 'version'), 'StackWatch API')} tone="amber" icon="♥" sparkline={history.cpu.map((c) => Math.min(100, c))} />
          </>}
        </section>
        <section className="dash-grid-main"><article className="dash-panel dash-chart-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">Live telemetry</span><h3>Workload pressure</h3></div><span className="dash-panel-context">Current snapshot</span></div><TrendChart resources={snapshot.resources} /></article><article className="dash-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">Operations</span><h3>Infrastructure status</h3></div><span className="dash-panel-context">{snapshot.hosts.length} host{snapshot.hosts.length === 1 ? '' : 's'}</span></div><div className="dash-host-list">{snapshot.hosts.length ? snapshot.hosts.map((host) => <div className="dash-host-row" key={host.id}><span className="dash-host-icon">⌁</span><span className="dash-host-name"><strong>{host.name || 'Unnamed host'}</strong><small>{host.base_url}</small></span><StatusPill status={host.status || 'unknown'} /></div>) : <div className="dash-panel-empty">
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
        </div>}</div></article></section>
        <section className="dash-panel">
          <div className="dash-panel-head">
            <div><span className="dash-eyebrow">Compute inventory</span><h3>Workloads and resources</h3></div>
            <div className="dash-panel-controls">
              <div className="dash-search-input">
                <span className="dash-search-icon" aria-hidden="true">⌕</span>
                <input type="search" placeholder="Search workloads..." value={search} onChange={(e) => setSearch(e.target.value)} aria-label="Search workloads by name, type, or status" />
              </div>
              <Link className="dash-text-link" to="/proxmox">Open full workspace →</Link>
            </div>
          </div>
          {filteredResources.length ? <div className="dash-resource-grid">{filteredResources.slice(0, 12).map((resource, index) => <div className="dash-resource-card" key={String(resource.id || resource.vmid || index)}><div className="dash-resource-head"><span className="dash-resource-type">{text(resource.type, 'resource')}</span><StatusPill status={resource.status} /></div><strong>{text(resource.name, `Workload ${text(resource.vmid, String(index + 1))}`)}</strong><div className="dash-resource-meta"><span>CPU <b>{typeof resource.cpu === 'number' ? formatPercent(resource.cpu) : '—'}</b></span><span>RAM <b>{formatBytes(resource.mem)}</b></span></div></div>)}</div> : search ? <div className="dash-empty-inline"><span>⌕</span><div><strong>No workloads match &ldquo;{search}&rdquo;</strong><p>Try a different search term, or clear the search to see all {snapshot.resources.length} workload{snapshot.resources.length === 1 ? '' : 's'}.</p></div></div> : <div className="dash-empty-inline"><span>◇</span><div><strong>No workloads reported</strong><p>The dashboard will populate as soon as the connected host returns resource inventory.</p></div></div>}
        </section>
      </div>
    </main>
    <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
  </div>;
}
