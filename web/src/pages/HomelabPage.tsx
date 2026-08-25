import { useCallback, useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import GlobalSearch from '../components/homelab/GlobalSearch';
import HomelabGrid from '../components/homelab/HomelabGrid';
import HomelabKpiStrip from '../components/homelab/HomelabKpiStrip';
import { motion, buttonSpring, pageEnter, useReducedMotion } from '../lib/motion';

/**
 * HomelabPage — Tier 10 Phase 1 (H1 — Widget Framework + Page shell).
 *
 * The personal/glanceable dashboard at /homelab. Phase 1 ships the
 * page shell, the tab strip, the KPI strip, and the HomelabGrid
 * container — every Phase 2-9 widget drops into a tab.
 *
 * Layout (top → bottom):
 *   - AppSidebar (left rail, page nav: 'homelab' is NEW — added to
 *     AppSidebar in Phase 1 task 1.9 alongside 'profile'/'settings')
 *   - Header (eyebrow + title + subtitle + Refresh + Reset buttons)
 *   - KPI strip (4 cards — see HomelabKpiStrip)
 *   - Tab strip (8 tabs — only 'overview' is interactive in Phase 1;
 *     the rest render placeholder text until their phase lands)
 *   - Active tab content (overview renders HomelabGrid; others
 *     render "Coming in Phase N" stubs)
 *
 * 8 tabs (Phase 1 ships 'overview' + placeholder text on the rest):
 *   - Overview   (Phase 1 — HomelabGrid)
 *   - Services   (Phase 2)
 *   - Notes      (Phase 3)
 *   - Calendar   (Phase 4)
 *   - Downloads  (Phase 5)
 *   - Media      (Phase 6)
 *   - RSS        (Phase 8 — H9)
 *   - Scheduler  (Phase 9 — H10)
 *
 * Phase 7 (Search — H7) is a global search bar in the topbar (not a
 * tab) — matches the proposal §US-7 ("global search bar at the top").
 * Wired via the GlobalSearch component (imported below).
 *
 * Per-user prefs (Phase 1):
 *   - GET /prefs runs once on mount so the HomelabKpiStrip knows
 *     the user's theme / refresh / default_landing. Phase 1 only
 *     uses default_landing; future phases read refresh_seconds to
 *     drive polling.
 *   - Theme is rendered as a data attribute on <main> so global CSS
 *     can theme the dashboard later.
 *
 * Motion: pageEnter on the page wrapper; buttonSpring on the header
 * Refresh + Reset buttons.
 */

type Tab = 'overview' | 'services' | 'notes' | 'calendar' | 'downloads' | 'media' | 'rss' | 'scheduler';

const TAB_LABELS: Record<Tab, string> = {
  overview: 'Overview',
  services: 'Services',
  notes: 'Notes',
  calendar: 'Calendar',
  downloads: 'Downloads',
  media: 'Media',
  rss: 'RSS',
  scheduler: 'Scheduler',
};

// Per-tab "coming in Phase N" stub copy. Honest about scope — no
// fake widgets pretending to work. The user immediately understands
// this is a phased rollout.
const TAB_PHASE_HINTS: Record<Tab, string> = {
  overview: '',
  services: 'Services widget (Phase 2) — pin services + see HTTP/TCP/ICMP health probes',
  notes: 'Notes widget (Phase 3) — Markdown notes with checklist support',
  calendar: 'Calendar widget (Phase 4) — overlay personal iCal + agent maintenance windows',
  downloads: 'Download stats (Phase 5) — Sonarr/Radarr/qBittorrent/SABnzbd queue + speed',
  media: 'Media server (Phase 6) — Plex/Jellyfin/Emby now-playing + recent additions',
  rss: 'RSS reader (Phase 8) — polled feeds for selfhosted release notes',
  scheduler: 'Task scheduler (Phase 9) — cron-style personal jobs (HTTP probes + webhooks)',
};

interface HomelabPrefs {
  theme: string;
  refresh_seconds: number;
  pinned_widgets: string[];
  default_landing: string;
  updated_at: string;
}

export default function HomelabPage() {
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('overview');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [prefs, setPrefs] = useState<HomelabPrefs | null>(null);

  // requireAuth — same pattern as IntelligencePage. Bails to a
  // friendly error rather than letting the request 401 in the wild.
  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view your homelab dashboard.');
      return false;
    }
    return true;
  };

  // Load prefs once on mount. Future phases read refresh_seconds to
  // drive polling — Phase 1 just stores it. Theme is set as a
  // data-attribute on <main> so CSS can react (Phase 1 ships the
  // wiring; the CSS is a follow-up polish task).
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (!requireAuth()) return;
      setBusy(true);
      try {
        const p = await api<HomelabPrefs>('GET', '/api/v1/homelab/prefs');
        if (!cancelled) setPrefs(p);
      } catch (cause) {
        if (!cancelled) {
          setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
        }
      } finally {
        if (!cancelled) setBusy(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  // Refresh — re-runs the layout/prefs load. Phase 1 doesn't poll,
  // so this is the only "refresh" mechanism. Future phases can swap
  // it for a setInterval that respects prefs.refresh_seconds.
  const onRefresh = useCallback(() => {
    if (!requireAuth()) return;
    // Force a reload by triggering a tiny state bounce. The grid
    // and prefs each re-fetch on their next mount-keyed effect;
    // simplest implementation is to navigate to the same route via
    // a key bump on the wrapper.
    setBusy(true);
    setTimeout(() => {
      setBusy(false);
      window.location.reload();
    }, 250);
  }, []);

  // Reset — DELETE /homelab/prefs/layout (idempotent), then trigger
  // a refresh so HomelabGrid reloads the default layout.
  const onResetLayout = useCallback(async () => {
    if (!requireAuth()) return;
    setBusy(true);
    try {
      await api('DELETE', '/api/v1/homelab/prefs/layout');
      // Bounce the page so HomelabGrid picks up the cleared layout.
      setTimeout(() => window.location.reload(), 250);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setTimeout(() => setBusy(false), 250);
    }
  }, []);

  // KPI card → tab navigation. Phase 1: services / notes / todos /
  // rss tabs are placeholders, but the click still switches the
  // active tab so the user can see the placeholder copy.
  const onKpiNavigate = useCallback((target: string) => {
    if (target === 'services' || target === 'notes' || target === 'todos' || target === 'rss') {
      setTab(target as Tab);
    }
  }, []);

  // GlobalSearch → tab navigation. Phase 7 only knows about the
  // kinds backed by the existing 6 tables. Unknown kinds fall
  // through to no-op (the backend shouldn't emit unknown kinds,
  // but a defensive early-return beats a switch-throw). The
  // services kind carries its own URL (e.g. an external Grafana
  // link) so we open that in a new tab instead of switching.
  const onSearchNavigate = useCallback((kind: string, _id: string, url: string) => {
    if (kind === 'services') {
      window.open(url, '_blank', 'noopener,noreferrer');
      return;
    }
    const tabMap: Record<string, Tab> = {
      notes: 'notes',
      todos: 'notes', // todos live inside the Notes tab (placeholder copy)
      calendars: 'calendar',
      downloads: 'downloads',
      media: 'media',
    };
    const target = tabMap[kind];
    if (target) setTab(target);
  }, []);

  const activePhaseHint = TAB_PHASE_HINTS[tab];

  return (
    <div className="dash-app" data-theme={prefs?.theme || 'auto'}>
      <AppSidebar
        active="homelab"
        onLogout={logout}
        show={[
          'dashboard',
          'homelab',
          'billing',
          'profile',
          'settings',
          'incidents',
          'notebooks',
          'intelligence',
          'enterprise',
        ]}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Homelab</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">
                Personal dashboard — pinned services, notes, calendar, media
              </strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">
                  {prefs ? `${prefs.refresh_seconds}s refresh · ${prefs.default_landing} landing` : 'Loading…'}
                </span>
              </span>
            </div>
            {/* Phase 7 — global search bar. Lives in the topbar
                under the greeting (not a tab) so it's reachable
                from any tab context. 300ms debounce on the backend
                suggestions keeps the keystroke loop responsive. */}
            <div className="dash-topbar-search">
              <GlobalSearch onNavigate={onSearchNavigate} />
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={onRefresh}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy}
              title="Reload homelab data"
            >
              ↻ Refresh
            </motion.button>
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => void onResetLayout()}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy}
              title="Reset widget layout to defaults (preferences preserved)"
            >
              ⤓ Reset layout
            </motion.button>
          </div>
        </header>

        <motion.div
          className="dash-page"
          initial="hidden"
          animate="show"
          variants={pageEnter}
        >
          {error ? (
            <div className="dash-error" role="alert">
              {error}
            </div>
          ) : null}

          {/* KPI strip — 4 cards. Phase 1 ships with placeholder
              counts; future phases replace with real data per card. */}
          <HomelabKpiStrip
            pinnedServicesCount={0}
            notesCount={0}
            todosDueSoonCount={0}
            rssUnreadCount={0}
            onNavigate={onKpiNavigate}
          />

          {/* Tab strip — horizontally scrollable so 8 tabs don't
              squeeze on tablet viewports. The tabs reuse the same
              `logs-tabs` class set other Tier pages use (their CSS
              is in styles-tier2.css — we add `homelab-tabs` for
              theme overrides if needed in a later polish pass). */}
          <div className="homelab-tabs" role="tablist" aria-label="Homelab sections" style={{ marginTop: 16 }}>
            {(Object.keys(TAB_LABELS) as Tab[]).map((t) => (
              <button
                key={t}
                role="tab"
                type="button"
                aria-selected={tab === t}
                className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
                onClick={() => setTab(t)}
              >
                {TAB_LABELS[t]}
              </button>
            ))}
          </div>

          {/* Active tab content. Overview renders HomelabGrid;
              every other tab renders a Phase N placeholder so the
              user knows when each surface ships. */}
          {tab === 'overview' ? (
            <div style={{ marginTop: 16 }}>
              <HomelabGrid onError={setError} />
            </div>
          ) : (
            <div className="homelab-tab-stub" role="tabpanel" style={{ marginTop: 16 }}>
              <h3 className="homelab-tab-stub-title">{TAB_LABELS[tab]}</h3>
              <p className="homelab-tab-stub-hint">{activePhaseHint}</p>
            </div>
          )}
        </motion.div>
      </main>
    </div>
  );
}