import { useCallback, useEffect, useMemo, useState, type ReactElement } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import TodoModal, { datetimeLocalToRFC3339 } from './TodoModal';
import TodoCard from './TodoCard';
import type {
  DueSoonResponse,
  TodoFormState,
  TodoPriority,
  TodoRow,
  TodosResponse,
} from './types';

/**
 * TodosWidget — Tier 10 Phase 3 (H3 — Personal Todos).
 *
 * The Todos tab content on the HomelabPage. Loads the caller's
 * todos via:
 *   GET /api/v1/homelab/todos          — list (filtered by ?completed=)
 *   GET /api/v1/homelab/todos/due-soon — next 7 days (powers the
 *                                         "Due soon" section at top)
 *
 * And writes via:
 *   POST   /api/v1/homelab/todos          — create (modal)
 *   PATCH  /api/v1/homelab/todos/:id      — edit (modal)
 *   DELETE /api/v1/homelab/todos/:id      — delete (confirm)
 *   POST   /api/v1/homelab/todos/:id/complete — inline checkbox
 *
 * Filter pills (top-right): All / Active / Done.
 * Priority pills (inline): muted=low, blue=medium, amber=high, red=urgent.
 * Overdue items get a red left border.
 *
 * Motion: pageEnter on the wrapper. Reuses existing motion tokens.
 */

const emptyForm: TodoFormState = {
  title: '',
  description: '',
  priority: 'medium',
  dueDate: '',
  tagsRaw: '',
};

const splitTags = (raw: string): string[] =>
  raw
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.length > 0);

interface TodosWidgetProps {
  /** Reserved for future config. */
  config?: { limit?: number };
}

type Filter = 'all' | 'active' | 'done';

