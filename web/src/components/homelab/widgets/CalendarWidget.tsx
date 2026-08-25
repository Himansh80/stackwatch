import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import AddCalendarModal from './AddCalendarModal';
import EventDetailModal from './EventDetailModal';
import {
  formatTime,
  isoFromDate,
  relativeTime,
  sameDay,
  startOfWeek,
} from './calendarTimeHelpers';
import type {
  CalendarFormState,
  CalendarRow,
  CalendarsResponse,
  EventRow,
  EventsResponse,
} from './types';

/**
 * CalendarWidget — Tier 10 Phase 4 (H4 — Calendar).
 *
 * The Calendar tab content on the HomelabPage. Loads the caller's
 * iCal subscriptions + upcoming events via:
 *   GET /api/v1/homelab/calendars         — list of pinned feeds
 *   GET /api/v1/homelab/calendars/events  — upcoming events (default 30d)
 *
 * And writes via:
 *   POST   /api/v1/homelab/calendars              — add a new feed
 *   POST   /api/v1/homelab/calendars/:id/refresh  — manual sync
 *   DELETE /api/v1/homelab/calendars/:id          — remove a feed
 *
 * Layout (Datadog-style):
 *   - Header: "+ Add calendar" + "Refresh all"
 *   - List of calendars as colored cards (one per feed with
 *     name, event count, last-synced timestamp, refresh +
 *     delete buttons)
 *   - Week grid (Mon-Sun) below with event chips colored from
 *     their parent calendar; click → EventDetailModal
 *
 * Motion: pageEnter on the wrapper. Reuses the tokens defined in
 * lib/motion.tsx — no new variants.
 */

const DAY_NAMES_SHORT = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const emptyForm: CalendarFormState = {
  name: '',
  ical_url: '',
  color: 'default',
};

