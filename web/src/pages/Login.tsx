import { FormEvent, useEffect, useState } from 'react';
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
  const [retryUntil, setRetryUntil] = useState<number | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(0);
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
    setRetryUntil(null);
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
        // Start a live countdown if the server told us when to retry.
        if (err.code === 'rate_limited' && err.retryAfterSeconds && err.retryAfterSeconds > 0) {
          setRetryUntil(Date.now() + err.retryAfterSeconds * 1000);
        }
      } else {
        setError(err?.message || 'Login failed.');
        setErrorCode('unknown');
      }
    } finally {
      setLoading(false);
    }
  }

  // Live countdown for the rate-limit error. Ticks once a second,
  // recomputes the message every tick, and clears the error when the
  // window expires so the user can try again without a stale timer.
  useEffect(() => {
    if (retryUntil == null) return;
    function tick() {
      const remaining = Math.max(0, Math.ceil((retryUntil! - Date.now()) / 1000));
      setSecondsLeft(remaining);
      if (remaining <= 0) {
        setError(null);
        setErrorCode(null);
        setRetryUntil(null);
      }
    }
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [retryUntil]);

  // Forgot password is only useful when we KNOW the email is
  // registered but the password was wrong. For email_not_found the
  // right next step is "sign up" (link is in the bottom of the card),
  // and for client-validation errors the user just needs to fix the
  // form. The link is otherwise dead noise.
  const showForgotLink = errorCode === 'bad_password';

  // Live rate-limit message overrides the static one once the
  // countdown is running.
  const errorMessage =
    errorCode === 'rate_limited' && secondsLeft > 0
      ? `Too many attempts. Please wait ${secondsLeft} second${secondsLeft === 1 ? '' : 's'} and try again.`
      : error;

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
              onChange={(e) => { setEmail(e.target.value); setError(null); setErrorCode(null); setRetryUntil(null); }}
              placeholder="you@example.com"
              required
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
              onChange={(e) => { setPassword(e.target.value); setError(null); setErrorCode(null); setRetryUntil(null); }}
              placeholder="Your password"
              required
              autoComplete="current-password"
              aria-invalid={errorCode === 'client_validation' && password.length > 0 && password.length < 8 ? 'true' : undefined}
            />
          </label>
          {error && <div className="auth-error"><span>{errorMessage}</span></div>}
          <button type="submit" className="auth-button-primary" disabled={loading || secondsLeft > 0}>
            {loading ? 'Signing in...' : secondsLeft > 0 ? `Try again in ${secondsLeft}s` : 'Sign in'}
          </button>
          {showForgotLink && (
            <p className="auth-forgot">
              <Link to="/forgot-password">Forgot password?</Link>
            </p>
          )}
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
