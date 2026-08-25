import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, pageEnter } from '../../lib/motion';

/**
 * HomelabGrid — Tier 10 Phase 1 (H1 — Widget Framework).
 *
 * Lightweight CSS-Grid-based layout container for the HomelabPage.
 * Reads / writes the per-user `layout` JSONB blob via:
 *   GET  /api/v1/homelab/layout           — load current layout
 *   PUT  /api/v1/homelab/layout           — save updated layout
 *   GET  /api/v1/homelab/layout/defaults  — first-run default
 *
 * The layout array carries 7 widget descriptors (Phase 1 default):
 *
 *   Row 0: [KPI services | KPI notes | KPI todos | KPI rss]
 *   Row 1: [Services widget (6 col, 4 rows) | Notes widget (6 col, 4 rows)]
 *   Row 2: [Todos widget (12 col, 2 rows)]
 *
 * Each widget is rendered as a CSS-Grid-placed card. Phase 1 ships
 * placeholder content ("Services widget — Phase 2 will add pinned
 * service status") because the data sub-features are NOT in this
 * phase. Future phases drop real widget components in here based
 * on `widget.type`.
 *
 * Why CSS Grid (not react-grid-layout):
 *   - NO new dependencies allowed in Phase 1.
 *   - react-grid-layout is a future polish task (drag-resize UX).
 *   - The layout JSONB is wire-compatible with react-grid-layout
 *     (same {i, x, y, w, h, ...} shape) — switching the renderer
 *     later is a one-component swap, no backend change.
 *
 * Auto-save: layout changes are debounced 500ms before PUT, so the
 * server isn't hammered on every drag tick. The auto-save can be
 * disabled (manual save only) by passing `autoSave={false}`.
 */

export interface HomelabLayoutWidget {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
  type: string;
  config?: Record<string, unknown>;
}

interface HomelabLayoutResponse {
  layout: HomelabLayoutWidget[];
  updated_at?: string | null;
  is_default: boolean;
}

interface HomelabGridProps {
  /** Optional callback fired after a successful PUT with the saved layout. */
  onLayoutSaved?: (layout: HomelabLayoutWidget[]) => void;
  /** Optional callback fired when load errors occur. */
  onError?: (message: string) => void;
  /** Disable the PUT auto-save (manual mode). Default: auto. */
  autoSave?: boolean;
}

/**
 * Render a single placeholder widget card for Phase 1.
 *
 * Future phases replace this with real widget components based on
 * `widget.type` (a switch on `type` that maps to NotesWidget,
 * TodosWidget, etc.).
 */
function WidgetPlaceholder({ widget }: { widget: HomelabLayoutWidget }) {
  const phaseHint: Record<string, string> = {
    'kpi-services': 'KPI — Services pinned (Phase 2)',
    'kpi-notes': 'KPI — Notes (Phase 3)',
    'kpi-todos': 'KPI — Todos due-soon (Phase 3)',
    'kpi-rss': 'KPI — RSS unread (Phase 8)',
    services: 'Services widget — Phase 2 will add pinned service status',
    notes: 'Notes widget — Phase 3 will add Markdown notes editor',
    todos: 'Todos widget — Phase 3 will add priority + due-date todos',
  };
  const hint = phaseHint[widget.type] || `Widget: ${widget.type}`;

  return (
    <div
      className={`homelab-widget homelab-widget-${widget.type}`}
      style={{ gridColumn: `span ${widget.w}`, gridRow: `span ${widget.h}` }}
      data-widget-i={widget.i}
      data-widget-type={widget.type}
    >
      <div className="homelab-widget-header">
        <span className="homelab-widget-title">{widget.type}</span>
        <span className="homelab-widget-phase-tag">Phase 1 stub</span>
      </div>
      <div className="homelab-widget-body">
        <p className="homelab-widget-hint">{hint}</p>
      </div>
    </div>
  );
}

export default function HomelabGrid({ onLayoutSaved, onError, autoSave = true }: HomelabGridProps) {
  const [layout, setLayout] = useState<HomelabLayoutWidget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [isDefault, setIsDefault] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  // One-time mount: GET /layout (which itself returns the default on
  // first-run). Single round-trip; no polling yet — refresh interval
  // lives in Phase 2 (when services / notes have actual data).
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (!getToken()) {
        setError('Sign in to view your homelab layout.');
        setLoading(false);
        return;
      }
      try {
        const resp = await api<HomelabLayoutResponse>('GET', '/api/v1/homelab/layout');
        if (cancelled) return;
        setLayout(resp.layout || []);
        setIsDefault(!!resp.is_default);
        setError('');
      } catch (cause) {
        if (cancelled) return;
        const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
        setError(msg);
        onError?.(msg);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [onError]);

  // Debounced auto-save: when `dirty` flips true, schedule a PUT 500ms
  // later. The 500ms matches the spec — drag-resize UX typically
  // settles in <500ms after pointer-up, and PUT-every-tick would
  // spam the backend during a drag.
  useEffect(() => {
    if (!autoSave || !dirty || loading) return;
    const handle = window.setTimeout(() => {
      void saveLayout();
    }, 500);
    return () => window.clearTimeout(handle);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dirty, autoSave, loading]);

  const saveLayout = useCallback(async () => {
    setSaving(true);
    try {
      const resp = await api<HomelabLayoutResponse>('PUT', '/api/v1/homelab/layout', { layout });
      setIsDefault(false);
      setDirty(false);
      onLayoutSaved?.(resp.layout || layout);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
      onError?.(msg);
    } finally {
      setSaving(false);
    }
  }, [layout, onError, onLayoutSaved]);

  // The grid container is a 12-column CSS Grid with auto-placed
  // rows. Each widget specifies its grid-column span and grid-row
  // span via inline style. Rows are packed top-to-bottom in the
  // natural widget order, so y-position in the JSONB is mostly
  // informational for Phase 1 (drag UX isn't wired yet — Phase 2+
  // can either build a tiny drag system or add react-grid-layout
  // as an optional dep).
  const renderedWidgets = useMemo(() => {
    return layout.map((w) => <WidgetPlaceholder key={w.i} widget={w} />);
  }, [layout]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        <span>Loading your homelab layout…</span>
      </div>
    );
  }

  return (
    <motion.div
      className="homelab-grid"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      {error ? (
        <div className="dash-error" role="alert">
          {error}
        </div>
      ) : null}

      {isDefault ? (
        <div className="homelab-grid-hint" role="status">
          First run — showing the default homelab layout. Drag widgets (coming soon)
          or use the page header to reset. Your edits will auto-save.
        </div>
      ) : null}

      <div className="homelab-grid-canvas" aria-label="Homelab widget grid">
        {renderedWidgets}
      </div>

      <div className="homelab-grid-status" aria-live="polite">
        {saving
          ? 'Saving layout…'
          : dirty
            ? 'Unsaved changes (auto-save pending)…'
            : `${layout.length} widget${layout.length === 1 ? '' : 's'} · layout saved`}
      </div>

      {!autoSave ? (
        <div className="homelab-grid-actions">
          <button type="button" className="empty-state-cta" onClick={() => void saveLayout()} disabled={!dirty || saving}>
            {saving ? 'Saving…' : 'Save layout'}
          </button>
        </div>
      ) : null}
    </motion.div>
  );
}