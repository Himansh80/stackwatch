import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import StatusPill from '../components/shared/StatusPill';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import PipelineCard from '../components/shared/PipelineCard';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

interface Pipeline {
  id: string;
  provider: string;
  repo: string;
  branch?: string | null;
  commit_sha?: string | null;
  status: string;
  started_at: string;
  finished_at?: string | null;
  duration_ms?: number;
}

interface Deployment {
  id: string;
  pipeline_id: string;
  service_id: string;
  service_name?: string;
  environment: string;
  version: string;
  deployed_at: string;
}

type Tab = 'pipelines' | 'deployments';
type StatusFilter = 'all' | 'success' | 'failed' | 'running' | 'pending';
type EnvFilter = 'all' | 'production' | 'staging' | 'preview';

/**
 * CicdPage — Tier 7.7 (D8) observability surface at /cicd.
 *
 * Layout:
 *   - Top: 3 KpiCards (pipelines today / success rate / avg duration)
 *   - Middle: tabbed view (Pipelines / Deployments)
 *     - Pipelines tab: list of PipelineCards with status dropdown
 *     - Deployments tab: table (commit_sha | service | environment | version | deployed_at)
 *   - Filters are reset on tab switch.
 *
 * Sidebar nav: "CI/CD" added in AppSidebar under observability.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip so the
 * three cards enter with a 50ms gap. Reuses existing tokens + classes.
 */
