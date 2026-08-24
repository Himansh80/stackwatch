import avatarColor from './avatarColor';
import AvatarUpload from './AvatarUpload';

interface HeroCardProps {
  profile: {
    avatar_url: string;
    full_name: string;
    role: string;
    status: string;
    email: string;
    user_id: string;
    tenant_plan: string;
  };
  initials: string;
  tone: 'admin' | 'operator' | 'viewer' | 'neutral';
  onAvatarUploaded: (dataUrl: string) => void;
  onAvatarRemoved: () => void;
}

/**
 * HeroCard — the avatar + name + status pills at the top of the
 * Profile page. Pure presentational — the parent owns all state.
 */
export default function HeroCard({
  profile,
  initials,
  tone,
  onAvatarUploaded,
  onAvatarRemoved,
}: HeroCardProps) {
  return (
    <section className="prof-hero" aria-labelledby="prof-hero-name">
      <AvatarUpload
        currentUrl={profile.avatar_url}
        email={profile.email}
        userId={profile.user_id}
        initials={initials}
        avatarColor={avatarColor}
        onUploaded={onAvatarUploaded}
        onRemoved={onAvatarRemoved}
      />
      <div className="prof-hero-body">
        <span className="prof-hero-eyebrow">Your account</span>
        <h2 id="prof-hero-name" className="prof-hero-name">{profile.full_name || '—'}</h2>
        <div className="prof-hero-meta">
          <span className={`prof-role prof-role-${tone}`}>{profile.role || 'no role'}</span>
          <span className={`prof-status prof-status-${profile.status === 'active' ? 'active' : 'inactive'}`}>
            <span className="prof-status-dot" />
            {profile.status || 'unknown'}
          </span>
          <span className="prof-hero-plan">{profile.tenant_plan || 'free'} plan</span>
        </div>
        <div className="prof-hero-email" title={profile.email}>
          <span className="prof-hero-email-icon" aria-hidden="true">@</span>
          {profile.email || 'no email'}
        </div>
      </div>
      <div className="prof-hero-actions">
        <a className="sw-button sw-button-quiet" href="/billing">Manage plan</a>
      </div>
    </section>
  );
}
