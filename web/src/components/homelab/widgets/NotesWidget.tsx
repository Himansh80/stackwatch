import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import Button from '../../shared/Button';
import NoteModal from './NoteModal';
import type { NoteFormState, NoteRow, NotesResponse } from './types';

/**
 * NotesWidget — Tier 10 Phase 3 (H3 — Personal Notes).
 *
 * The Notes tab content on the HomelabPage. Loads the caller's
 * notes via:
 *   GET /api/v1/homelab/notes         — list of notes
 *   GET /api/v1/homelab/notes/search  — search by q (used by search box)
 *
 * And writes via:
 *   POST   /api/v1/homelab/notes      — create (modal)
 *   PATCH  /api/v1/homelab/notes/:id  — edit (modal)
 *   DELETE /api/v1/homelab/notes/:id  — delete (confirm)
 *   PATCH  /api/v1/homelab/notes/:id  — toggle pinned (inline)
 *
 * Card list (Datadog-style):
 *   - title (bold) + pin indicator
 *   - tags (small badges)
 *   - body preview (first 100 chars)
 *   - updated_at (relative — "2h ago")
 *   - hover reveals edit + delete buttons
 *
 * Modal uses NoteModal (shared with edit mode). Markdown is NOT
 * rendered client-side — react-markdown isn't a dep, so the body
 * preview is plain text. The note detail view shows the raw body
 * inside a <pre> for now; a Markdown renderer can be added later
 * when the dep is in.
 *
 * Motion: pageEnter on the wrapper. Reuses existing motion tokens;
 * no new variants.
 */

const emptyForm: NoteFormState = { title: '', body: '', tagsRaw: '', pinned: false };

const bodyPreview = (s: string): string => {
  if (s.length <= 100) return s;
  return `${s.slice(0, 100)}…`;
};

