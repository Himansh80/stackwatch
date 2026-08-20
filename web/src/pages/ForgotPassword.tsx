import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api } from '../lib/api';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

interface ForgotResponse {
  ok: boolean;
  message?: string;
  // Dev-mode fields (returned because no SMTP is wired yet).
  dev_token?: string;
  dev_url?: string;
  expires_at?: string;
}

export default function ForgotPassword() {
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<ForgotResponse | null>(null);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    const trimmed = email.trim();
    if (!trimmed) {
      setError('Please enter your email address.');
      return;
    }
    if (!EMAIL_RE.test(trimmed)) {
      setError('Please enter a valid email address.');
      return;
    }
    setLoading(true);
    try {
      const body = await api<ForgotResponse>(
        'POST',
        '/api/v1/auth/forgot',
        { email: trimmed },
        false,
      );
      setResponse(body);
    } catch (cause: any) {
      if (cause instanceof ApiError) {
        setError(cause.friendlyMessage);
      } else {
        setError(cause?.message || 'Could not start the password reset.');
      }
    } finally {
      setLoading(false);
    }
  }

  function copyDevLink() {
    if (!response?.dev_url) return;
    const fullUrl = `${window.location.origin}${response.dev_url}`;
    void navigator.clipboard.writeText(fullUrl);
  }

  function goToReset() {
    if (!response?.dev_token) return;
    nav(`/reset-password?token=${response.dev_token}`);
  }

  // Show the dev_token + dev_url when present (no SMTP wired yet).
  const devUrl = response?.dev_url
    ? `${window.location.origin}${response.dev_url}`
    : null;

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <span className="auth-brand-mark">S</span>
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Account recovery</span>
          <h1>Forgot password?</h1>
          <p>Enter the email tied to your workspace. We&apos;ll send a reset link.</p>
        </header>
        {!response ? (
          <form onSubmit={onSubmit} noValidate>
            <label>
              <span>Email</span>
              <input
                type="email"
                value={email}
                onChange={(e) => { setEmail(e.target.value); setError(null); }}
                placeholder="you@example.com"
                required
                autoFocus
                autoComplete="email"
              />
            </label>
            {error && <div className="auth-error">{error}</div>}
            <button type="submit" className="auth-button-primary" disabled={loading}>
              {loading ? 'Sending…' : 'Send reset link'}
            </button>
            <p className="auth-switch">
              <Link to="/login">← Back to sign in</Link>
            </p>
          </form>
        ) : (
          <div className="auth-success">
            <p>
              <strong>{response.message || 'If that email exists, a reset link has been sent.'}</strong>
            </p>
            {devUrl ? (
              <>
                <p className="auth-success-note">
                  SMTP is not configured on this server, so we surfaced the reset link here for development.
                  In production this message goes to your inbox.
                </p>
                <div className="auth-success-actions">
                  <button type="button" className="auth-button-primary" onClick={goToReset}>
                    Open reset page
                  </button>
                  <button type="button" className="sw-button sw-button-quiet" onClick={copyDevLink}>
                    Copy link
                  </button>
                </div>
                <pre className="auth-success-pre">{devUrl}</pre>
              </>
            ) : (
              <p className="auth-success-note">
                Check your email for a link to set a new password. The link expires in 1 hour.
              </p>
            )}
            <p className="auth-switch">
              <Link to="/login">← Back to sign in</Link>
            </p>
          </div>
        )}
      </div>
      <p className="auth-foot">
        Forgot which email you used? Ask your workspace admin to look it up from the People page.
      </p>
    </div>
  );
}
