import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, kpiEnter, kpiStagger, pageEnter } from '../../../lib/motion';
import Button from '../../shared/Button';
import AddMediaModal from './AddMediaModal';
import MediaServerCard from './MediaServerCard';
import NowPlayingList from './NowPlayingList';
import RecentAdditionsCarousel from './RecentAdditionsCarousel';
import { kindLabel } from './mediaFormatters';
import type {
  MediaServerFormState,
  MediaServerKind,
  MediaServerRow,
  MediaStateResponse,
} from './types';

/**
 * MediaWidget — Tier 10 Phase 6 (H6 — Media Server).
 *
 * The Media tab content on the HomelabPage. Loads the caller's
 * pinned media servers (Plex/Jellyfin/Emby) via:
 *   GET /api/v1/homelab/media/servers — registry list
 *   GET /api/v1/homelab/media/state   — now-playing + recent additions
 *
 * And writes via:
 *   POST   /api/v1/homelab/media/servers     — pin a new server
 *   DELETE /api/v1/homelab/media/servers/:id — unpin
 *
 * The MediaWorker (backend, internal/homelab/media.go) ticks
 * every 60s and UPSERTs now_playing + recent_additions rows.
 *
 * Layout (Datadog-style):
 *   - Header: "Media" title + "+ Add server" + "Refresh" buttons
 *   - Tab strip: All / Plex / Jellyfin / Emby (filters all sections)
 *   - 3 KPI cards: servers / active sessions / recent additions
 *   - Server chips (via MediaServerCard)
 *   - Now Playing list (via NowPlayingList)
 *   - Recent Additions carousel (via RecentAdditionsCarousel)
 *
 * Motion: pageEnter on the wrapper, kpiEnter + kpiStagger on
 * the KPI strip (reuses existing tokens — no new variants).
 *
 * Formatters (kindLabel, progressPercent, fmtProgressMs) live
 * in mediaFormatters.ts. Per-server cards live in
 * MediaServerCard.tsx. Now-playing list lives in
 * NowPlayingList.tsx. Recent-additions carousel lives in
 * RecentAdditionsCarousel.tsx. Add-server modal lives in
 * AddMediaModal.tsx. This split keeps the orchestrator file
 * under the 400-LOC cap.
 */

const POLL_INTERVAL_MS = 60_000;

const emptyForm: MediaServerFormState = {
  name: '',
  kind: 'plex',
  base_url: '',
  api_key: '',
};

type TabFilter = 'all' | MediaServerKind;

