import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, kpiEnter, kpiStagger, pageEnter } from '../../../lib/motion';
import Button from '../../shared/Button';
import AddDownloadModal from './AddDownloadModal';
import DownloadClientCard from './DownloadClientCard';
import { fmtBytes, fmtSpeed, kindLabel } from './downloadFormatters';
import type {
  DownloadClientFormState,
  DownloadClientRow,
  DownloadKind,
  DownloadsResponse,
} from './types';

/**
 * DownloadStatsWidget — Tier 10 Phase 5 (H5 — Download Stats).
 *
 * The Downloads tab content on the HomelabPage. Loads the caller's
 * pinned download clients (Sonarr/Radarr/qBittorrent/SABnzbd/
 * Lidarr/Readarr) via:
 *   GET /api/v1/homelab/downloads/clients — list + latest snapshot
 *
 * And writes via:
 *   POST   /api/v1/homelab/downloads/clients     — pin a new client (modal)
 *   DELETE /api/v1/homelab/downloads/clients/:id — unpin a client
 *
 * The DownloadsWorker (backend, internal/homelab/downloads.go)
 * ticks every 60s and INSERTs parsed-state rows into
 * homelab_download_snapshots. POST /clients fires an immediate
 * fire-and-forget poll after create so a freshly-pinned client
 * shows stats within seconds rather than waiting a full tick.
 *
 * Layout (Datadog-style):
 *   - Header: "+ Add client" + "Refresh" buttons
 *   - 3 KPI cards (Active queue / Download speed / Today downloaded)
 *   - Per-client cards (one per client with name + kind + status
 *     pill + 4 stat fields: queue count / queue bytes / up speed /
 *     down speed)
 *
 * Motion: pageEnter on the wrapper. kpiEnter + kpiStagger on the
 * KPI strip (reuses existing tokens — no new variants).
 *
 * Polling cadence: 60s — matches the worker. Dashboard data ages
 * in real-time without manual refresh, but the Refresh button
 * forces an immediate re-fetch.
 *
 * Formatter helpers (fmtBytes, fmtSpeed, kindLabel, relativeTime,
 * statusPillClass, statusPillDataAttr) live in downloadFormatters.ts
 * so this file stays under the 400-LOC cap.
 */

const POLL_INTERVAL_MS = 60_000;

const emptyForm: DownloadClientFormState = {
  name: '',
  kind: 'sonarr',
  base_url: '',
  api_key: '',
  username: '',
  password: '',
};

export default function DownloadStatsWidget({
  config: _config,
}: {
  config?: { poll_seconds?: number };
}) {
  void _config;
  const [clients, setClients] = useState<DownloadClientRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<DownloadClientFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your download stats.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api<DownloadsResponse>('GET', '/api/v1/homelab/downloads/clients');
      setClients(res.clients || []);
      setError('');
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // 60s polling — matches the worker tick cadence. Refresh button
  // forces an immediate re-fetch.
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
      const username = form.username.trim();
      const password = form.password;
      const kind = form.kind as DownloadKind;
      if (!name) {
        setFormError('Name is required.');
        return;
      }
      if (!baseURL) {
        setFormError('Base URL is required.');
        return;
      }
      if (kind === 'qbittorrent') {
        if (!apiKey && !(username && password)) {
          setFormError('qBittorrent requires api_key OR (username + password).');
          return;
        }
      } else if (!apiKey) {
        setFormError(`${kindLabel(kind)} requires an api_key.`);
        return;
      }
      const body: Record<string, unknown> = { name, kind, base_url: baseURL };
      if (apiKey) body.api_key = apiKey;
      if (username) body.username = username;
      if (password) body.password = password;
      await api('POST', '/api/v1/homelab/downloads/clients', body);
      setModalOpen(false);
      // The POST handler fires an immediate fire-and-forget poll;
      // give the worker ~1.5s for the INSERT to land before
      // reloading so the freshly-pinned client shows real stats.
      window.setTimeout(() => {
        void load();
      }, 1500);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, load]);

  const deleteClient = useCallback(
    async (c: DownloadClientRow) => {
      if (!window.confirm(`Remove client "${c.name}" and its snapshot history?`)) return;
      try {
        await api('DELETE', `/api/v1/homelab/downloads/clients/${c.id}`);
        await load();
      } catch (cause) {
        const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
        setError(msg);
      }
    },
    [load],
  );

  // Aggregate the KPI cards across all clients (latest snapshot
  // per client). 0 when nothing's pinned yet.
  const kpis = useMemo(() => {
    let queue = 0;
    let queueBytes = 0;
    let downSpeed = 0;
    let upSpeed = 0;
    let todayDown = 0;
    let todayUp = 0;
    for (const c of clients) {
      const s = c.latest_snapshot;
      if (!s) continue;
      queue += s.queue_count;
      queueBytes += s.queue_size_bytes;
      downSpeed += s.download_speed_bytes_per_sec;
      upSpeed += s.upload_speed_bytes_per_sec;
      todayDown += s.today_downloaded_bytes;
      todayUp += s.today_uploaded_bytes;
    }
    return { queue, queueBytes, downSpeed, upSpeed, todayDown, todayUp };
  }, [clients]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your download stats…
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

      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <Button variant="primary" size="sm" onClick={openForm}>
          + Add client
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
            <span className="homelab-service-card-name">Active queue</span>
          </div>
          <div className="homelab-service-card-stat" style={{ fontSize: 22, fontWeight: 600 }}>
            {kpis.queue}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">{fmtBytes(kpis.queueBytes)} remaining</span>
          </div>
        </motion.div>
        <motion.div className="homelab-service-card" variants={kpiEnter}>
          <div className="homelab-service-card-header">
            <span className="homelab-service-card-name">Download speed</span>
          </div>
          <div className="homelab-service-card-stat" style={{ fontSize: 22, fontWeight: 600 }}>
            {fmtSpeed(kpis.downSpeed)}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">↑ {fmtSpeed(kpis.upSpeed)}</span>
          </div>
        </motion.div>
        <motion.div className="homelab-service-card" variants={kpiEnter}>
          <div className="homelab-service-card-header">
            <span className="homelab-service-card-name">Today downloaded</span>
          </div>
          <div className="homelab-service-card-stat" style={{ fontSize: 22, fontWeight: 600 }}>
            {fmtBytes(kpis.todayDown)}
          </div>
          <div className="homelab-service-card-footer">
            <span className="homelab-service-latency">
              ↑ {fmtBytes(kpis.todayUp)} uploaded
            </span>
          </div>
        </motion.div>
      </motion.div>

      {clients.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">No download clients yet</p>
          <p className="homelab-services-empty-hint">
            Click <strong>+ Add client</strong> to pin a Sonarr / Radarr /
            qBittorrent / SABnzbd / Lidarr / Readarr server. The worker
            polls every 60s and upserts the latest queue + speed + today
            totals into the dashboard. qBittorrent accepts either an
            api_key or a username + password pair.
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
          {clients.map((c) => (
            <DownloadClientCard key={c.id} client={c} onDelete={deleteClient} />
          ))}
        </div>
      )}

      <div className="homelab-services-footer">
        <span>
          {clients.length} client{clients.length === 1 ? '' : 's'} · 60s poll · auto-refresh
        </span>
      </div>

      {modalOpen ? (
        <AddDownloadModal
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