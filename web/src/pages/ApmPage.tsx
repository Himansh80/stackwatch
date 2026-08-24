import { useCallback, useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import KpiCard from '../components/shared/KpiCard';
import TraceSummary, { TraceSummaryService } from '../components/shared/TraceSummary';
import EmptyState from '../components/shared/EmptyState';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

interface Service {
  id: string;
  name: string;
  language: string;
  framework: string;
  environment: string;
  created_at: string;
}

interface ServiceDetail {
  id: string;
  name: string;
  language: string;
  framework: string;
  environment: string;
  created_at: string;
  request_rate: number;
  error_rate: number;
  p95_latency: number;
}

interface Deployment {
  id: string;
  service_id: string;
  service_name: string;
  environment: string;
  version: string;
  commit_sha: string;
  deployed_at: string;
  rolled_back: boolean;
}

interface ServiceMapResponse {
  nodes: Array<{ id: string; name: string; request_rate: number }>;
  edges: Array<{ source: string; target: string; request_count: number; error_count: number }>;
}

/**
 * ApmPage — APM overview at /apm.
 *
 * Top: KPI strip (services count / traces/min / errors/min / p95 global)
 * Middle: grid of TraceSummary cards (one per service)
 * Bottom: deployments table
 *
 * Sidebar nav item: "APM" → /apm (set in AppSidebar).
 *
 * Motion: pageEnter on the page, kpiStagger on the KPI strip.
 */
export default function ApmPage() {
  const nav = useNavigate();
  const logout = useLogout();
  const [services, setServices] = useState<Service[]>([]);
  const [serviceMetrics, setServiceMetrics] = useState<Record<string, Pick<ServiceDetail, 'request_rate' | 'error_rate' | 'p95_latency'>>>({});
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [serviceMap, setServiceMap] = useState<ServiceMapResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [registerOpen, setRegisterOpen] = useState(false);
  const [registerForm, setRegisterForm] = useState({ name: '', language: 'go', framework: 'gin', environment: 'production' });
  const [registerBusy, setRegisterBusy] = useState(false);
  const [registerErr, setRegisterErr] = useState('');

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view APM.');
      setLoading(false);
      return;
    }
    setError('');
    setLoading(true);
    try {
      const [svcRes, depRes, mapRes] = await Promise.all([
        api<{ services?: Service[] }>('GET', '/api/v1/apm/services'),
        api<{ deployments?: Deployment[] }>('GET', '/api/v1/apm/deployments?limit=20'),
        api<ServiceMapResponse>('GET', '/api/v1/apm/service-map').catch(() => ({ nodes: [], edges: [] }) as ServiceMapResponse),
      ]);
      const svcs = svcRes.services || [];
      setServices(svcs);
      setDeployments(depRes.deployments || []);
      setServiceMap(mapRes);
      // Best-effort per-service metrics (fire-and-forget — empty if any fail).
      const metricsEntries = await Promise.all(
        svcs.slice(0, 25).map(async (s) => {
          try {
            const detail = await api<ServiceDetail>('GET', `/api/v1/apm/services/${s.id}`);
            return [s.id, {
              request_rate: detail.request_rate,
              error_rate: detail.error_rate,
              p95_latency: detail.p95_latency,
            }] as const;
          } catch {
            return [s.id, { request_rate: 0, error_rate: 0, p95_latency: 0 }] as const;
          }
        }),
      );
      setServiceMetrics(Object.fromEntries(metricsEntries));
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message || 'Unable to load APM data.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const submitRegister = async (event: FormEvent) => {
    event.preventDefault();
    setRegisterErr('');
    if (!registerForm.name.trim()) {
      setRegisterErr('Service name is required.');
      return;
    }
    setRegisterBusy(true);
    try {
      await api('POST', '/api/v1/apm/services', {
        name: registerForm.name.trim(),
        language: registerForm.language.trim(),
        framework: registerForm.framework.trim(),
        environment: registerForm.environment.trim(),
      });
      setRegisterOpen(false);
      setRegisterForm({ name: '', language: 'go', framework: 'gin', environment: 'production' });
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message || 'Unable to register service.';
      setRegisterErr(msg);
    } finally {
      setRegisterBusy(false);
    }
  };

  // Aggregate KPIs from per-service metrics.
  const totals = useMemo(() => {
    const m = Object.values(serviceMetrics);
    const totalReqPerSec = m.reduce((acc, x) => acc + x.request_rate, 0);
    const totalReqPerMin = totalReqPerSec * 60;
    const totalErrorsPerMin = m.reduce((acc, x) => acc + x.request_rate * x.error_rate, 0) * 60;
    const p95Values = m.map((x) => x.p95_latency).filter((v) => v > 0).sort((a, b) => a - b);
    const globalP95 = p95Values.length ? p95Values[Math.floor(p95Values.length * 0.95)] : 0;
    return {
      services: services.length,
      tracesPerMin: Math.round(totalReqPerMin),
      errorsPerMin: totalErrorsPerMin < 1 ? Number(totalErrorsPerMin.toFixed(2)) : Math.round(totalErrorsPerMin),
      p95: Math.round(globalP95),
    };
  }, [serviceMetrics, services.length]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="apm"
        onLogout={logout}
        show={['dashboard', 'billing', 'profile', 'settings', 'proxmox', 'truenas', 'apm']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Observability</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">APM</strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">
                  {services.length} service{services.length === 1 ? '' : 's'}
                  {' · '}
                  {serviceMap?.edges?.length ?? 0} edge{(serviceMap?.edges?.length ?? 0) === 1 ? '' : 's'}
                </span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <button
              type="button"
              className="sw-button sw-button-primary"
              onClick={() => setRegisterOpen((v) => !v)}
            >
              {registerOpen ? 'Close' : 'Register service'}
            </button>
          </div>
        </header>

        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {registerOpen ? (
            <section className="dash-section">
              <span className="dash-eyebrow">New service</span>
              <h2 className="dash-section-title">Register an APM service</h2>
              <form className="apm-register-form" onSubmit={submitRegister}>
                <label>
                  <span>Name</span>
                  <input
                    type="text"
                    value={registerForm.name}
                    onChange={(e) => setRegisterForm({ ...registerForm, name: e.target.value })}
                    placeholder="checkout-api"
                    required
                    maxLength={128}
                  />
                </label>
                <label>
                  <span>Language</span>
                  <input
                    type="text"
                    value={registerForm.language}
                    onChange={(e) => setRegisterForm({ ...registerForm, language: e.target.value })}
                    placeholder="go"
                    maxLength={64}
                  />
                </label>
                <label>
                  <span>Framework</span>
                  <input
                    type="text"
                    value={registerForm.framework}
                    onChange={(e) => setRegisterForm({ ...registerForm, framework: e.target.value })}
                    placeholder="gin"
                    maxLength={64}
                  />
                </label>
                <label>
                  <span>Environment</span>
                  <input
                    type="text"
                    value={registerForm.environment}
                    onChange={(e) => setRegisterForm({ ...registerForm, environment: e.target.value })}
                    placeholder="production"
                    maxLength={64}
                  />
                </label>
                {registerErr ? <p className="apm-form-error">{registerErr}</p> : null}
                <div className="apm-form-actions">
                  <button type="submit" className="sw-button sw-button-primary" disabled={registerBusy}>
                    {registerBusy ? 'Registering…' : 'Register'}
                  </button>
                </div>
              </form>
            </section>
          ) : null}

          <section className="dash-section">
            <span className="dash-eyebrow">Last 5 minutes</span>
            <h2 className="dash-section-title">Service health at a glance</h2>
            <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">
              <KpiCard label="Services" value={totals.services} accent="indigo" />
              <KpiCard label="Traces/min" value={totals.tracesPerMin} accent="cyan" />
              <KpiCard label="Errors/min" value={totals.errorsPerMin} accent="red" />
              <KpiCard label="P95 global" value={totals.p95 > 0 ? `${totals.p95}ms` : '—'} accent="amber" />
            </motion.div>
          </section>

          {error ? (
            <div className="dash-error" role="alert">
              {error}
            </div>
          ) : null}

          <section className="dash-section">
            <span className="dash-eyebrow">Services</span>
            <h2 className="dash-section-title">Registered APM services</h2>
            {loading ? (
              <p className="dash-section-lede">Loading services…</p>
            ) : services.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◴</span>}
                headline="No services registered yet"
                subhead="Register a service to start collecting traces, then POST spans via /api/v1/apm/traces."
                cta={{ label: 'Register your first service', onClick: () => setRegisterOpen(true) }}
              />
            ) : (
              <div className="apm-service-grid">
                {services.map((s) => {
                  const metrics = serviceMetrics[s.id] || { request_rate: 0, error_rate: 0, p95_latency: 0 };
                  const card: TraceSummaryService = {
                    id: s.id,
                    name: s.name,
                    language: s.language || '',
                    framework: s.framework || '',
                    request_rate: metrics.request_rate,
                    error_rate: metrics.error_rate,
                    p95_latency: metrics.p95_latency,
                  };
                  return (
                    <TraceSummary
                      key={s.id}
                      service={card}
                      onClick={(svc) => nav(`/apm/service?id=${encodeURIComponent(svc.id)}`)}
                    />
                  );
                })}
              </div>
            )}
          </section>

          {serviceMap && serviceMap.nodes.length > 0 ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Service map</span>
              <h2 className="dash-section-title">Live trace flow</h2>
              <div className="apm-servicemap" role="img" aria-label="Service map">
                {serviceMap.nodes.map((node) => (
                  <div key={node.id} className="apm-servicemap-node" title={`${node.name} — ${node.request_rate.toFixed(2)} req/s`}>
                    <strong>{node.name}</strong>
                    <small>{node.request_rate.toFixed(2)} req/s</small>
                  </div>
                ))}
                {serviceMap.edges.map((edge, idx) => {
                  const src = serviceMap.nodes.find((n) => n.id === edge.source);
                  const tgt = serviceMap.nodes.find((n) => n.id === edge.target);
                  if (!src || !tgt) return null;
                  return (
                    <div key={`${edge.source}-${edge.target}-${idx}`} className="apm-servicemap-edge">
                      {src.name} → {tgt.name}: {edge.request_count} req
                      {edge.error_count > 0 ? ` (${edge.error_count} err)` : ''}
                    </div>
                  );
                })}
              </div>
            </section>
          ) : null}

          <section className="dash-section">
            <span className="dash-eyebrow">Deployments</span>
            <h2 className="dash-section-title">Recent deployments</h2>
            {deployments.length === 0 ? (
              <p className="dash-section-lede">No deployments recorded yet. POST to <code>/api/v1/apm/deployments</code> to track release markers.</p>
            ) : (
              <table className="dash-table apm-deployments">
                <thead>
                  <tr>
                    <th>Service</th>
                    <th>Environment</th>
                    <th>Version</th>
                    <th>Commit</th>
                    <th>Deployed</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {deployments.map((d) => (
                    <tr key={d.id}>
                      <td>{d.service_name}</td>
                      <td>{d.environment}</td>
                      <td>{d.version}</td>
                      <td><code>{d.commit_sha ? d.commit_sha.slice(0, 7) : '—'}</code></td>
                      <td>{d.deployed_at ? new Date(d.deployed_at).toLocaleString() : '—'}</td>
                      <td>
                        <span className={`apm-pill ${d.rolled_back ? 'apm-pill-error' : 'apm-pill-good'}`}>
                          {d.rolled_back ? 'rolled back' : 'live'}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        </motion.div>
      </main>
    </div>
  );
}
