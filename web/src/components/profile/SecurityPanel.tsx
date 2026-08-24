import { ReactNode } from 'react';
import { Link } from 'react-router-dom';

function SecurityRow({
  label,
  value,
  action,
  badge,
}: {
  label: string;
  value: string;
  action?: ReactNode;
  badge?: ReactNode;
}) {
  return (
    <div className="prof-security-row">
      <div className="prof-security-text">
        <span className="prof-security-label">{label}</span>
        <span className="prof-security-value">{value}</span>
      </div>
      <div className="prof-security-actions">
        {badge}
        {action}
      </div>
    </div>
  );
}

interface SecurityPanelProps {
  onSignOut: () => void;
  onToggleTokens: () => void;
  tokensOpen: boolean;
  tokenCount: number;
}

/**
 * SecurityPanel — the right card on the second Profile row.
 * Lists Password / 2FA / Active sessions / API tokens with the
 * appropriate action buttons. Tokens panel itself is owned by
 * TokensPanel (rendered separately) so this stays focused.
 */
export default function SecurityPanel({
  onSignOut,
  onToggleTokens,
  tokensOpen,
  tokenCount,
}: SecurityPanelProps) {
  return (
    <article className="dash-panel prof-panel prof-panel-security">
      <div className="dash-panel-head">
        <div>
          <span className="dash-eyebrow">Security</span>
          <h3>Sign-in & access</h3>
        </div>
      </div>
      <div className="prof-security-list">
        <SecurityRow
          label="Password"
          value="Last changed at signup"
          action={<Link className="sw-button sw-button-quiet" to="/settings">Change password</Link>}
        />
        <SecurityRow
          label="Two-factor authentication"
          value="Not enabled"
          badge={<span className="prof-badge-warn">Recommended</span>}
          action={
            <button
              type="button"
              className="sw-button sw-button-quiet"
              disabled
              title="Coming soon — 2FA rolls out in a future release"
            >
              Configure
            </button>
          }
        />
        <SecurityRow
          label="Active sessions"
          value="1 device · this browser"
          action={
            <button type="button" className="sw-button sw-button-quiet" onClick={onSignOut}>
              Sign out
            </button>
          }
        />
        <SecurityRow
          label="API tokens"
          value={tokensOpen ? `${tokenCount} token${tokenCount === 1 ? '' : 's'}` : 'Manage personal access tokens'}
          action={
            <button type="button" className="sw-button sw-button-quiet" onClick={onToggleTokens}>
              {tokensOpen ? 'Hide' : 'Manage'}
            </button>
          }
        />
      </div>
    </article>
  );
}
