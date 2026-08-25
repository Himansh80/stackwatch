import type { TodoRow } from './types';

/**
 * TodoCard — one card per todo. Hover reveals the complete / edit /
 * delete actions. Click on the checkbox toggles completion.
 *
 * The card is dumb on purpose: it just renders props and fires
 * callbacks. The parent (TodosWidget) owns the API calls.
 */

const relativeTime = (iso: string | null | undefined): string => {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '';
  const diffSec = Math.round((t - Date.now()) / 1000);
  const abs = Math.abs(diffSec);
  if (abs < 60) return diffSec < 0 ? 'just now' : 'soon';
  if (abs < 3600) {
    const m = Math.floor(abs / 60);
    return diffSec < 0 ? `${m}m overdue` : `in ${m}m`;
  }
  if (abs < 86400) {
    const h = Math.floor(abs / 3600);
    return diffSec < 0 ? `${h}h overdue` : `in ${h}h`;
  }
  const d = Math.floor(abs / 86400);
  return diffSec < 0 ? `${d}d overdue` : `in ${d}d`;
};

const isOverdue = (due: string | null | undefined, completedAt: string | null | undefined): boolean => {
  if (!due || completedAt) return false;
  return new Date(due).getTime() < Date.now();
};

interface TodoCardProps {
  todo: TodoRow;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onComplete: () => void;
}

export default function TodoCard({
  todo, onToggle, onEdit, onDelete, onComplete,
}: TodoCardProps) {
  const overdue = isOverdue(todo.due_date, todo.completed_at);
  const done = !!todo.completed_at;
  const status: 'up' | 'down' | 'unknown' = done ? 'up' : overdue ? 'down' : 'unknown';
  return (
    <article
      className="homelab-service-card"
      data-status={status}
      data-disabled="false"
      style={{ textAlign: 'left', cursor: 'default' }}
    >
      <div className="homelab-service-card-header">
        <input
          type="checkbox"
          checked={done}
          onChange={(e) => {
            e.preventDefault();
            if (e.target.checked) onComplete();
            else onToggle();
          }}
          aria-label={done ? 'Mark as not done' : 'Mark as done'}
          style={{ cursor: 'pointer' }}
        />
        <span
          className="homelab-service-card-name"
          style={{ textDecoration: done ? 'line-through' : 'none', opacity: done ? 0.6 : 1 }}
        >
          {todo.title}
        </span>
        <span
          className="homelab-service-pill"
          data-status="unknown"
          style={{ fontSize: 10, padding: '2px 6px', marginLeft: 'auto' }}
          title={`Priority: ${todo.priority}`}
        >
          {todo.priority}
        </span>
      </div>
      {todo.description ? (
        <span className="homelab-service-card-url" style={{ whiteSpace: 'pre-wrap' }}>
          {todo.description}
        </span>
      ) : null}
      <div className="homelab-service-card-footer">
        {todo.due_date ? (
          <span
            className="homelab-service-latency"
            style={overdue ? { color: 'var(--red, #ef4444)', fontWeight: 600 } : undefined}
          >
            {relativeTime(todo.due_date)}
          </span>
        ) : (
          <span className="homelab-service-latency">no due date</span>
        )}
      </div>
      {todo.tags && todo.tags.length > 0 ? (
        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
          {todo.tags.map((t) => (
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
      <div
        className="homelab-service-card-actions"
        onClick={(e) => e.stopPropagation()}
      >
        {!done ? (
          <button
            type="button"
            className="homelab-service-action-btn"
            onClick={onComplete}
            title="Mark complete"
          >
            ✓
          </button>
        ) : null}
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={onEdit}
          title="Edit"
        >
          Edit
        </button>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={onDelete}
          title="Delete"
        >
          ×
        </button>
      </div>
    </article>
  );
}
