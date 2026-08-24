/**
 * WelcomeHeader — eyebrow + heading + Live sync meta card.
 *
 * Shows the user's first name (if loaded) or a generic welcome.
 * The "Live sync" card pulses a green dot while the dashboard is
 * actively fetching data; "Updated X ago" shows how recent.
 */
interface WelcomeHeaderProps {
  userName: string;
  lastUpdated: Date | null;
  formatRelative: (date: Date, now: Date) => string;
  now: Date;
}

export default function WelcomeHeader({
  userName,
  lastUpdated,
  formatRelative,
  now,
}: WelcomeHeaderProps) {
  const firstName = (userName || '').split(' ')[0];
  return (
    <section className="dash-welcome">
      <div>
        <span className="dash-welcome-eyebrow">Infrastructure overview</span>
        <h2>{firstName ? `Good to see you, ${firstName}.` : 'Welcome to StackWatch'}</h2>
        <p>One place to see the health of your infrastructure and move from signal to action.</p>
      </div>
      <div className="dash-welcome-meta">
        <div className="dash-welcome-meta-row">
          <span className="dash-welcome-meta-dot" />
          Live sync
        </div>
        <div className="dash-welcome-meta-time" title={lastUpdated?.toLocaleString()}>
          {lastUpdated ? `Updated ${formatRelative(lastUpdated, now)}` : 'Syncing…'}
        </div>
      </div>
    </section>
  );
}
