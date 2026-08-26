/**
 * SidebarFooter.tsx — bottom of sidebar.
 * Shows: status pill (operational / degraded / down) + version + sign-out button.
 *
 * Data fetched on mount. If /health returns ok, show "Operational";
 * otherwise show the connection status string. Falls back to "Offline".
 */
import { useEffect, useState } from 'react';
import { LogOutIcon } from '../icons';
import { useLogout } from '../../lib/useLogout';

interface SidebarFooterProps {
  /** Used by mobile drawer to close after sign-out */
  onItemClick?: () => void;
}

interface HealthResponse {
  ok?: boolean;
  status?: string;
  db_ok?: boolean;
}

export default function SidebarFooter({ onItemClick }: SidebarFooterProps) {
  const logout = useLogout();
  const [status, setStatus] = useState<'operational' | 'degraded' | 'offline'>('offline');

  useEffect(() => {
    let cancelled = false;
    fetch('/api/v1/health')
      .then((r) => (r.ok ? r.json() : null))
      .then((data: HealthResponse | null) => {
        if (cancelled) return;
        if (data?.ok) setStatus('operational');
        else setStatus('degraded');
      })
      .catch(() => {
        if (!cancelled) setStatus('offline');
      });
    return () => { cancelled = true; };
  }, []);

  const statusDot = {
    operational: 'sb-status-dot sb-status-dot-ok',
    degraded: 'sb-status-dot sb-status-dot-warn',
    offline: 'sb-status-dot sb-status-dot-down',
  }[status];

  const statusLabel = {
    operational: 'Operational',
    degraded: 'Degraded',
    offline: 'Offline',
  }[status];

  return (
    <div className="sb-footer">
      <div className="sb-status">
        <span className={statusDot} aria-hidden="true" />
        <span className="sb-status-label">{statusLabel}</span>
      </div>
      <button
        type="button"
        className="sb-signout"
        onClick={() => { onItemClick?.(); logout(); }}
      >
        <LogOutIcon size={14} />
        <span>Sign out</span>
      </button>
    </div>
  );
}