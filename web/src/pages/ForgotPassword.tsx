import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api } from '../lib/api';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// /auth/forgot returns the reset link in the response body when SMTP
// is not configured. When SMTP IS configured the backend strips those
// fields and just sends the email. Either way the frontend should not
// assume the email went out — it should display whatever the backend
// gave it.
interface ForgotResponse {
  ok: boolean;
  message?: string;
  reset_token?: string;
  reset_url?: string;
  expires_at?: string;
  email_sent?: boolean;
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

  function copyLink(url: string) {
    void navigator.clipboard.writeText(url);
  }

  function goToReset(url: string) {
    const u = new URL(url, window.location.origin);
    nav(`${u.pathname}${u.search}`);
  }

  // Whether the backend surfaced a clickable reset link in this
  // response. We treat it as the canonical source of truth regardless
  // of whether SMTP was supposed to send the email — if a link is in
  // the body, we show it.
  const resetLink = response?.reset_url
    ? `${window.location.origin}${response.reset_url}`
    : null;

  // Whether to render the form (asking for email) vs the result panel.
  const showForm = !response;

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <span className="auth-brand-mark">S</span>
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Account recovery</span>
          <h1>{showForm ? 'Reset your password' : 'Check your inbox'}</h1>
          {showForm ? (
            <p>Enter the email tied to your workspace. We&apos;ll start the reset.</p>
          ) : (
            <p>{response?.message || 'If that email exists, a reset link has been generated.'}</p>
          )}
        </header>
        {showForm ? (
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
              {loading ? 'Working…' : 'Reset password'}
            </button>
            <p className="auth-switch">
              <Link to="/login">← Back to sign in</Link>
            </p>
          </form>
        ) : (
          <div className="auth-success">
            {resetLink ? (
              <>
                <p className="auth-success-note">
                  Click the button below to open the reset page and set a new password. The link expires in 1 hour.
                </p>
                <div className="auth-success-actions">
                  <button type="button" className="auth-button-primary" onClick={() => goToReset(resetLink)}>
                    Open reset page
                  </button>
                  <button type="button" className="sw-button sw-button-quiet" onClick={() => copyLink(resetLink)}>
                    Copy link
                  </button>
                </div>
                <pre className="auth-success-pre">{resetLink}</pre>
                <details className="auth-success-details">
                  <summary>Use a different email?</summary>
                  <form onSubmit={(e) => { e.preventDefault(); setResponse(null); setEmail(''); }} className="auth-success-resend">
                    <button type="submit" className="auth-button-ghost">Send a fresh reset link</button>
                  </form>
                </details>
              </>
            ) : (
              <>
                <p className="auth-success-note">
                  Check your email for a link to set a new password. The link expires in 1 hour.
                </p>
                <details className="auth-success-details">
                  <summary>Didn&apos;t get the email?</summary>
                  <form onSubmit={(e) => { e.preventDefault(); setResponse(null); setEmail(''); }} className="auth-success-resend">
                    <button type="submit" className="auth-button-ghost">Send a fresh reset link</button>
                  </form>
                </details>
              </>
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
