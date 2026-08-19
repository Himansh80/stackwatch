import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api, clearToken, health, me } from '../lib/api';
import { listFrom, objectFrom, ProxmoxHost, ProxmoxResource, pxGet, formatBytes, formatPercent } from '../lib/proxmox';

type Json = Record<string, unknown>;

type Snapshot = {
  hosts: ProxmoxHost[];
  nodes: Json[];
  resources: ProxmoxResource[];
  health: Json | null;
  user: Json | null;
  tenant: Json | null;
};

const emptySnapshot: Snapshot = { hosts: [], nodes: [], resources: [], health: null, user: null, tenant: null };

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

function MetricCard({ label, value: metric, hint, tone, icon }: { label: string; value: string; hint: string; tone: string; icon: string }) {
  return <article className={`dash-metric dash-metric-${tone}`}>
    <div className="dash-metric-top"><span className="dash-metric-icon">{icon}</span><span className="dash-metric-label">{label}</span></div>
    <strong>{metric}</strong>
    <span className="dash-metric-hint">{hint}</span>
  </article>;
}

function TrendChart({ resources }: { resources: ProxmoxResource[] }) {
  const values = resources.map((row) => Number(row.cpu)).filter((number) => Number.isFinite(number));
  if (!values.length) {
    return <div className="dash-chart-empty"><span>⌁</span><strong>Waiting for live resource data</strong><small>Connect a Proxmox host to populate this view.</small></div>;
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
  const navigate = useNavigate();
  const [snapshot, setSnapshot] = useState<Snapshot>(emptySnapshot);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [now, setNow] = useState<Date>(() => new Date());

  async function loadDashboard() {
    setError('');
    try {
      const [healthData, meData, hostsData] = await Promise.all([
        health(),
        me(),
        api<{ hosts?: ProxmoxHost[] }>('GET', '/api/v1/proxmox/hosts'),
      ]);
      const hosts = hostsData.hosts || [];
      const primary = hosts[0];
      const [nodesPayload, resourcePayload] = primary
        ? await Promise.all([pxGet(primary.id, '/nodes'), pxGet(primary.id, '/cluster/resources')])
        : [{}, {}];
      setSnapshot({
        hosts,
        nodes: listFrom(nodesPayload, 'nodes'),
        resources: listFrom(resourcePayload, 'resources', 'vms') as ProxmoxResource[],
        health: objectFrom(healthData),
        user: objectFrom(meData.user),
        tenant: objectFrom(meData.tenant),
      });
      setLastUpdated(new Date());
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
    };
  }, []);

  const running = useMemo(() => snapshot.resources.filter(isRunning).length, [snapshot.resources]);
  const stopped = useMemo(() => snapshot.resources.filter((row) => text(row.status, '').toLowerCase() === 'stopped').length, [snapshot.resources]);
  const tenantName = text(value(snapshot.tenant, 'name'), 'Workspace');
  const userName = text(value(snapshot.user, 'full_name'), text(value(snapshot.user, 'email'), 'Operator'));

  function logout() {
    clearToken();
    navigate('/login');
  }

  return <div className="dash-app">
    <aside className="dash-sidebar">
      <Link className="dash-brand" to="/dashboard"><span className="dash-brand-mark">S</span><span><strong>StackWatch</strong><small>Infrastructure control plane</small></span></Link>
      <div className="dash-nav-section"><span className="dash-nav-heading">Workspace</span>
        <Link className="dash-nav-item dash-nav-active" to="/dashboard"><span>⌂</span>Overview</Link>
        <Link className="dash-nav-item" to="/dashboard"><span>◫</span>Servers <em>{snapshot.hosts.length || ''}</em></Link>
        <Link className="dash-nav-item" to="/dashboard"><span>⌁</span>Metrics</Link>
        <Link className="dash-nav-item" to="/dashboard"><span>△</span>Alerts</Link>
      </div>
      <div className="dash-nav-section"><span className="dash-nav-heading">Infrastructure</span>
        <Link className="dash-nav-item" to="/proxmox"><span>◈</span>Proxmox</Link>
        <Link className="dash-nav-item" to="/truenas"><span>▤</span>TrueNAS</Link>
        <Link className="dash-nav-item" to="/dashboard"><span>⌘</span>Terminal</Link>
      </div>
      <div className="dash-sidebar-bottom"><div className="dash-connection"><span className="dash-live-dot" />Control plane online<small>{text(value(snapshot.health, 'version'), 'StackWatch API')}</small></div><button className="dash-sidebar-logout" onClick={logout}>↪ Sign out</button></div>
    </aside>
    <main className="dash-main">
      <header className="dash-topbar"><div className="dash-greeting"><span className="dash-greeting-eyebrow">Hello, {firstNameOf(userName)}</span><div className="dash-greeting-row"><strong className="dash-greeting-text">{greetingFor(now)}, {firstNameOf(userName)}.</strong><span className="dash-greeting-clock"><span className="dash-greeting-clock-time">{formatClock(now)}</span><span className="dash-greeting-clock-dot" /><span className="dash-greeting-clock-day">{formatDayLabel(now)}</span></span></div></div><div className="dash-top-actions"><button className="dash-icon-button" onClick={() => void loadDashboard()} aria-label="Refresh dashboard">↻</button><div className="dash-user"><span className="dash-avatar">{userName.charAt(0).toUpperCase()}</span><span><strong>{userName}</strong><small>{tenantName}</small></span></div></div></header>
      <div className="dash-content">
        {error && <div className="dash-error"><strong>Live data unavailable</strong><span>{error}</span><button onClick={() => void loadDashboard()}>Retry</button></div>}
        <section className="dash-welcome"><div><span className="dash-eyebrow">Infrastructure overview</span><h2>Good to see you, {userName.split(' ')[0]}.</h2><p>One place to see the health of your infrastructure and move from signal to action.</p></div><div className="dash-welcome-meta"><span className="dash-live-dot" />Live sync<div>{lastUpdated ? `Updated ${lastUpdated.toLocaleTimeString()}` : 'Syncing now'}</div></div></section>
        <section className="dash-metric-grid">
          <MetricCard label="Connected hosts" value={loading ? '—' : String(snapshot.hosts.length)} hint="Registered control planes" tone="cyan" icon="◫" />
          <MetricCard label="Compute nodes" value={loading ? '—' : String(snapshot.nodes.length)} hint="Across your Proxmox fabric" tone="indigo" icon="◇" />
          <MetricCard label="Running workloads" value={loading ? '—' : String(running)} hint={`${stopped} stopped`} tone="green" icon="▶" />
          <MetricCard label="API health" value={loading ? '—' : text(value(snapshot.health, 'status'), 'unknown')} hint={text(value(snapshot.health, 'version'), 'StackWatch API')} tone="amber" icon="♥" />
        </section>
        <section className="dash-grid-main"><article className="dash-panel dash-chart-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">Live telemetry</span><h3>Workload pressure</h3></div><span className="dash-panel-context">Current snapshot</span></div><TrendChart resources={snapshot.resources} /></article><article className="dash-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">Operations</span><h3>Infrastructure status</h3></div><span className="dash-panel-context">{snapshot.hosts.length} hosts</span></div><div className="dash-host-list">{snapshot.hosts.length ? snapshot.hosts.map((host) => <div className="dash-host-row" key={host.id}><span className="dash-host-icon">⌁</span><span className="dash-host-name"><strong>{host.name || 'Unnamed host'}</strong><small>{host.base_url}</small></span><StatusPill status={host.status || 'unknown'} /></div>) : <div className="dash-panel-empty"><strong>No infrastructure connected yet</strong><span>Register a Proxmox host to see your environment here.</span><Link to="/dashboard">Open Proxmox workspace →</Link></div>}</div></article></section>
        <section className="dash-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">Compute inventory</span><h3>Workloads and resources</h3></div><Link className="dash-text-link" to="/dashboard">Open full workspace →</Link></div>{snapshot.resources.length ? <div className="dash-resource-grid">{snapshot.resources.slice(0, 8).map((resource, index) => <div className="dash-resource-card" key={String(resource.id || resource.vmid || index)}><div className="dash-resource-head"><span className="dash-resource-type">{text(resource.type, 'resource')}</span><StatusPill status={resource.status} /></div><strong>{text(resource.name, `Workload ${text(resource.vmid, String(index + 1))}`)}</strong><div className="dash-resource-meta"><span>CPU <b>{typeof resource.cpu === 'number' ? formatPercent(resource.cpu) : '—'}</b></span><span>RAM <b>{formatBytes(resource.mem)}</b></span></div></div>)}</div> : <div className="dash-empty-inline"><span>◇</span><div><strong>No workloads reported</strong><p>The dashboard will populate as soon as the connected host returns resource inventory.</p></div></div>}</section>
      </div>
    </main>
  </div>;
}