export default function MediaWidget({
  config: _config,
}: {
  config?: { poll_seconds?: number };
}) {
  void _config;
  const [state, setState] = useState<MediaStateResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<TabFilter>('all');
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<MediaServerFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your media servers.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api<MediaStateResponse>(
        'GET',
        '/api/v1/homelab/media/state',
      );
      setState(res);
      setError('');
    } catch (cause) {
      const msg =
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // 60s polling — matches the worker tick cadence. Refresh
  // button forces an immediate re-fetch.
  useEffect(() => {
    const t = window.setInterval(() => {
      void load();
    }, POLL_INTERVAL_MS);
    return () => window.clearInterval(t);
  }, [load]);

  const openForm = useCallback(() => {
    setForm(emptyForm);
    setFormError('');
    setModalOpen(true);
  }, []);

  const closeForm = useCallback(() => setModalOpen(false), []);

  const submitForm = useCallback(async () => {
    setBusy(true);
    setFormError('');
    try {
      const name = form.name.trim();
      const baseURL = form.base_url.trim();
      const apiKey = form.api_key.trim();
      if (!name) {
        setFormError('Name is required.');
        return;
      }
      if (!baseURL) {
        setFormError('Base URL is required.');
        return;
      }
      if (!apiKey) {
        setFormError(`${kindLabel(form.kind)} requires an api_key / token.`);
        return;
      }
      await api('POST', '/api/v1/homelab/media/servers', {
        name,
        kind: form.kind,
        base_url: baseURL,
        api_key: apiKey,
      });
      setModalOpen(false);
      // The POST handler fires an immediate fire-and-forget
      // poll; give the worker ~1.5s for the UPSERTs to land
      // before reloading so the freshly-pinned server shows
      // real state.
      window.setTimeout(() => {
        void load();
      }, 1500);
    } catch (cause) {
      const msg =
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, load]);

  const deleteServer = useCallback(
    async (s: MediaServerRow) => {
      if (!window.confirm(`Remove server "${s.name}" and its state history?`))
        return;
      try {
        await api('DELETE', `/api/v1/homelab/media/servers/${s.id}`);
        await load();
      } catch (cause) {
        const msg =
          cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
        setError(msg);
      }
    },
    [load],
  );

  // Filtered views per the active tab. When 'all', show
  // everything; otherwise scope to the matching kind.
  const filteredServers = useMemo<MediaServerRow[]>(() => {
    if (!state) return [];
    if (tab === 'all') return state.servers;
    return state.servers.filter((s) => s.kind === tab);
  }, [state, tab]);

  const filteredNowPlaying = useMemo(() => {
    if (!state) return [];
    if (tab === 'all') return state.now_playing;
    return state.now_playing.filter((n) => n.server_kind === tab);
  }, [state, tab]);

  const filteredRecent = useMemo(() => {
    if (!state) return [];
    if (tab === 'all') return state.recent_additions;
    return state.recent_additions.filter((r) => r.server_kind === tab);
  }, [state, tab]);

  const kpis = useMemo(() => {
    if (!state) {
      return { servers: 0, sessions: 0, recent: 0 };
    }
    if (tab === 'all') {
      return {
        servers: state.servers.length,
        sessions: state.now_playing.length,
        recent: state.recent_additions.length,
      };
    }
    return {
      servers: state.servers.filter((s) => s.kind === tab).length,
      sessions: state.now_playing.filter((n) => n.server_kind === tab).length,
      recent: state.recent_additions.filter((r) => r.server_kind === tab).length,
    };
  }, [state, tab]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your media servers…
      </div>
    );
  }

  return (
    <motion.div
      className="homelab-services-widget"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      {error ? (
        <div className="dash-error" role="alert">
          {error}
        </div>
      ) : null}

      <div
        style={{
          display: 'flex',
          gap: 8,
          alignItems: 'center',
          flexWrap: 'wrap',
        }}
      >
        <h3 style={{ margin: 0, fontSize: 18 }}>Media</h3>
        <Button variant="primary" size="sm" onClick={openForm}>
          + Add server
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void load()}
          disabled={loading}
        >
          ↻ Refresh
        </Button>
      </div>

      {/* Tab strip: All / Plex / Jellyfin / Emby. Datadog-style. */}
      <div
        className="homelab-tab-strip"
        style={{ marginTop: 10, display: 'flex', gap: 4 }}
      >
        {(['all', 'plex', 'jellyfin', 'emby'] as TabFilter[]).map((t) => (
          <button
            key={t}
            type="button"
            className="empty-state-cta"
            onClick={() => setTab(t)}
            style={{
              opacity: tab === t ? 1 : 0.55,
              fontWeight: tab === t ? 600 : 400,
              fontSize: 12,
              padding: '4px 10px',
            }}
          >
            {t === 'all' ? 'All' : kindLabel(t)}
          </button>
        ))}
      </div>

      {/* 3 KPI cards — Datadog-style. Reuses kpiEnter + kpiStagger
          from lib/motion so the entrance matches the rest of the
          homelab tab strip. */}
      <motion.div
        className="homelab-services-grid"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{
          gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
          marginTop: 12,
        }}
      >
        <motion.div className="homelab-service-card" variants={kpiEnter}>
          <div className="homelab-service-card-header">
            <span className="homelab-service-card-name">Servers</span>
          </div>
          <div
            className="homelab-service-card-stat"
            style={{ fontSize: 22, fontWeight: 600 }}
          >
            {kpis.servers}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">
              pinned {tab === 'all' ? 'total' : kindLabel(tab)}
            </span>
          </div>
        </motion.div>
        <motion.div className="homelab-service-card" variants={kpiEnter}>
          <div className="homelab-service-card-header">
            <span className="homelab-service-card-name">Active sessions</span>
          </div>
          <div
            className="homelab-service-card-stat"
            style={{ fontSize: 22, fontWeight: 600 }}
          >
            {kpis.sessions}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">
              right now {tab === 'all' ? 'across all servers' : `on ${kindLabel(tab)}`}
            </span>
          </div>
        </motion.div>
        <motion.div className="homelab-service-card" variants={kpiEnter}>
          <div className="homelab-service-card-header">
            <span className="homelab-service-card-name">Recent additions</span>
          </div>
          <div
            className="homelab-service-card-stat"
            style={{ fontSize: 22, fontWeight: 600 }}
          >
            {kpis.recent}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">
              last few polls {tab === 'all' ? 'across all' : `on ${kindLabel(tab)}`}
            </span>
          </div>
        </motion.div>
      </motion.div>

      {/* Server chips — one per pinned server. The "all" tab
          shows every server; a kind-specific tab filters down. */}
      {filteredServers.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">
            {state && state.servers.length > 0
              ? `No ${kindLabel(tab as MediaServerKind)} servers pinned`
              : 'No media servers yet'}
          </p>
          <p className="homelab-services-empty-hint">
            Click <strong>+ Add server</strong> to pin a Plex / Jellyfin / Emby
            server. The worker polls every 60s and UPSERTs now-playing
            sessions + recent additions into the dashboard.
          </p>
        </div>
      ) : (
        <div
          className="homelab-services-grid"
          style={{
            gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
            marginTop: 16,
          }}
        >
          {filteredServers.map((s) => (
            <MediaServerCard key={s.id} server={s} onDelete={deleteServer} />
          ))}
        </div>
      )}

      <NowPlayingList sessions={filteredNowPlaying} />
      <RecentAdditionsCarousel items={filteredRecent} />

      <div className="homelab-services-footer">
        <span>
          {state?.servers.length ?? 0} server
          {(state?.servers.length ?? 0) === 1 ? '' : 's'} · 60s poll ·
          auto-refresh
        </span>
      </div>

      {modalOpen ? (
        <AddMediaModal
          form={form}
          setForm={setForm}
          busy={busy}
          error={formError}
          onClose={closeForm}
          onSubmit={() => void submitForm()}
        />
      ) : null}
    </motion.div>
  );
}
