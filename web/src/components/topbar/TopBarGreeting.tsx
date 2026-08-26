/**
 * TopBarGreeting.tsx — left-most topbar element (replaces logo).
 * Shows a friendly greeting + LIVE indicator with pulsing animation.
 *
 * Behaviour:
 *   - Fetches /api/v1/auth/me on mount to get user's name
 *   - Falls back to "Operator" if name unavailable
 *   - LIVE dot pulses (CSS animation) to indicate the session is active
 *   - Hidden on mobile (the sidebar drawer is the mobile nav entry point)
 */
import { useEffect, useState } from 'react';
import { me } from '../../lib/api';

interface MeResponse {
  user?: {
    full_name?: string;
    email?: string;
  };
  tenant?: { name?: string };
}

export default function TopBarGreeting() {
  const [name, setName] = useState<string>('Operator');
  // Tick every minute to keep the greeting in sync (placeholder for future "Good morning/afternoon/evening")
  const [, setNow] = useState<Date>(new Date());

  useEffect(() => {
    let cancelled = false;
    me()
      .then((data: MeResponse) => {
        if (cancelled) return;
        const fullName = data?.user?.full_name || data?.user?.email || 'Operator';
        const firstName = fullName.split(' ')[0] || 'Operator';
        setName(firstName);
      })
      .catch(() => { /* silent */ });
    return () => { cancelled = true; };
  }, []);

  // Refresh greeting minute boundary every minute
  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div className="tb-greeting">
      <span className="tb-greeting-text">
        <span className="tb-greeting-eyebrow">Hello,</span>
        <strong className="tb-greeting-name">{name}.</strong>
      </span>
      <span className="tb-greeting-live" title="Session is live">
        <span className="tb-greeting-live-dot" aria-hidden="true" />
        <span className="tb-greeting-live-text">Live</span>
      </span>
    </div>
  );
}