export default function TodosWidget({ config: _config }: TodosWidgetProps) {
  void _config;
  const [todos, setTodos] = useState<TodoRow[]>([]);
  const [dueSoon, setDueSoon] = useState<DueSoonResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<Filter>('active');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<TodoFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');

  const load = useCallback(async (f: Filter) => {
    if (!getToken()) {
      setError('Sign in to view your todos.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const completedParam = f === 'active' ? '?completed=false' : f === 'done' ? '?completed=true' : '';
      const [list, dueSoonResp] = await Promise.all([
        api<TodosResponse>('GET', `/api/v1/homelab/todos${completedParam}`),
        api<DueSoonResponse>('GET', '/api/v1/homelab/todos/due-soon'),
      ]);
      setTodos(list.todos || []);
      setDueSoon(dueSoonResp);
      setError('');
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(filter);
  }, [load, filter]);

  const openCreate = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setFormError('');
    setModalOpen(true);
  }, []);

  const openEdit = useCallback((t: TodoRow) => {
    setEditingId(t.id);
    // The server returns due_date as RFC3339 ("2025-12-31T23:59:00Z").
    // datetime-local wants "2025-12-31T23:59" (no Z, no seconds).
    // Trim the seconds + Z so the input populates cleanly.
    let dueLocal = '';
    if (t.due_date) {
      const m = t.due_date.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2})/);
      if (m) dueLocal = m[1];
    }
    setForm({
      title: t.title,
      description: t.description ?? '',
      priority: (t.priority ?? 'medium') as TodoPriority,
      dueDate: dueLocal,
      tagsRaw: (t.tags || []).join(', '),
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
      const dueDate = datetimeLocalToRFC3339(form.dueDate);
      const body: {
        title: string;
        description?: string;
        priority: TodoPriority;
        due_date?: string | null;
        tags: string[];
      } = {
        title: form.title.trim(),
        priority: form.priority,
        tags,
      };
      if (form.description.trim()) body.description = form.description;
      if (dueDate) body.due_date = dueDate;
      if (editingId) {
        await api('PATCH', `/api/v1/homelab/todos/${editingId}`, body);
      } else {
        await api('POST', '/api/v1/homelab/todos', body);
      }
      setModalOpen(false);
      await load(filter);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }, [form, editingId, load, filter]);

  const deleteTodo = useCallback(async (id: string) => {
    if (!window.confirm('Delete this todo?')) return;
    try {
      await api('DELETE', `/api/v1/homelab/todos/${id}`);
      await load(filter);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load, filter]);

  const completeTodo = useCallback(async (id: string) => {
    try {
      await api('POST', `/api/v1/homelab/todos/${id}/complete`);
      await load(filter);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load, filter]);

  const toggleComplete = useCallback(async (t: TodoRow) => {
    if (t.completed_at) {
      // Re-open via PATCH with mark_complete: false
      try {
        await api('PATCH', `/api/v1/homelab/todos/${t.id}`, { mark_complete: false });
        await load(filter);
      } catch (cause) {
        const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
        setError(msg);
      }
    } else {
      await completeTodo(t.id);
    }
  }, [completeTodo, load, filter]);

  // Build the list — due-soon section first (if any + active filter),
  // then the regular list (deduped). When filter is 'done', skip the
  // due-soon section.
  const sections = useMemo<ReactElement[]>(() => {
    const sectionElems: ReactElement[] = [];
    const dueSoonTodos = dueSoon?.todos || [];
    const dueSoonIds = new Set(dueSoonTodos.map((t) => t.id));
    const overdueCount = dueSoon?.overdue_count ?? 0;

    if (filter === 'active' && dueSoonTodos.length > 0) {
      sectionElems.push(
        <div key="due-soon" style={{ marginBottom: 16 }}>
          <div style={{
            display: 'flex', gap: 8, alignItems: 'center',
            fontSize: 12, color: 'var(--text-muted, #94a3b8)',
            marginBottom: 8, textTransform: 'uppercase', letterSpacing: '0.04em',
          }}>
            <strong>Due soon</strong>
            <span>·</span>
            <span>{dueSoonTodos.length} in next {dueSoon?.window_days ?? 7}d</span>
            {overdueCount > 0 ? (
              <span style={{ color: 'var(--red, #ef4444)' }}>
                · {overdueCount} overdue
              </span>
            ) : null}
          </div>
          <div className="homelab-services-grid">
            {dueSoonTodos.map((t) => (
              <TodoCard
                key={t.id}
                todo={t}
                onToggle={() => void toggleComplete(t)}
                onEdit={() => openEdit(t)}
                onDelete={() => void deleteTodo(t.id)}
                onComplete={() => void completeTodo(t.id)}
              />
            ))}
          </div>
        </div>
      );
    }

    const rest = todos.filter((t) => !dueSoonIds.has(t.id));
    sectionElems.push(
      <div key="all">
        {filter === 'active' && dueSoonTodos.length > 0 ? (
          <div style={{
            fontSize: 12, color: 'var(--text-muted, #94a3b8)',
            marginBottom: 8, textTransform: 'uppercase', letterSpacing: '0.04em',
          }}>
            <strong>Other active</strong>
          </div>
        ) : null}
        {rest.length === 0 && dueSoonTodos.length === 0 ? (
          <div className="homelab-services-empty">
            <p className="homelab-services-empty-title">
              {filter === 'done' ? 'No completed todos' : 'No todos yet'}
            </p>
            <p className="homelab-services-empty-hint">
              {filter === 'done'
                ? 'Mark a todo complete and it\'ll show up here.'
                : 'Track homelab tasks with priority + due date. Active todos float to the top.'}
            </p>
          </div>
        ) : rest.length === 0 ? null : (
          <div className="homelab-services-grid">
            {rest.map((t) => (
              <TodoCard
                key={t.id}
                todo={t}
                onToggle={() => void toggleComplete(t)}
                onEdit={() => openEdit(t)}
                onDelete={() => void deleteTodo(t.id)}
                onComplete={() => void completeTodo(t.id)}
              />
            ))}
          </div>
        )}
      </div>
    );

    return sectionElems;
  }, [todos, dueSoon, filter, toggleComplete, openEdit, deleteTodo, completeTodo]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your todos…
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
        <button type="button" className="empty-state-cta" onClick={openCreate}>
          + New todo
        </button>
        <div style={{ display: 'flex', gap: 4, marginLeft: 'auto' }}>
          {(['all', 'active', 'done'] as Filter[]).map((f) => (
            <button
              key={f}
              type="button"
              className={`logs-tab ${filter === f ? 'logs-tab-active' : ''}`}
              onClick={() => setFilter(f)}
              style={{ textTransform: 'capitalize' }}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {sections}

      <div className="homelab-services-footer">
        <span>{todos.length} todo{todos.length === 1 ? '' : 's'} loaded</span>
        {dueSoon ? (
          <span>
            {dueSoon.overdue_count > 0
              ? `${dueSoon.overdue_count} overdue`
              : 'nothing overdue'}
          </span>
        ) : null}
      </div>

      {modalOpen ? (
        <TodoModal
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
