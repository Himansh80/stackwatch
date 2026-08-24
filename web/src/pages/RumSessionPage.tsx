import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import {
  VitalsTab,
  InteractionsTab,
  ResourcesTab,
  ErrorsTab,
  type RumVital,
  type RumInteraction,
} from '../components/shared/RumSessionTabs';
import type { WaterfallResource } from '../components/shared/ResourceWaterfall';
import { motion, pageEnter } from '../lib/motion';

/**
 * RumSessionPage — session detail at /rum/session?id=<session_id>.
 *
 * Top: session header (web vitals / resources / interactions / long
 *      tasks counts).
 * Middle: tabs (Vitals / Resources / Interactions / Errors).
 *
 * Tab bodies live in `RumSessionTabs.tsx` so this page stays small.
 * Wrapped with `pageEnter` for the page entrance.
 */

type Tab = 'vitals' | 'resources' | 'interactions' | 'errors';

interface SessionHeader {
  session_id: string;
  first_url: string;
  started_at: string;
  last_seen: string;
  web_vitals: number;
  resources: number;
  interactions: number;
  long_tasks: number;
  errors: number;
}

interface WaterfallPayload {
  session_id: string;
  resources: WaterfallResource[];
  vitals: RumVital[];
  interactions: RumInteraction[];
}

export default function RumSessionPage() {
  const logout = useLogout();
  const [params] = useSearchParams();
  const sessionId = params.get('id') || '';
  const [header, setHeader] = useState<SessionHeader | null>(null);
  const [waterfall, setWaterfall] = useState<WaterfallPayload | null>(null);
  const [tab, setTab] = useState<Tab>('resources');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view RUM sessions.');
      setLoading(false);
      return;
    }
    if (!sessionId) {
      setError('Missing session id.');
      setLoading(false);
      return;
    }
    setError('');
    setLoading(true);
    try {
      const [hdr, wf] = await Promise.all([
        api<SessionHeader>('GET', `/api/v1/rum/sessions/${encodeURIComponent(sessionId)}`),
        api<WaterfallPayload>('GET', `/api/v1/rum/sessions/${encodeURIComponent(sessionId)}/waterfall`),
      ]);
      setHeader(hdr);
      setWaterfall(wf);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message || 'Unable to load session.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  useEffect(() => { void load(); }, [load]);

  const tabEyebrow: Record<Tab, string> = {
    resources: 'Network',
    vitals: 'Performance',
    interactions: 'UX',
    errors: 'Reliability',
  };
  const tabTitle: Record<Tab, string> = {
    resources: 'Resource waterfall',
    vitals: 'Web vitals',
    interactions: 'User interactions',
    errors: 'Errors in this session',
  };

  return (
    <div className="dash-app">
      <AppSidebar
        active="rum"
        onLogout={logout}
        show={['dashboard', 'billing', 'profile', 'settings', 'proxmox', 'truenas', 'apm', 'logs', 'rum']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">RUM session</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">
                {header?.first_url
                  ? header.first_url.slice(0, 64)
                  : (loading ? 'Loading…' : sessionId.slice(0, 12))}
              </strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">
                  {sessionId.slice(0, 16)}…
                </span>
              </span>
            </div>
          </div>
        </header>

        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? (
            <div className="dash-error" role="alert">{error}</div>
          ) : null}

          {header ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Overview</span>
              <h2 className="dash-section-title">Session timeline</h2>
              <div className="dash-kpi-strip">
                <Kpi label="Web vitals" value={String(header.web_vitals)} />
                <Kpi label="Resources" value={String(header.resources)} />
                <Kpi label="Interactions" value={String(header.interactions)} />
                <Kpi label="Long tasks" value={String(header.long_tasks)} />
              </div>
              <p className="dash-section-lede">
                Started {header.started_at ? new Date(header.started_at).toLocaleString() : '—'}
                {header.last_seen ? ` · last seen ${new Date(header.last_seen).toLocaleString()}` : ''}
              </p>
            </section>
          ) : null}

          <div className="logs-tabs" role="tablist">
            {(['resources', 'vitals', 'interactions', 'errors'] as Tab[]).map((t) => (
              <button
                key={t}
                role="tab"
                type="button"
                aria-selected={tab === t}
                className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
                onClick={() => setTab(t)}
              >
                {t}
              </button>
            ))}
          </div>

          <section className="dash-section">
            <span className="dash-eyebrow">{tabEyebrow[tab]}</span>
            <h2 className="dash-section-title">{tabTitle[tab]}</h2>
            {loading ? (
              <p className="dash-section-lede">Loading session data…</p>
            ) : tab === 'resources' ? (
              <ResourcesTab resources={waterfall?.resources || []} />
            ) : tab === 'vitals' ? (
              <VitalsTab vitals={waterfall?.vitals || []} />
            ) : tab === 'interactions' ? (
              <InteractionsTab interactions={waterfall?.interactions || []} />
            ) : (
              <ErrorsTab errorCount={header?.errors ?? 0} />
            )}
          </section>
        </motion.div>
      </main>
    </div>
  );
}

// Kpi is a tiny local tile used for the session summary row.
function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="dash-metric">
      <small>{label}</small>
      <strong>{value}</strong>
    </div>
  );
}
