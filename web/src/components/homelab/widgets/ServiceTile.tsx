import type { Status } from './types';

/**
 * ServiceTile — one card per pinned service. Hover reveals the
 * probe / toggle / delete actions. Click navigates to the URL.
 *
 * The tile is dumb on purpose: it just renders props and fires
 * callbacks. The parent (ServiceStatusWidget) owns the API calls.
 */

export interface PinnedService {
  id: string;
  name: string;
  url: string;
  kind: 'http' | 'https' | 'tcp' | 'icmp';
  icon?: string | null;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  health?: {
    status: Status;
    latency_ms?: number | null;
    status_code?: number | null;
    error_message?: string | null;
    checked_at: string;
  } | null;
}

interface ServiceTileProps {
  svc: PinnedService;
  probing: boolean;
  onClick: () => void;
  onProbe: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

export default function ServiceTile({
  svc, probing, onClick, onProbe, onToggle, onDelete,
}: ServiceTileProps) {
  const status: Status = svc.health?.status ?? 'unknown';
  const checkedAt = svc.health?.checked_at ?? null;
  const latency = svc.health?.latency_ms ?? null;

  return (
    <button
      type="button"
      className="homelab-service-card"
      data-status={status}
      data-disabled={svc.enabled ? 'false' : 'true'}
      onClick={onClick}
      title={`Open ${svc.url} (last checked ${checkedAt ?? 'never'})`}
    >
      <div className="homelab-service-card-header">
        {svc.icon ? <span className="homelab-service-card-icon">{svc.icon}</span> : null}
        <span className="homelab-service-card-name">{svc.name}</span>
      </div>
      <span className="homelab-service-card-url">{svc.url}</span>
      <div className="homelab-service-card-footer">
        <span className="homelab-service-pill" data-status={status}>{status}</span>
        <span className="homelab-service-latency">
          {probing ? 'probing…' : latency != null ? `${latency}ms` : '—'}
        </span>
      </div>

      <div className="homelab-service-card-actions" onClick={(e) => e.stopPropagation()}>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={onProbe}
          disabled={probing || !svc.enabled}
          title="Probe now"
        >
          Probe
        </button>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={onToggle}
          title={svc.enabled ? 'Pause probing' : 'Resume probing'}
        >
          {svc.enabled ? 'Pause' : 'Resume'}
        </button>
        <button
          type="button"
          className="homelab-service-action-btn"
          onClick={onDelete}
          title="Unpin"
        >
          ×
        </button>
      </div>
    </button>
  );
}