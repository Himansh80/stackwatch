import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import LogSearchTab from '../components/logs/LogSearchTab';
import LogMonitorsTab, { LogMonitor } from '../components/logs/LogMonitorsTab';
import LogArchivesTab, { LogArchive, LogRehydration } from '../components/logs/LogArchivesTab';
import LogRetentionTab, { LogRetention } from '../components/logs/LogRetentionTab';
import LogPatternsTab, { LogPattern } from '../components/logs/LogPatternsTab';
import KpiCard from '../components/shared/KpiCard';
import StatusPill from '../components/shared/StatusPill';
import type { LogEntryData } from '../components/shared/LogEntry';
import { motion, pageEnter } from '../lib/motion';

type Tab = 'search' | 'retention' | 'archives' | 'monitors' | 'patterns';
const TAB_LABELS: Record<Tab, string> = {
  search: 'Search',
  retention: 'Retention',
  archives: 'Archives',
  monitors: 'Monitors',
  patterns: 'Patterns',
};

/**
 * LogsFullPage — Log Management Full (D3) at /logs.
 *
 * Tab dispatcher that wires data loads per-tab and renders one of the
 * five LogSearchTab / LogMonitorsTab / LogArchivesTab / LogRetentionTab /
 * LogPatternsTab view components. Each view file is its own module
 * (under 400 LOC) so this page stays small.
 *
 * All data loads happen lazily per-tab — we don't fetch retention data
 * while the user is on the Search tab, etc. Keeps the initial render
 * cheap and avoids unnecessary 401s when the user lands on the page.
 */
export default function LogsFullPage() {
  const [tab, setTab] = useState<Tab>('search');
  const [error, setError] = useState('');
  const [logs, setLogs] = useState<LogEntryData[]>([]);
  const [monitors, setMonitors] = useState<LogMonitor[]>([]);
  const [archives, setArchives] = useState<LogArchive[]>([]);
  const [rehydrations, setRehydrations] = useState<LogRehydration[]>([]);
  const [retention, setRetention] = useState<LogRetention[]>([]);
  const [patterns, setPatterns] = useState<LogPattern[]>([]);
  const [searchBusy, setSearchBusy] = useState(false);

  const requireAuth = () => {
    if (!getToken()) {
      setError('Sign in to view logs.');
      return false;
    }
    return true;
  };

  const loadMonitors = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ monitors?: LogMonitor[] }>('GET', '/api/v1/logs/monitors');
      setMonitors(r.monitors || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadArchives = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const [ar, rh] = await Promise.all([
        api<{ archives?: LogArchive[] }>('GET', '/api/v1/logs/archives'),
        api<{ rehydrations?: LogRehydration[] }>('GET', '/api/v1/logs/rehydrations'),
      ]);
      setArchives(ar.archives || []);
      setRehydrations(rh.rehydrations || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadRetention = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ policies?: LogRetention[] }>('GET', '/api/v1/logs/retention');
      setRetention(r.policies || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadPatterns = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<{ patterns?: LogPattern[] }>('GET', '/api/v1/logs/patterns');
      setPatterns(r.patterns || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  useEffect(() => {
    setError('');
    if (tab === 'monitors') void loadMonitors();
    else if (tab === 'archives') void loadArchives();
    else if (tab === 'retention') void loadRetention();
    else if (tab === 'patterns') void loadPatterns();
  }, [tab, loadArchives, loadMonitors, loadRetention, loadPatterns]);

  const handleSearch = useCallback(async (q: string) => {
    setError('');
    if (!requireAuth()) return;
    setSearchBusy(true);
    try {
      // /api/v1/logs/query is part of the Tier 6 v1 log search surface
      // (see spec.md §"Logs Full endpoints"). Phase 2 keeps it as the
      // backing source for the Search tab — when it returns no rows,
      // the search tab surfaces an empty state pointing the user at
      // the Patterns tab for what's flowing.
      const params = new URLSearchParams();
      if (q) params.set('q', q);
      const r = await api<{ entries?: LogEntryData[] }>('GET', `/api/v1/logs/query?${params.toString()}`)
        .catch(() => ({ entries: [] }));
      setLogs(r.entries || []);
    } finally {
      setSearchBusy(false);
    }
  }, []);

  const _summary = useMemo(() => {
    if (tab === 'search') return `${logs.length} entries`;
    if (tab === 'monitors') return `${monitors.length} monitor${monitors.length === 1 ? '' : 's'}`;
    if (tab === 'archives') return `${archives.length} archive${archives.length === 1 ? '' : 's'}`;
    if (tab === 'retention') return `${retention.length} polic${retention.length === 1 ? 'y' : 'ies'}`;
    return `${patterns.length} pattern${patterns.length === 1 ? '' : 's'}`;
  }, [tab, logs, monitors, archives, retention, patterns]);

  // Log-level KPIs computed client-side from the loaded logs array.
  const logLevelKpis = useMemo(() => {
    const counts = { error: 0, warn: 0, info: 0, debug: 0 };
    for (const entry of logs) {
      const lvl = String(entry.level ?? '').toLowerCase();
      if (lvl === 'error' || lvl === 'fatal' || lvl === 'critical') counts.error += 1;
      else if (lvl === 'warn' || lvl === 'warning') counts.warn += 1;
      else if (lvl === 'info' || lvl === 'notice') counts.info += 1;
      else if (lvl === 'debug' || lvl === 'trace') counts.debug += 1;
    }
    return counts;
  }, [logs]);

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          <section className="logs-kpi-strip">
            <KpiCard label="Errors" value={logLevelKpis.error} accent="red" />
            <KpiCard label="Warnings" value={logLevelKpis.warn} accent="amber" />
            <KpiCard label="Info" value={logLevelKpis.info} accent="cyan" />
            <KpiCard label="Debug" value={logLevelKpis.debug} accent="indigo" />
          </section>

          <div className="logs-tabs" role="tablist" style={{ marginTop: 16 }}>
            {(Object.keys(TAB_LABELS) as Tab[]).map((t) => (
              <button
                key={t}
                role="tab"
                type="button"
                aria-selected={tab === t}
                className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
                onClick={() => setTab(t)}
              >
                <StatusPill
                  status={
                    t === 'search' && logs.length > 0 ? 'ok' :
                    t === 'monitors' && monitors.length > 0 ? 'warn' :
                    t === 'archives' && archives.length > 0 ? 'ok' :
                    t === 'retention' && retention.length > 0 ? 'ok' :
                    t === 'patterns' && patterns.length > 0 ? 'ok' :
                    'unknown'
                  }
                  label={TAB_LABELS[t]}
                  size="sm"
                />
              </button>
            ))}
          </div>

          {tab === 'search' ? (
            <LogSearchTab entries={logs} busy={searchBusy} error={error} onSearch={handleSearch} />
          ) : null}
          {tab === 'monitors' ? (
            <LogMonitorsTab monitors={monitors} onChanged={loadMonitors} setError={setError} />
          ) : null}
          {tab === 'archives' ? (
            <LogArchivesTab archives={archives} rehydrations={rehydrations} onChanged={loadArchives} setError={setError} />
          ) : null}
          {tab === 'retention' ? (
            <LogRetentionTab policies={retention} onChanged={loadRetention} />
          ) : null}
          {tab === 'patterns' ? (
            <LogPatternsTab patterns={patterns} />
          ) : null}
        </motion.div>
  );
}