export default function CalendarWidget({ config: _config }: { config?: { calendar_id?: string } }) {
  void _config;
  const [calendars, setCalendars] = useState<CalendarRow[]>([]);
  const [events, setEvents] = useState<EventRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<CalendarFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');
  const [refreshBusyId, setRefreshBusyId] = useState<string | null>(null);
  const [detailEvent, setDetailEvent] = useState<EventRow | null>(null);

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your calendar.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const today = new Date();
      const in30 = new Date(today.getTime() + 30 * 86400 * 1000);
      const params = `from=${encodeURIComponent(isoFromDate(today))}&to=${encodeURIComponent(isoFromDate(in30))}`;
      const [cals, evs] = await Promise.all([
        api<CalendarsResponse>('GET', '/api/v1/homelab/calendars'),
        api<EventsResponse>('GET', `/api/v1/homelab/calendars/events?${params}`),
      ]);
      setCalendars(cals.calendars || []);
      setEvents(evs.events || []);
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
      const body: { name: string; ical_url: string; color: string } = {
        name: form.name.trim(),
        ical_url: form.ical_url.trim(),
        color: form.color,
      };
      await api('POST', '/api/v1/homelab/calendars', body);
      setModalOpen(false);
      // The POST handler fires an immediate sync; give the
      // dashboard ~1.5s for the upsert to land before reloading.
      window.setTimeout(() => { void load(); }, 1500);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, load]);

  const refreshOne = useCallback(async (id: string) => {
    setRefreshBusyId(id);
    try {
      await api('POST', `/api/v1/homelab/calendars/${id}/refresh`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setRefreshBusyId(null);
    }
  }, [load]);

  const deleteCalendar = useCallback(async (c: CalendarRow) => {
    if (!window.confirm(`Remove calendar "${c.name}" and its events?`)) return;
    try {
      await api('DELETE', `/api/v1/homelab/calendars/${c.id}`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load]);

  const refreshAll = useCallback(async () => {
    if (calendars.length === 0) return;
    for (const c of calendars) {
      void api('POST', `/api/v1/homelab/calendars/${c.id}/refresh`).catch(() => undefined);
    }
    window.setTimeout(() => { void load(); }, 2500);
  }, [calendars, load]);

  const weekStart = useMemo(() => startOfWeek(new Date()), []);
  const weekDays = useMemo(() => {
    return Array.from({ length: 7 }, (_, i) => {
      const d = new Date(weekStart);
      d.setDate(d.getDate() + i);
      return d;
    });
  }, [weekStart]);

  const colorById = useMemo(() => {
    const out: Record<string, string> = {};
    for (const c of calendars) out[c.id] = c.color || '#3b82f6';
    return out;
  }, [calendars]);

  const nameById = useMemo(() => {
    const out: Record<string, string> = {};
    for (const c of calendars) out[c.id] = c.name;
    return out;
  }, [calendars]);

  const eventsByDay = useMemo(() => {
    const out: EventRow[][] = [[], [], [], [], [], [], []];
    const weekEnd = new Date(weekStart);
    weekEnd.setDate(weekEnd.getDate() + 7);
    for (const e of events) {
      const t = new Date(e.starts_at);
      if (!Number.isFinite(t.getTime())) continue;
      if (t < weekStart || t >= weekEnd) continue;
      const idx = Math.floor((t.getTime() - weekStart.getTime()) / 86400000);
      if (idx >= 0 && idx < 7) out[idx].push(e);
    }
    return out;
  }, [events, weekStart]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your calendar…
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
      {error ? <div className="dash-error" role="alert">{error}</div> : null}

      <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
        <button type="button" className="empty-state-cta" onClick={openForm}>
          + Add calendar
        </button>
        <button
          type="button"
          className="empty-state-cta"
          onClick={() => void refreshAll()}
          disabled={calendars.length === 0}
        >
          Refresh all
        </button>
      </div>

      {calendars.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">No calendars yet</p>
          <p className="homelab-services-empty-hint">
            Paste an iCal URL (Google Calendar → Settings → Integrate
            calendar → Secret address in iCal format). The worker
            syncs every feed every 30min and upserts events into the
            database.
          </p>
        </div>
      ) : (
        <div
          className="homelab-services-grid"
          style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))' }}
        >
          {calendars.map((c) => (
            <article
              key={c.id}
              className="homelab-service-card"
              data-status="unknown"
              style={{
                borderLeftColor: c.color || '#3b82f6',
                cursor: 'default',
              }}
            >
              <div className="homelab-service-card-header">
                <span className="homelab-service-card-name">{c.name}</span>
                <span
                  className="homelab-service-pill"
                  data-status="unknown"
                  style={{ fontSize: 10, padding: '2px 6px' }}
                >
                  {c.event_count} event{c.event_count === 1 ? '' : 's'}
                </span>
              </div>
              <span className="homelab-service-card-url" style={{ wordBreak: 'break-all' }}>
                {c.ical_url}
              </span>
              <div className="homelab-service-card-footer">
                <span className="homelab-service-latency">
                  {c.last_synced_at
                    ? `synced ${relativeTime(c.last_synced_at)}`
                    : 'never synced'}
                </span>
                {c.last_sync_status === 'error' && c.last_sync_error ? (
                  <span title={c.last_sync_error} style={{ color: 'var(--red, #ef4444)' }}>
                    ⚠
                  </span>
                ) : null}
              </div>
              <div
                className="homelab-service-card-actions"
                onClick={(e) => e.stopPropagation()}
              >
                <button
                  type="button"
                  className="homelab-service-action-btn"
                  onClick={() => void refreshOne(c.id)}
                  disabled={refreshBusyId === c.id}
                  title="Sync now"
                >
                  {refreshBusyId === c.id ? '…' : 'Refresh'}
                </button>
                <button
                  type="button"
                  className="homelab-service-action-btn"
                  onClick={() => void deleteCalendar(c)}
                  title="Remove"
                >
                  ×
                </button>
              </div>
            </article>
          ))}
        </div>
      )}

      {events.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">No events scheduled</p>
          <p className="homelab-services-empty-hint">
            Add an iCal URL above and events from the next 30 days
            will appear here.
          </p>
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(7, 1fr)',
            gap: 8,
            marginTop: 16,
          }}
        >
          {weekDays.map((d, idx) => (
            <div
              key={d.toISOString()}
              style={{
                background: sameDay(d, new Date()) ? 'var(--surface-2)' : 'transparent',
                border: '1px solid var(--surface-3)',
                borderRadius: 8,
                padding: 8,
                minHeight: 120,
              }}
            >
              <div style={{ fontSize: 11, fontWeight: 600, opacity: 0.7, marginBottom: 6 }}>
                {DAY_NAMES_SHORT[idx]} {d.getDate()}
              </div>
              {eventsByDay[idx].length === 0 ? (
                <div style={{ fontSize: 11, opacity: 0.4 }}>—</div>
              ) : (
                eventsByDay[idx].map((e) => {
                  const color = colorById[e.calendar_id] || '#3b82f6';
                  return (
                    <button
                      key={e.id}
                      type="button"
                      onClick={() => setDetailEvent(e)}
                      style={{
                        display: 'block',
                        width: '100%',
                        textAlign: 'left',
                        background: color,
                        color: '#fff',
                        border: 'none',
                        borderRadius: 4,
                        padding: '4px 6px',
                        marginBottom: 4,
                        fontSize: 11,
                        cursor: 'pointer',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                      title={`${e.summary} (${nameById[e.calendar_id] || 'calendar'})`}
                    >
                      {e.all_day ? '⏳ ' : `${formatTime(e.starts_at)} `}
                      {e.summary}
                    </button>
                  );
                })
              )}
            </div>
          ))}
        </div>
      )}

      <div className="homelab-services-footer">
        <span>{events.length} event{events.length === 1 ? '' : 's'} in next 30d</span>
      </div>

      {modalOpen ? (
        <AddCalendarModal
          form={form}
          setForm={setForm}
          busy={busy}
          error={formError}
          onClose={closeForm}
          onSubmit={() => void submitForm()}
        />
      ) : null}

      {detailEvent ? (
        <EventDetailModal
          event={detailEvent}
          calendarName={nameById[detailEvent.calendar_id] || ''}
          calendarColor={colorById[detailEvent.calendar_id] || '#3b82f6'}
          onClose={() => setDetailEvent(null)}
        />
      ) : null}
    </motion.div>
  );
}