export default function CicdPage() {
  const [tab, setTab] = useState<Tab>('pipelines');
  const [error, setError] = useState('');

  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  // Rolling history for KPI sparklines + trend chart (last 24 polls).
  const [pipelineHistory, setPipelineHistory] = useState<number[]>([]);
  const [failureHistory, setFailureHistory] = useState<number[]>([]);

  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [envFilter, setEnvFilter] = useState<EnvFilter>('all');

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view CI/CD.');
      return false;
    }
    return true;
  };

  const loadPipelines = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (statusFilter !== 'all') params.set('status', statusFilter);
      const qs = params.toString();
      const r = await api<{ pipelines?: Pipeline[] }>('GET', `/api/v1/cicd/pipelines${qs ? `?${qs}` : ''}`);
      setPipelines(r.pipelines || []);
      // Update rolling history for KPI sparklines + trend chart.
      const append = (prev: number[], next: number, max = 24) => {
        const out = [...prev, next];
        return out.length > max ? out.slice(out.length - max) : out;
      };
      setPipelineHistory((prev) => append(prev, (r.pipelines || []).length));
      const failed = (r.pipelines || []).filter((p) => String(p.status || '').toLowerCase() === 'failed').length;
      setFailureHistory((prev) => append(prev, failed));
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [statusFilter]);

  const loadDeployments = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (envFilter !== 'all') params.set('environment', envFilter);
      const qs = params.toString();
      const r = await api<{ deployments?: Deployment[] }>('GET', `/api/v1/cicd/deployments${qs ? `?${qs}` : ''}`);
      setDeployments(r.deployments || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [envFilter]);

  useEffect(() => {
    setError('');
    if (tab === 'pipelines') void loadPipelines();
    else void loadDeployments();
  }, [tab, loadPipelines, loadDeployments]);

  // KPIs computed client-side from the loaded pipelines list.
  const todayCutoff = Date.now() - 24 * 60 * 60 * 1000;
  const todayPipelines = pipelines.filter((p) => new Date(p.started_at).getTime() >= todayCutoff);
  const successPipelines = pipelines.filter((p) => (p.status || '').toLowerCase() === 'success');
  const successRatePct = pipelines.length === 0 ? 0 : Math.round((successPipelines.length / pipelines.length) * 100);
  const durations = pipelines.map((p) => p.duration_ms ?? 0).filter((d) => d > 0);
  const avgDurationMs = durations.length === 0 ? 0 : Math.round(durations.reduce((a, b) => a + b, 0) / durations.length);
  const avgDurationLabel = avgDurationMs === 0
    ? '—'
    : avgDurationMs < 60_000
      ? `${Math.round(avgDurationMs / 1000)}s`
      : `${Math.round(avgDurationMs / 60_000)}m`;

  const tabSummary = tab === 'pipelines'
    ? `${pipelines.length} pipeline${pipelines.length === 1 ? '' : 's'}`
    : `${deployments.length} deployment${deployments.length === 1 ? '' : 's'}`;

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Pipelines today"
              value={todayPipelines.length}
              status={todayPipelines.length > 0 ? 'up' : 'neutral'}
              accent="cyan"
              sparkline={pipelineHistory}
            />
            <KpiCard
              label="Success rate"
              value={`${successRatePct}%`}
              status={successRatePct >= 90 ? 'up' : successRatePct >= 70 ? 'stale' : 'crit'}
              accent={successRatePct >= 90 ? 'green' : successRatePct >= 70 ? 'amber' : 'red'}
            />
            <KpiCard
              label="Avg duration"
              value={avgDurationLabel}
              status={avgDurationMs === 0 ? 'neutral' : 'up'}
              accent="indigo"
            />
            {pipelineHistory.length > 1 && (
              <section className="cicd-trend">
                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Trend</span><h3>Pipelines</h3></div>
                    <span className="dash-panel-context">last {pipelineHistory.length} polls</span>
                  </div>
                  <TimeSeriesChart values={pipelineHistory} color="cyan" height={100} emptyMessage="Waiting for data…" />
                </article>
                <article className="dash-panel">
                  <div className="dash-panel-head">
                    <div><span className="dash-eyebrow">Trend</span><h3>Failures</h3></div>
                    <span className="dash-panel-context">last {failureHistory.length} polls</span>
                  </div>
                  <TimeSeriesChart values={failureHistory} color="red" height={100} emptyMessage="No failures yet" />
                </article>
              </section>
            )}
          </motion.div>

          {tab === 'pipelines' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Pipelines</span>
              <h2 className="dash-section-title">Recent CI/CD runs</h2>
              <div className="synth-filter-row">
                <label>
                  <span>Status</span>
                  <select
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                  >
                    <option value="all">All</option>
                    <option value="success">Success</option>
                    <option value="failed">Failed</option>
                    <option value="running">Running</option>
                    <option value="pending">Pending</option>
                  </select>
                </label>
              </div>
              {pipelines.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>⇄</span>}
                  headline="No pipelines match these filters"
                  subhead="Pipeline runs surface here when a webhook fires (GitHub / GitLab) or an external system POSTs to /api/v1/cicd/pipelines. Configure a webhook in your repo settings to start streaming."
                />
              ) : (
                <div className="pipeline-card-list">
                  {pipelines.map((p) => (
                    <PipelineCard key={p.id} pipeline={p} />
                  ))}
                </div>
              )}
            </section>
          ) : null}

          {tab === 'deployments' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Deployments</span>
              <h2 className="dash-section-title">Artifact deployments to APM services</h2>
              <div className="synth-filter-row">
                <label>
                  <span>Environment</span>
                  <select
                    value={envFilter}
                    onChange={(e) => setEnvFilter(e.target.value as EnvFilter)}
                  >
                    <option value="all">All</option>
                    <option value="production">Production</option>
                    <option value="staging">Staging</option>
                    <option value="preview">Preview</option>
                  </select>
                </label>
              </div>
              {deployments.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>⇉</span>}
                  headline="No deployments yet"
                  subhead="Deployments surface here when a pipeline creates one via POST /api/v1/cicd/deployments (linking an APM service + version + environment)."
                />
              ) : (
                <table className="dash-table">
                  <thead>
                    <tr>
                      <th>Pipeline</th>
                      <th>Service</th>
                      <th>Environment</th>
                      <th>Version</th>
                      <th>Deployed at</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deployments.map((d) => (
                      <tr key={d.id}>
                        <td><code>{d.pipeline_id.slice(0, 8)}</code></td>
                        <td><strong>{d.service_name || d.service_id.slice(0, 8)}</strong></td>
                        <td>
                          <StatusPill status="unknown" label={d.environment} size="sm" />
                        </td>
                        <td><code>{d.version}</code></td>
                        <td><small>{new Date(d.deployed_at).toLocaleString()}</small></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          ) : null}
        </motion.div>
  );
}