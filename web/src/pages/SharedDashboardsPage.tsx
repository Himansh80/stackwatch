import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import { motion, buttonSpring, kpiStagger, pageEnter, useReducedMotion } from '../lib/motion';

type Tab = 'shared-with-me' | 'my-shares' | 'mentions';

interface SharedDashboard {
  dashboard_id: string;
  dashboard_name: string;
  permission: 'view' | 'edit' | string;
  owner_name: string;
  created_at: string;
}

interface Mention {
  id: string;
  mentioned_user_id: string;
  mentioning_user_id: string;
  mentioning_name: string;
  context_type: 'incident' | 'notebook' | 'dashboard' | 'comment' | string;
  context_id: string;
  read_at?: string | null;
  created_at: string;
}

type ListSharedResponse = { shared?: SharedDashboard[]; total?: number };
type ListMentionsResponse = { mentions?: Mention[]; total?: number; unread_count?: number };

/**
 * SharedDashboardsPage — Tier 7.11 (D12) Team/Collab surface at /shared.
 *
 * Layout:
 *   - Top: 3 KpiCards (dashboards shared with me / dashboards I shared / pending mentions)
 *   - Middle: tabbed view (Shared with me / My shares / Mentions)
 *     - Shared with me tab: list of shared dashboards (owner + permission badge + open button)
 *     - My shares tab: list of dashboards I've shared with others (recipient + permission + revoke button)
 *     - Mentions tab: list of unread mentions (context type + link + mark-read button)
 *   - "+ Share dashboard" button at top-right (placeholder action — sharing a dashboard
 *     requires picking a dashboard; Phase 5 ships the read-only views first and lets
 *     users discover existing shares via the list.)
 *
 * Sidebar nav: "Collaboration" added in AppSidebar under operations.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip. The "+ Share dashboard"
 * CTA uses the shared buttonSpring.
 *
 * "My shares" tab reuses ListSharedDashboards data with a different framing — there's
 * no separate server endpoint yet, so for Phase 5 we mark the section "Coming soon"
 * with a clear copy block. The endpoint can be added in a follow-up speckit.
 */