const relativeTime = (iso: string): string => {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '';
  const diffSec = Math.round((Date.now() - t) / 1000);
  if (diffSec < 60) return 'just now';
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  if (diffSec < 86400 * 7) return `${Math.floor(diffSec / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
};

const splitTags = (raw: string): string[] =>
  raw
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.length > 0);

interface NotesWidgetProps {
  /** Reserved for future config (filter, sort). */
  config?: { limit?: number };
}

export default function NotesWidget({ config: _config }: NotesWidgetProps) {
  void _config;
  const [notes, setNotes] = useState<NoteRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchQ, setSearchQ] = useState('');
  const [appliedQuery, setAppliedQuery] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<NoteFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');

  const load = useCallback(async (q?: string) => {
    if (!getToken()) {
      setError('Sign in to view your notes.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      // Use the search endpoint when there's an applied query;
      // otherwise use the list endpoint (which honors the same
      // ?tag= filter if we ever add it).
      const path = q && q.trim()
        ? `/api/v1/homelab/notes/search?q=${encodeURIComponent(q.trim())}`
        : '/api/v1/homelab/notes';
      const resp = await api<NotesResponse>('GET', path);
      setNotes(resp.notes || []);
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

  const openCreate = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setFormError('');
    setModalOpen(true);
  }, []);

  const openEdit = useCallback((n: NoteRow) => {
    setEditingId(n.id);
    setForm({
      title: n.title,
      body: n.body,
      tagsRaw: (n.tags || []).join(', '),
      pinned: n.pinned,
    });
    setFormError('');
    setModalOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setModalOpen(false);
  }, []);

  const submitForm = useCallback(async () => {
    setBusy(true);
    setFormError('');
    try {
      const tags = splitTags(form.tagsRaw);
      const body: {
        title: string;
        body: string;
        tags: string[];
        pinned: boolean;
      } = {
        title: form.title.trim(),
        body: form.body,
        tags,
        pinned: form.pinned,
      };
      if (editingId) {
        await api('PATCH', `/api/v1/homelab/notes/${editingId}`, body);
      } else {
        await api('POST', '/api/v1/homelab/notes', body);
      }
      setModalOpen(false);
      await load(appliedQuery);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, editingId, load, appliedQuery]);

  const deleteNote = useCallback(async (id: string) => {
    if (!window.confirm('Delete this note? This cannot be undone.')) return;
    try {
      await api('DELETE', `/api/v1/homelab/notes/${id}`);
      await load(appliedQuery);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load, appliedQuery]);

  const togglePinned = useCallback(async (n: NoteRow) => {
    try {
      await api('PATCH', `/api/v1/homelab/notes/${n.id}`, { pinned: !n.pinned });
      await load(appliedQuery);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load, appliedQuery]);

  const runSearch = useCallback(() => {
    setAppliedQuery(searchQ);
    void load(searchQ);
  }, [searchQ, load]);

  const clearSearch = useCallback(() => {
    setSearchQ('');
    setAppliedQuery('');
    void load();
  }, [load]);

  const cards = useMemo(() => notes.map((n) => (
    <article
      key={n.id}
      className="homelab-service-card"
      data-status={n.pinned ? 'up' : 'unknown'}
      data-disabled={n.pinned ? 'false' : 'true'}
      style={{ textAlign: 'left', cursor: 'default' }}
    >
      <div className="homelab-service-card-header">
        <span className="homelab-service-card-name">{n.title}</span>
        {n.pinned ? <span title="Pinned" aria-label="Pinned">📌</span> : null}
      </div>
      {n.body ? (
        <span className="homelab-service-card-url" style={{ whiteSpace: 'pre-wrap' }}>
          {bodyPreview(n.body)}
        </span>
      ) : null}
      {n.tags && n.tags.length > 0 ? (
        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
          {n.tags.map((t) => (
            <span
              key={t}
              className="homelab-service-pill"
              data-status="unknown"
              style={{ fontSize: 10, padding: '2px 6px' }}
            >
              {t}
            </span>
          ))}
        </div>
      ) : null}
      <div className="homelab-service-card-footer">
        <span className="homelab-service-latency">{relativeTime(n.updated_at)}</span>
      </div>
      <div
        className="homelab-service-card-actions"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => void togglePinned(n)}
          title={n.pinned ? 'Unpin' : 'Pin to top'}
        >
          {n.pinned ? 'Unpin' : 'Pin'}
        </button>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => openEdit(n)}
          title="Edit"
        >
          Edit
        </button>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={() => void deleteNote(n.id)}
          title="Delete"
        >
          ×
        </button>
      </div>
    </article>
  )), [notes, togglePinned, openEdit, deleteNote]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your notes…
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
        <Button variant="primary" size="sm" onClick={openCreate}>
          + New note
        </Button>
        <div style={{ display: 'flex', gap: 4, alignItems: 'center', marginLeft: 'auto' }}>
          <input
            type="search"
            placeholder="Search title / body / tag…"
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                runSearch();
              }
            }}
            style={{
              background: 'var(--surface-2)',
              color: 'var(--text)',
              border: '1px solid var(--surface-3)',
              borderRadius: 8,
              padding: '8px 10px',
              font: 'inherit',
              minWidth: 200,
            }}
          />
          <Button
            variant="primary"
            size="sm"
            onClick={runSearch}
            disabled={!searchQ.trim()}
          >
            Search
          </Button>
          {appliedQuery ? (
            <Button variant="ghost" size="sm" onClick={clearSearch}>
              Clear
            </Button>
          ) : null}
        </div>
      </div>

      {notes.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">
            {appliedQuery ? 'No matching notes' : 'No notes yet'}
          </p>
          <p className="homelab-services-empty-hint">
            {appliedQuery
              ? 'Try a different search or clear the filter to see all your notes.'
              : 'Capture homelab ideas, links, and reminders. Notes are private to you — your co-founder\'s notes live in their own list.'}
          </p>
        </div>
      ) : (
        <div className="homelab-services-grid">{cards}</div>
      )}

      <div className="homelab-services-footer">
        <span>
          {notes.length} note{notes.length === 1 ? '' : 's'}
          {appliedQuery ? ` matching "${appliedQuery}"` : ''}
        </span>
      </div>

      {modalOpen ? (
        <NoteModal
          form={form}
          setForm={setForm}
          busy={busy}
          error={formError}
          editingId={editingId}
          onClose={closeModal}
          onSubmit={() => void submitForm()}
        />
      ) : null}
    </motion.div>
  );
}
