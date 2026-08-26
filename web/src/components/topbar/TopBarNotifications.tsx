/**
 * TopBarNotifications.tsx — bell icon with badge + dropdown.
 *
 * Currently displays a notification channel list (email, webhook, slack, etc.)
 * using the /api/v1/notifications/channels endpoint. Falls back to "No
 * channels configured" if empty.
 *
 * TODO (follow-on phase): wire to real user notifications once that
 * endpoint exists.
 */
import { useEffect, useRef, useState } from 'react';
import { BellIcon } from '../icons';
import { api } from '../../lib/api';

interface Channel {
  id: string;
  name: string;
  kind: string;
  enabled: boolean;
}

export default function TopBarNotifications() {
  const [open, setOpen] = useState(false);
  const [channels, setChannels] = useState<Channel[]>([]);
  const ref = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [open]);

  // Load channels when opened
  useEffect(() => {
    if (!open || channels.length > 0) return;
    let cancelled = false;
    api<{ channels: Channel[] }>('GET', '/api/v1/notifications/channels')
      .then((data) => { if (!cancelled) setChannels(data.channels || []); })
      .catch(() => { /* silent */ });
    return () => { cancelled = true; };
  }, [open, channels.length]);

  const activeCount = channels.filter((c) => c.enabled).length;

  return (
    <div className="tb-notif" ref={ref}>
      <button
        type="button"
        className="tb-notif-btn"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifications"
        aria-expanded={open}
      >
        <BellIcon size={14} />
        {activeCount > 0 && (
          <span className="tb-notif-badge" aria-label={`${activeCount} active`}>
            {activeCount}
          </span>
        )}
      </button>
      {open && (
        <div className="tb-notif-popover">
          <div className="tb-notif-head">Notification Channels</div>
          {channels.length === 0 ? (
            <div className="tb-notif-empty">No channels configured.</div>
          ) : (
            <ul className="tb-notif-list">
              {channels.map((c) => (
                <li key={c.id} className="tb-notif-item">
                  <span className="tb-notif-kind">{c.kind}</span>
                  <span className="tb-notif-name">{c.name}</span>
                  <span className={'tb-notif-state' + (c.enabled ? ' on' : ' off')}>
                    {c.enabled ? 'On' : 'Off'}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}