import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, login, setToken } from '../lib/api';

// Lightweight RFC-5322-ish check. Same shape browser's <input type="email">
// uses, but we surface the error inline instead of relying on the
// browser's native popup. Empty string is allowed here so the user can
// clear and retype; the submit handler does the real check.
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const nav = useNavigate();

  function clientValidate(): string | null {
    const trimmed = email.trim();
    if (!trimmed) return 'Please enter your email address.';
    if (!EMAIL_RE.test(trimmed)) return 'Please enter a valid email address.';
    if (!password) return 'Please enter your password.';
    if (password.length < 8) return 'Your password needs at least 8 characters.';
    return null;
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setErrorCode(null);
    const clientError = clientValidate();
    if (clientError) {
      setError(clientError);
      setErrorCode('client_validation');
      return;
    }
    setLoading(true);
    try {
      const resp = await login(email.trim(), password);
      setToken(resp.token);
      nav('/dashboard');
    } catch (err: any) {
      if (err instanceof ApiError) {
        setError(err.friendlyMessage);
        setErrorCode(err.code);
      } else {
        setError(err?.message || 'Login failed.');
        setErrorCode('unknown');
      }
    } finally {
      setLoading(false);
    }
  }

  // When the email-not-found error is shown, surface a direct link to
  // signup so the user has a clear next step.
  const showSignupHint = errorCode === 'email_not_found';

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <span className="auth-brand-mark">S</span>
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Welcome back</span>
          <h1>Sign in</h1>
          <p>Use your StackWatch workspace email to connect to the control plane.</p>
        </header>
        <form onSubmit={onSubmit} noValidate>
          <label>
            <span>Email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => { setEmail(e.target.value); if (error) { setError(null); setErrorCode(null); } }}
              placeholder="you@example.com"
              autoFocus
              autoComplete="email"
              aria-invalid={errorCode === 'client_validation' && !EMAIL_RE.test(email.trim()) ? 'true' : undefined}
            />
          </label>
          <label>
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(e) => { setPassword(e.target.value); if (error) { setError(null); setErrorCode(null); } }}
              placeholder="Your password"
              autoComplete="current-password"
              aria-invalid={errorCode === 'client_validation' && password.length > 0 && password.length < 8 ? 'true' : undefined}
            />
          </label>
          {error && (
            <div className="auth-error">
              <span>{error}</span>
              {showSignupHint && (
                <>
                  {' '}
                  <Link to="/signup">Create a free workspace →</Link>
                </>
              )}
            </div>
          )}
          <button type="submit" className="auth-button-primary" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
        <p className="auth-switch">
          New to StackWatch? <Link to="/signup">Create a free workspace</Link>
        </p>
      </div>
      <p className="auth-foot">
        StackWatch runs on hardware you control. Self-host with a single binary, or use the hosted control plane.
      </p>
    </div>
  );
}
