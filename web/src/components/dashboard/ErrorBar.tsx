/**
 * ErrorBar — the inline error notice shown above the welcome header
 * when Dashboard data fails to load.
 *
 * Two variants:
 *  - **Auth error** (matches `/unauthor|sign.in|token|401/i`): softer
 *    amber bar with a "Sign in" button. The button clears the token
 *    and routes to /login.
 *  - **Generic error**: louder red bar with a Retry button that
 *    re-runs `loadDashboard()`.
 *
 * Pattern-detection is intentionally loose. False positives are fine
 * (worst case: an amber bar instead of red). False negatives would
 * strand the user on a "session expired" message with no recovery.
 */
import { useNavigate } from 'react-router-dom';
import { clearToken } from '../../lib/api';
import Button from '../shared/Button';

interface ErrorBarProps {
  error: string;
  onRetry: () => void;
}

export default function ErrorBar({ error, onRetry }: ErrorBarProps) {
  const nav = useNavigate();
  const isAuthError = /unauthor|sign.in|token|expired|401/i.test(error);
  if (isAuthError) {
    return (
      <div className="dash-error dash-error-info">
        <span className="dash-error-icon" aria-hidden="true">!</span>
        <strong>Sign in again</strong>
        <span>Your session ended. Sign back in to load live infrastructure data.</span>
        <Button
          variant="primary"
          size="sm"
          onClick={() => {
            clearToken();
            nav('/login');
          }}
        >
          Sign in
        </Button>
      </div>
    );
  }
  return (
    <div className="dash-error">
      <span className="dash-error-icon" aria-hidden="true">!</span>
      <strong>Live data unavailable</strong>
      <span>{error}</span>
      <Button variant="primary" size="sm" onClick={onRetry}>
        Retry
      </Button>
    </div>
  );
}
