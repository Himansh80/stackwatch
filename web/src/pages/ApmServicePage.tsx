import { useCallback, useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ApiError, api, getToken } from '../lib/api';
import FlameGraph, { FlameSpan } from '../components/shared/FlameGraph';
import StatusPill from '../components/shared/StatusPill';
import { motion, pageEnter } from '../lib/motion';

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

interface FlameGraphResponse {
  service_id: string;
  spans: FlameSpan[];
  total: number;
  cap: number;
}

interface TraceRow {
  trace_id: string;
  root_span_id: string;
  service_id: string;
  service_name: string;
  duration_us: number;
  status: string;
  started_at: string;
}

/**
 * ApmServicePage — service detail at /apm/service?id=<uuid>.
 *
 * Top: service detail header (name, language, framework, env, metrics)
 * Middle: FlameGraph of latest traces
 * Bottom: recent traces table
 */
export default function ApmServicePage() {
  const [params] = useSearchParams();
  const serviceId = params.get('id') || '';
  const [detail, setDetail] = useState<ServiceDetail | null>(null);
  const [flame, setFlame] = useState<FlameGraphResponse | null>(null);
  const [traces, setTraces] = useState<TraceRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    if (!serviceId) {
      setError('Missing service id.');
      setLoading(false);
      return;
    }
    if (!getToken()) {
      setError('Sign in to view service detail.');
      setLoading(false);
      return;
    }
    setError('');
    setLoading(true);
    try {
      const [det, fg] = await Promise.all([
        api<ServiceDetail>('GET', `/api/v1/apm/services/${encodeURIComponent(serviceId)}`),
        api<FlameGraphResponse>('GET', `/api/v1/apm/services/${encodeURIComponent(serviceId)}/flame-graph`).catch(() => ({ service_id: serviceId, spans: [], total: 0, cap: 500 })),
      ]);
      setDetail(det);
      setFlame(fg);
      // Traces — pick recent spans for this service then group by trace_id.
      const spansResp = await api<{ spans?: Array<Record<string, unknown>> }>('GET', '/api/v1/traces?limit=200').catch(() => ({ spans: [] } as { spans?: Array<Record<string, unknown>> }));
      const svcSpans = (spansResp.spans || []).filter((row) => row.service === det.name);
      const seen = new Set<string>();
      const grouped: TraceRow[] = [];
      for (const row of svcSpans) {
        const tid = String(row.trace_id || '');
        if (!tid || seen.has(tid)) continue;
        seen.add(tid);
        grouped.push({
          trace_id: tid,
          root_span_id: String(row.span_id || ''),
          service_id: serviceId,
          service_name: det.name,
          duration_us: Number(row.duration_ms || 0) * 1000,
          status: String(row.status || 'ok'),
          started_at: String(row.ts || ''),
        });
        if (grouped.length >= 20) break;
      }
      setTraces(grouped);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message || 'Unable to load service.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [serviceId]);

  useEffect(() => { void load(); }, [load]);

  function formatDuration(us: number): string {
    if (!Number.isFinite(us) || us <= 0) return '—';
    if (us >= 1_000_000) return `${(us / 1_000_000).toFixed(2)}s`;
    if (us >= 1_000) return `${(us / 1_000).toFixed(1)}ms`;
    return `${us}µs`;
  }

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? (
            <div className="dash-error" role="alert">{error}</div>
          ) : null}

          {detail ? (
            <>
              <section className="dash-section">
                <div className="apm-service-meta">
                  <span className="apm-pill apm-pill-language">{detail.language || 'unknown'}</span>
                  <span className="apm-pill apm-pill-framework">{detail.framework || '—'}</span>
                  <span className="apm-pill apm-pill-env">{detail.environment || '—'}</span>
                  <span className="apm-pill apm-pill-neutral">
                    {detail.request_rate.toFixed(2)} req/s
                  </span>
                  <span className={`apm-pill ${detail.error_rate >= 0.05 ? 'apm-pill-error' : 'apm-pill-good'}`}>
                    {(detail.error_rate * 100).toFixed(2)}% err
                  </span>
                  <span className={`apm-pill ${detail.p95_latency >= 500 ? 'apm-pill-warn' : 'apm-pill-good'}`}>
                    p95 {detail.p95_latency > 0 ? `${Math.round(detail.p95_latency)}ms` : '—'}
                  </span>
                </div>
              </section>

              <section className="dash-section">
                <span className="dash-eyebrow">Flame graph</span>
                <h2 className="dash-section-title">Latest trace spans</h2>
                {flame && flame.spans.length > 0 ? (
                  <FlameGraph spans={flame.spans} />
                ) : (
                  <p className="dash-section-lede">
                    No spans yet. Ingest a trace via <code>POST /api/v1/apm/traces</code> with this service as the root.
                  </p>
                )}
              </section>

              <section className="dash-section">
                <span className="dash-eyebrow">Recent traces</span>
                <h2 className="dash-section-title">Trace summaries</h2>
                {traces.length === 0 ? (
                  <p className="dash-section-lede">
                    No traces captured yet for <strong>{detail.name}</strong>.
                  </p>
                ) : (
                  <table className="dash-table apm-traces">
                    <thead>
                      <tr>
                        <th>Trace ID</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Started</th>
                      </tr>
                    </thead>
                    <tbody>
                      {traces.map((t) => (
                        <tr key={t.trace_id}>
                          <td><code>{t.trace_id.slice(0, 12)}…</code></td>
                          <td><StatusPill status={t.status} /></td>
                          <td>{formatDuration(t.duration_us)}</td>
                          <td>{t.started_at ? new Date(t.started_at).toLocaleString() : '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </section>
            </>
          ) : loading ? (
            <p className="dash-section-lede">Loading service detail…</p>
          ) : null}
        </motion.div>
  );
}
