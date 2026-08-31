import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import EmptyState from '../components/shared/EmptyState';
import ThreatCard from '../components/shared/ThreatCard';
import ComplianceBar from '../components/shared/ComplianceBar';
import KpiCard from '../components/shared/KpiCard';
import Select from '../components/shared/Select';
import StatusPill from '../components/shared/StatusPill';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import { motion, pageEnter, kpiStagger } from '../lib/motion';

interface Threat {
  id: string;
  severity: string;
  threat_type: string;
  source_ip?: string | null;
  user_id?: string | null;
  description: string;
  detected_at: string;
  resolved_at?: string | null;
}

interface ComplianceRule {
  rule_id: string;
  description: string;
  severity: string;
  remediation?: string;
  status: string;
  evidence?: string;
}

interface ComplianceFramework {
  framework: string;
  total_rules: number;
  pass_count: number;
  fail_count: number;
  unknown_count: number;
  rules: ComplianceRule[];
}

interface SiemEvent {
  id: string;
  event_type: string;
  severity: string;
  message: string;
  source: string;
  occurred_at: string;
}

type Tab = 'threats' | 'compliance' | 'siem' | 'audit';

/**
 * SecurityPage — Tier 7.5 (D6) observability surface at /security.
 *
 * Four tabs:
 *   - Threats   → list of ThreatCard (filter by severity + resolved)
 *   - Compliance → 4 ComplianceBar widgets (PCI/SOC2/GDPR/HIPAA) + rule list
 *   - SIEM       → list of recent SIEM events (filter by severity)
 *   - Audit      → placeholder linking to existing /audit (future phase)
 *
 * Sidebar nav: "Security" added in AppSidebar under observability.
 *
 * Motion: pageEnter on the page. ThreatCard uses tokens (no hex).
 */
