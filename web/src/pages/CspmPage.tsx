import { useCallback, useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import CspmSeverityBadge from '../components/shared/CspmSeverityBadge';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import { motion, kpiStagger, pageEnter } from '../lib/motion';

interface CspmResource {
  id: string;
  provider: string;
  resource_type: string;
  resource_id: string;
  name: string;
  region?: string;
  last_scanned_at: string;
}

interface CspmFinding {
  id: string;
  resource_id: string;
  severity: string;
  finding_type: string;
  description: string;
  remediation?: string;
  detected_at: string;
  resolved_at?: string | null;
}

type SevFilter = 'all' | 'critical' | 'high' | 'medium' | 'low' | 'info';
type ResolvedFilter = 'open' | 'resolved' | 'all';

/**
 * CspmPage — Tier 7.6 (D7) observability surface at /cspm.
 *
 * Layout:
 *   - Top: 2 KpiCards (total resources / total findings)
 *   - Middle: 2 tables (resources / findings) stacked
 *   - Findings table filter: severity + resolved/open
 *
 * Sidebar nav: "CSPM" added in AppSidebar under observability.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip so the
 * two cards enter with a 50ms gap. Reuses existing tokens.
 */
export default function CspmPage() {
  const [error, setError] = useState('');
  const [resources, setResources] = useState<CspmResource[]>([]);
  const [findings, setFindings] = useState<CspmFinding[]>([]);
  const [sevFilter, setSevFilter] = useState<SevFilter>('all');
  const [resolvedFilter, setResolvedFilter] = useState<ResolvedFilter>('open');

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view CSPM.');
      return false;
    }
    return true;
  };

  const loadResources = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ resources?: CspmResource[] }>('GET', '/api/v1/cspm/resources');
      setResources(r.resources || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadFindings = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (sevFilter !== 'all') params.set('severity', sevFilter);
      if (resolvedFilter !== 'all') {
        params.set('resolved', resolvedFilter === 'resolved' ? 'true' : 'false');
      }
      const qs = params.toString();
      const r = await api<{ findings?: CspmFinding[] }>(
        'GET',
        `/api/v1/cspm/findings${qs ? `?${qs}` : ''}`,
      );
      setFindings(r.findings || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [sevFilter, resolvedFilter]);

  useEffect(() => {
    setError('');
    void loadResources();
  }, [loadResources]);

  useEffect(() => {
    setError('');
    void loadFindings();
  }, [loadFindings]);

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard label="Tracked resources" value={resources.length} status="up" accent="cyan" />
            <KpiCard
              label="Open findings"
              value={findings.filter((f) => !f.resolved_at).length}
              status={findings.length > 0 ? 'crit' : 'neutral'}
              accent={findings.length > 0 ? 'red' : 'green'}
            />
          </motion.div>

          <section className="dash-section">
            <span className="dash-eyebrow">Resources</span>
            <h2 className="dash-section-title">Cloud resources enrolled in posture management</h2>
            {resources.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◈</span>}
                headline="No resources enrolled yet"
                subhead="Resources appear here once Proxmox / TrueNAS / AWS / GCP / Azure accounts are connected. CSPM scans run on the registered resource set."
              />
            ) : (
              <table className="dash-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Provider</th>
                    <th>Type</th>
                    <th>Region</th>
                    <th>Last scanned</th>
                  </tr>
                </thead>
                <tbody>
                  {resources.map((r) => (
                    <tr key={r.id}>
                      <td><strong>{r.name}</strong></td>
                      <td><code>{r.provider}</code></td>
                      <td><code>{r.resource_type}</code></td>
                      <td>{r.region || '—'}</td>
                      <td><small>{new Date(r.last_scanned_at).toLocaleString()}</small></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>

          <section className="dash-section">
            <span className="dash-eyebrow">Findings</span>
            <h2 className="dash-section-title">Misconfigurations detected by CSPM</h2>
            <div className="synth-filter-row">
              <label>
                <span>Severity</span>
                <select
                  value={sevFilter}
                  onChange={(e) => setSevFilter(e.target.value as SevFilter)}
                >
                  <option value="all">All</option>
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                  <option value="info">Info</option>
                </select>
              </label>
              <label>
                <span>Status</span>
                <select
                  value={resolvedFilter}
                  onChange={(e) => setResolvedFilter(e.target.value as ResolvedFilter)}
                >
                  <option value="open">Open</option>
                  <option value="resolved">Resolved</option>
                  <option value="all">All</option>
                </select>
              </label>
            </div>
            {findings.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◔</span>}
                headline="No findings match these filters"
                subhead="CSPM scans resources for misconfigurations (open ports, default credentials, unencrypted volumes). Findings surface here as scans complete."
              />
            ) : (
              <table className="dash-table">
                <thead>
                  <tr>
                    <th>Resource</th>
                    <th>Severity</th>
                    <th>Finding</th>
                    <th>Description</th>
                    <th>Remediation</th>
                  </tr>
                </thead>
                <tbody>
                  {findings.map((f) => (
                    <tr key={f.id}>
                      <td><code>{f.resource_id}</code></td>
                      <td><CspmSeverityBadge severity={f.severity} /></td>
                      <td><code>{f.finding_type}</code></td>
                      <td>{f.description}</td>
                      <td>{f.remediation || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        </motion.div>
  );
}