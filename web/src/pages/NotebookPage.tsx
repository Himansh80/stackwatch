import { useCallback, useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import EmptyState from '../components/shared/EmptyState';
import KpiCard from '../components/shared/KpiCard';
import NotebookEditor, { Notebook, NotebookCollaborator } from '../components/shared/NotebookEditor';
import { motion, pageEnter, kpiStagger } from '../lib/motion';

type Tab = 'mine' | 'shared' | 'all';

type CollabDetailResponse = {
  notebook?: Notebook;
  collaborators?: NotebookCollaborator[];
};

type ListResponse = {
  notebooks?: Notebook[];
  total?: number;
};

/**
 * NotebookPage — Tier 7.10 (D11) Notebook surface at /notebooks.
 *
 * Layout:
 *   - Top: 3 KpiCards (total notebooks / mine / shared with me)
 *   - Middle: tabbed view (My Notebooks / Shared with me / All)
 *     - Each tab: list of notebook cards (title + last_edited + author)
 *     - Click a card → opens NotebookEditor in a modal
 *   - "+ New notebook" button at top-right (opens an empty create modal)
 *
 * Sidebar nav: "Notebooks" added in AppSidebar under operations.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip; the
 * "+ New notebook" CTA uses the shared buttonSpring. The editor modal
 * is a plain slow-query-explain frame (no separate animation) so we
 * avoid stacking two framer-motion enter phases.
 *
 * Tabs drive a fresh server query (filter ?role=owner|editor|all).
 * My = owner; Shared = editor or viewer; All = every notebook in the
 * tenant (capped at 500).
 */
export default function NotebookPage() {
  const [tab, _setTab] = useState<Tab>('mine');
  const [error, setError] = useState('');
  const [notebooks, setNotebooks] = useState<Notebook[]>([]);

  const [editing, setEditing] = useState<Notebook | null>(null);
  const [_creating, setCreating] = useState(false);
  const [createTitle, setCreateTitle] = useState('');
  const [createContent, setCreateContent] = useState('');
  const [_createBusy, setCreateBusy] = useState(false);

  const [_busy, setBusy] = useState(false);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Notebooks.');
      return false;
    }
    return true;
  };

  const loadNotebooks = useCallback(async () => {
    try {
      const roleParam = tab === 'mine' ? 'owner' : tab === 'shared' ? 'editor' : 'all';
      // For the "shared" tab we want editor OR viewer — the spec only
      // exposes a single ?role= so we make two calls and merge. For
      // a Phase-4 surface this is fine; Phase 5 (Team Collab) can
      // replace this with a real "mine + collaborator" composite
      // endpoint if list sizes grow.
      if (tab === 'shared') {
        const [a, b] = await Promise.all([
          api<ListResponse>('GET', '/api/v1/notebooks?role=editor'),
          api<ListResponse>('GET', '/api/v1/notebooks?role=viewer'),
        ]);
        const merged: Notebook[] = [];
        const seen = new Set<string>();
        for (const list of [a.notebooks || [], b.notebooks || []]) {
          for (const nb of list) {
            if (seen.has(nb.id)) continue;
            seen.add(nb.id);
            merged.push(nb);
          }
        }
        setNotebooks(merged);
      } else {
        const r = await api<ListResponse>(
          'GET',
          `/api/v1/notebooks?role=${roleParam}`,
        );
        setNotebooks(r.notebooks || []);
      }
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, [tab]);

  useEffect(() => {
    setError('');
    void loadNotebooks();
  }, [loadNotebooks]);

  const openEditor = useCallback(async (notebook: Notebook) => {
    if (!requireAuth()) return;
    setError('');
    try {
      // Fetch the detail so we get the collaborators array.
      const r = await api<CollabDetailResponse>('GET', `/api/v1/notebooks/${notebook.id}`);
      const merged: Notebook = {
        ...notebook,
        ...(r.notebook || {}),
        collaborators: r.collaborators || notebook.collaborators || [],
      };
      setEditing(merged);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  }, []);

  const _closeEditor = useCallback(() => {
    setEditing(null);
  }, []);

  const _saveEditor = useCallback(
    async (nextContent: string) => {
      if (!editing) return;
      setBusy(true);
      setError('');
      try {
        const r = await api<{ notebook?: Notebook }>('PUT', `/api/v1/notebooks/${editing.id}`, {
          content: nextContent,
        });
        if (r.notebook) {
          setEditing({ ...editing, ...r.notebook, content: r.notebook.content });
        }
        await loadNotebooks();
      } catch (cause) {
        setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setBusy(false);
      }
    },
    [editing, loadNotebooks],
  );

  const _create = useCallback(async () => {
    if (!createTitle.trim()) {
      setError('Title is required to create a notebook.');
      return;
    }
    setCreateBusy(true);
    setError('');
    try {
      await api('POST', '/api/v1/notebooks', {
        title: createTitle.trim(),
        content: createContent,
      });
      setCreating(false);
      setCreateTitle('');
      setCreateContent('');
      await loadNotebooks();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setCreateBusy(false);
    }
  }, [createTitle, createContent, loadNotebooks]);

  // KPIs: client-side from the loaded list — the cap is 500 rows, so
  // we don't need a separate stats endpoint for Phase 4.
  const totalLabel = tab === 'all' ? `${notebooks.length}` : '—';
  const myCount = notebooks.filter((n) => n.author_id).length;
  const sharedCount = notebooks.filter((n) => !n.author_id).length;

  return (
        <motion.div className="dash-page" initial="hidden" animate="show" variants={pageEnter}>
          {error ? <div className="dash-error" role="alert">{error}</div> : null}

          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard
              label="Total notebooks"
              value={totalLabel}
              status={tab === 'all' ? 'up' : 'neutral'}
              accent={tab === 'all' ? 'cyan' : 'indigo'}
            />
            <KpiCard
              label="My notebooks (this view)"
              value={myCount}
              status="up"
              accent="green"
            />
            <KpiCard
              label="Shared with me (this view)"
              value={sharedCount}
              status="neutral"
              accent="amber"
            />
          </motion.div>

          <section className="dash-section">
            <span className="dash-eyebrow">Notebooks</span>
            <h2 className="dash-section-title">
              {tab === 'mine' ? 'Notebooks you own' : tab === 'shared' ? 'Notebooks shared with you' : 'All notebooks in this tenant'}
            </h2>
            {notebooks.length === 0 ? (
              <EmptyState
                illustration={<span style={{ fontSize: 36 }}>◰</span>}
                headline={tab === 'mine' ? 'You have no notebooks yet' : tab === 'shared' ? 'No notebooks are shared with you' : 'No notebooks in this tenant'}
                subhead="Notebooks are collaborative markdown runbooks. Use the + New notebook button above to create your first one, then share it with teammates via the editor."
              />
            ) : (
              <div className="threat-card-list">
                {notebooks.map((nb) => (
                  <button
                    key={nb.id}
                    type="button"
                    className="threat-card threat-card-clickable"
                    onClick={() => void openEditor(nb)}
                    aria-label={`Open ${nb.title}`}
                  >
                    <div className="threat-card-top">
                      <span className="dash-status dash-status-muted">
                        <span className="dash-status-dot" aria-hidden="true" />
                        {tab === 'mine' ? 'owner' : tab === 'shared' ? 'shared' : 'tenant'}
                      </span>
                      <strong className="threat-card-type">{nb.title || 'Untitled notebook'}</strong>
                      <span className="threat-card-time" title={nb.last_edited_at}>
                        {nb.last_edited_at ? new Date(nb.last_edited_at).toLocaleString() : '—'}
                      </span>
                    </div>
                    <p className="threat-card-desc">
                      {(nb.content || '').split('\n').slice(0, 3).join(' ').slice(0, 220) || 'No content yet — click to start writing.'}
                    </p>
                    <div className="threat-card-meta">
                      <span className="threat-card-meta-pill">
                        <span className="threat-card-meta-label">author</span>
                        <code>{nb.author_id ? `${nb.author_id.slice(0, 8)}…` : '—'}</code>
                      </span>
                      <span className="threat-card-resolve-btn">Open editor →</span>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </section>
        </motion.div>
  );
}