export default function SecurityPage() {
  const [tab, _setTab] = useState<Tab>('threats');
  const [error, setError] = useState('');

  const [threats, setThreats] = useState<Threat[]>([]);
  const [threatFilter, setThreatFilter] = useState<'all' | 'critical' | 'high' | 'medium' | 'low'>('all');
  const [threatResolved, setThreatResolved] = useState<'all' | 'open' | 'resolved'>('open');

  const [frameworks, setFrameworks] = useState<ComplianceFramework[]>([]);
  const [expandedFw, setExpandedFw] = useState<string | null>(null);

  const [siem, setSiem] = useState<SiemEvent[]>([]);
  const [siemSev, setSiemSev] = useState<'all' | 'critical' | 'high' | 'medium' | 'low'>('all');

  // Rolling history for KPI sparklines + threat trend chart (24 polls).
  const [threatCountHistory, setThreatCountHistory] = useState<number[]>([]);
  const [criticalHistory, setCriticalHistory] = useState<number[]>([]);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Security.');
      return false;
    }
    return true;
  };

  const loadThreats = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (threatFilter !== 'all') params.set('severity', threatFilter);
      if (threatResolved !== 'all') params.set('resolved', threatResolved === 'resolved' ? 'true' : 'false');
      const qs = params.toString();
      const r = await api<{ threats?: Threat[] }>('GET', `/api/v1/security/threats${qs ? `?${qs}` : ''}`);
      const rows = r.threats || [];
      setThreats(rows);
      // Update rolling history (always track total + critical, regardless of filter).
      const append = (prev: number[], next: number, max = 24) => {
        const out = [...prev, next];
        return out.length > max ? out.slice(out.length - max) : out;
      };
      const critical = rows.filter((t) => (t.severity || '').toLowerCase() === 'critical').length;
      if (threatResolved !== 'resolved') {
        setThreatCountHistory((prev) => append(prev, rows.length));
        setCriticalHistory((prev) => append(prev, critical));
      }
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [threatFilter, threatResolved]);

  const loadCompliance = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ frameworks?: ComplianceFramework[] }>('GET', '/api/v1/security/compliance');
      setFrameworks(r.frameworks || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadSiem = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const params = new URLSearchParams();
      if (siemSev !== 'all') params.set('severity', siemSev);
      const qs = params.toString();
      const r = await api<{ events?: SiemEvent[] }>('GET', `/api/v1/security/siem${qs ? `?${qs}` : ''}`);
      setSiem(r.events || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [siemSev]);

  useEffect(() => {
    setError('');
    if (tab === 'threats') void loadThreats();
    else if (tab === 'compliance') void loadCompliance();
    else if (tab === 'siem') void loadSiem();
  }, [tab, loadThreats, loadCompliance, loadSiem]);

  const _tabSummary = (() => {
    if (tab === 'threats') return `${threats.length} threat${threats.length === 1 ? '' : 's'}`;
    if (tab === 'compliance') return `${frameworks.length} framework${frameworks.length === 1 ? '' : 's'}`;
    if (tab === 'siem') return `${siem.length} event${siem.length === 1 ? '' : 's'}`;
    return 'legacy audit log';
  })();

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          {tab === 'threats' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Threats</span>
              <h2 className="dash-section-title">Active security threats</h2>
              <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">
                <KpiCard label="Open threats" value={threats.filter((t) => !(t as any).resolved).length} status={threats.length > 0 ? 'crit' : 'up'} accent="red" sparkline={threatCountHistory} />
                <KpiCard label="Critical" value={threats.filter((t) => (t.severity || '').toLowerCase() === 'critical').length} status={threats.filter((t) => (t.severity || '').toLowerCase() === 'critical').length > 0 ? 'crit' : 'up'} accent="red" sparkline={criticalHistory} />
                <KpiCard label="High" value={threats.filter((t) => (t.severity || '').toLowerCase() === 'high').length} accent="amber" />
                <KpiCard label="Total" value={threats.length} accent="indigo" />
              </motion.div>

              {threatCountHistory.length > 1 && (
                <section className="sec-trend">
                  <article className="dash-panel">
                    <div className="dash-panel-head">
                      <div><span className="dash-eyebrow">Trend</span><h3>Threat count</h3></div>
                      <span className="dash-panel-context">last {threatCountHistory.length} polls</span>
                    </div>
                    <TimeSeriesChart values={threatCountHistory} color="red" height={100} emptyMessage="Waiting for data…" />
                  </article>
                  <article className="dash-panel">
                    <div className="dash-panel-head">
                      <div><span className="dash-eyebrow">Trend</span><h3>Critical severity</h3></div>
                      <span className="dash-panel-context">last {criticalHistory.length} polls</span>
                    </div>
                    <TimeSeriesChart values={criticalHistory} color="red" height={100} emptyMessage="No critical threats" />
                  </article>
                </section>
              )}
              <div className="synth-filter-row">
                <Select
                  label="Severity"
                  size="sm"
                  value={threatFilter}
                  onChange={(e) => setThreatFilter(e.target.value as typeof threatFilter)}
                  options={[
                    { value: 'all', label: 'All' },
                    { value: 'critical', label: 'Critical' },
                    { value: 'high', label: 'High' },
                    { value: 'medium', label: 'Medium' },
                    { value: 'low', label: 'Low' },
                  ]}
                />
                <Select
                  label="Status"
                  size="sm"
                  value={threatResolved}
                  onChange={(e) => setThreatResolved(e.target.value as typeof threatResolved)}
                  options={[
                    { value: 'open', label: 'Open' },
                    { value: 'resolved', label: 'Resolved' },
                    { value: 'all', label: 'All' },
                  ]}
                />
              </div>
              {threats.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>◴</span>}
                  headline="No threats match these filters"
                  subhead="Threats surface here when the ingest pipeline detects anomalies — failed logins, suspicious IPs, or exfil attempts."
                />
              ) : (
                <div className="threat-card-list">
                  {threats.map((t) => (
                    <ThreatCard key={t.id} threat={t} />
                  ))}
                </div>
              )}
            </section>
          ) : null}

          {tab === 'compliance' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Compliance</span>
              <h2 className="dash-section-title">Framework posture</h2>
              {frameworks.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>◇</span>}
                  headline="No compliance rules defined"
                  subhead="Seed rules from /api/v1/setup/initialize or wait for an ingest worker to populate evaluations."
                />
              ) : (
                <div className="compliance-bar-list">
                  {frameworks.map((fw) => (
                    <ComplianceBar
                      key={fw.framework}
                      framework={fw.framework}
                      total={fw.total_rules}
                      pass={fw.pass_count}
                      fail={fw.fail_count}
                      unknown={fw.unknown_count}
                      onClick={() => setExpandedFw(expandedFw === fw.framework ? null : fw.framework)}
                    />
                  ))}
                </div>
              )}
              {expandedFw ? (
                <div className="compliance-rules">
                  <h3 className="dash-section-subtitle">{expandedFw.toUpperCase()} rules</h3>
                  <table className="dash-table">
                    <thead>
                      <tr>
                        <th>Rule</th>
                        <th>Severity</th>
                        <th>Status</th>
                        <th>Description</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(frameworks.find((f) => f.framework === expandedFw)?.rules || []).map((r) => (
                        <tr key={r.rule_id}>
                          <td><code>{r.rule_id}</code></td>
                          <td>{r.severity}</td>
                          <td>
                                                      <StatusPill
                                                        status={
                                                          r.status === 'pass'
                                                            ? 'ok'
                                                            : r.status === 'fail'
                                                            ? 'crit'
                                                            : 'warn'
                                                        }
                                                        label={r.status}
                                                        size="sm"
                                                      />
                                                    </td>
                          <td>{r.description}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : null}
            </section>
          ) : null}

          {tab === 'siem' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">SIEM</span>
              <h2 className="dash-section-title">Security event stream</h2>
              <div className="synth-filter-row">
                <Select
                  label="Severity"
                  size="sm"
                  value={siemSev}
                  onChange={(e) => setSiemSev(e.target.value as typeof siemSev)}
                  options={[
                    { value: 'all', label: 'All' },
                    { value: 'critical', label: 'Critical' },
                    { value: 'high', label: 'High' },
                    { value: 'medium', label: 'Medium' },
                    { value: 'low', label: 'Low' },
                  ]}
                />
              </div>
              {siem.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>≡</span>}
                  headline="No SIEM events in this window"
                  subhead="Events appear as ingest sources forward them. Default window is 24 hours."
                />
              ) : (
                <table className="dash-table">
                  <thead>
                    <tr>
                      <th>Severity</th>
                      <th>Type</th>
                      <th>Source</th>
                      <th>Message</th>
                      <th>Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {siem.map((e) => (
                      <tr key={e.id}>
                        <td>
                                                  <StatusPill
                                                    status={
                                                      e.severity === 'critical' || e.severity === 'high'
                                                        ? 'crit'
                                                        : e.severity === 'medium'
                                                        ? 'warn'
                                                        : 'ok'
                                                    }
                                                    label={e.severity}
                                                    size="sm"
                                                  />
                                                </td>
                        <td><code>{e.event_type}</code></td>
                        <td><code>{e.source || '—'}</code></td>
                        <td>{e.message}</td>
                        <td><small>{new Date(e.occurred_at).toLocaleString()}</small></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          ) : null}

          {tab === 'audit' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Audit</span>
              <h2 className="dash-section-title">Audit log</h2>
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>▤</span>}
                headline="See Audit Log under Settings"
                subhead="The Security tab here surfaces a curated view of security-relevant audit events. The full audit log — covering every privileged action in the platform — is accessible from the Settings page."
              />
            </section>
          ) : null}
        </motion.div>
  );
}