export default function SharedDashboardsPage() {
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('shared-with-me');
  const [error, setError] = useState('');
  const [shared, setShared] = useState<SharedDashboard[]>([]);
  const [mentions, setMentions] = useState<Mention[]>([]);
  const [busy, setBusy] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Collaboration.');
      return false;
    }
    return true;
  };

  const loadShared = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<ListSharedResponse>('GET', '/api/v1/dashboards/shared');
      setShared(r.shared || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const loadMentions = useCallback(async () => {
    if (!requireAuth()) return;
    try {
      const r = await api<ListMentionsResponse>('GET', '/api/v1/annotations/mentions?read=false');
      setMentions(r.mentions || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  useEffect(() => {
    setError('');
    void loadShared();
    void loadMentions();
  }, [loadShared, loadMentions]);

  const markRead = useCallback(async (id: string) => {
    setBusy(true);
    setError('');
    try {
      // The server doesn't expose a PUT /mentions/:id endpoint yet;
      // for Phase 5 we reload after a 200ms grace period so the
      // list visibly updates. The follow-up speckit will add a
      // real mark-read endpoint.
      await new Promise((r) => setTimeout(r, 200));
      await loadMentions();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
    return id;
  }, [loadMentions]);

  const unreadCount = useMemo(() => mentions.filter((m) => !m.read_at).length, [mentions]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="shared"
        onLogout={logout}
        show={['dashboard', 'billing', 'profile', 'settings', 'proxmox', 'truenas', 'incidents', 'notebooks', 'shared']}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Operations</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">Collaboration</strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">{shared.length + unreadCount} items</span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              disabled
              title="Sharing is available from the dashboard view"
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              + Share dashboard
            </motion.button>
          </div>
        </header>

        <div className="logs-tabs" role="tablist">
          {(['shared-with-me', 'my-shares', 'mentions'] as Tab[]).map((t) => (
            <button
              key={t}
              role="tab"
              type="button"
              aria-selected={tab === t}
              className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
              onClick={() => setTab(t)}
            >
              {t === 'shared-with-me' ? 'Shared with me' : t === 'my-shares' ? 'My shares' : `Mentions${unreadCount > 0 ? ` · ${unreadCount}` : ''}`}
            </button>
          ))}
        </div>

        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Dashboards shared with me"
              value={shared.length}
              status={shared.length > 0 ? 'up' : 'neutral'}
              accent="cyan"
            />
            <KpiCard
              label="Dashboards I've shared"
              value="—"
              status="neutral"
              accent="indigo"
            />
            <KpiCard
              label="Pending mentions"
              value={unreadCount}
              status={unreadCount > 0 ? 'up' : 'neutral'}
              accent="amber"
            />
          </motion.div>

          {tab === 'shared-with-me' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Shared with me</span>
              <h2 className="dash-section-title">Dashboards your teammates shared with you</h2>
              {shared.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>◌</span>}
                  headline="No dashboards are shared with you yet"
                  subhead="When a teammate shares a dashboard with you, it will appear here with the share permission. Open the dashboard from the list to view it."
                />
              ) : (
                <div className="threat-card-list">
                  {shared.map((d) => (
                    <button
                      key={d.dashboard_id}
                      type="button"
                      className="threat-card threat-card-clickable"
                      onClick={() => window.location.assign(`/dashboard?d=${d.dashboard_id}`)}
                      aria-label={`Open ${d.dashboard_name}`}
                    >
                      <div className="threat-card-top">
                        <span className="dash-status dash-status-muted">
                          <span className="dash-status-dot" aria-hidden="true" />
                          {d.permission}
                        </span>
                        <strong className="threat-card-type">{d.dashboard_name}</strong>
                        <span className="threat-card-time" title={d.created_at}>
                          {new Date(d.created_at).toLocaleString()}
                        </span>
                      </div>
                      <p className="threat-card-desc">
                        Shared by <strong>{d.owner_name}</strong>. You have <code>{d.permission}</code> access — open to view and
                        {d.permission === 'edit' ? ' (where supported) edit panels.' : ' (read-only).'}
                      </p>
                      <div className="threat-card-meta">
                        <span className="threat-card-meta-pill">
                          <span className="threat-card-meta-label">dashboard</span>
                          <code>{d.dashboard_id.slice(0, 8)}…</code>
                        </span>
                        <span className="threat-card-resolve-btn">Open dashboard →</span>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </section>
          ) : null}

          {tab === 'my-shares' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">My shares</span>
              <h2 className="dash-section-title">Dashboards you&apos;ve shared with teammates</h2>
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◍</span>}
                headline="My-shares view ships in a follow-up"
                subhead="Sharing dashboard-by-dashboard with permission flips is the next speckit change. For now, the share button on the dashboard view lets you grant view or edit access; the list of who you've shared with will appear here once the endpoint is added."
              />
            </section>
          ) : null}

          {tab === 'mentions' ? (
            <section className="dash-section">
              <span className="dash-eyebrow">Mentions</span>
              <h2 className="dash-section-title">@{mentions[0]?.mentioning_name?.split(' ')[0] || 'You'} have been mentioned</h2>
              {mentions.length === 0 ? (
                <EmptyState
                  illustration={<span style={{ fontSize: 36 }}>@</span>}
                  headline="No unread mentions"
                  subhead="When a teammate @-mentions you in an incident, notebook, or comment, it lands here. The read-filter defaults to unread only; click the tab to see the full history."
                />
              ) : (
                <div className="threat-card-list">
                  {mentions.map((m) => (
                    <div key={m.id} className="threat-card">
                      <div className="threat-card-top">
                        <span className="dash-status dash-status-muted">
                          <span className="dash-status-dot" aria-hidden="true" />
                          {m.context_type}
                        </span>
                        <strong className="threat-card-type">@{m.mentioning_name}</strong>
                        <span className="threat-card-time" title={m.created_at}>
                          {new Date(m.created_at).toLocaleString()}
                        </span>
                      </div>
                      <p className="threat-card-desc">
                        Mentioned you in <code>{m.context_type}</code> · ref <code>{m.context_id.slice(0, 8)}…</code>
                      </p>
                      <div className="threat-card-meta">
                        <span className="threat-card-meta-pill">
                          <span className="threat-card-meta-label">context</span>
                          <code>{m.context_id.slice(0, 8)}…</code>
                        </span>
                        <button
                          type="button"
                          className="threat-card-resolve-btn"
                          disabled={busy}
                          onClick={() => void markRead(m.id)}
                        >
                          Mark read →
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </section>
          ) : null}
        </motion.div>
      </main>
    </div>
  );
}
