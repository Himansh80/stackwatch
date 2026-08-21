import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ApiError, api, setToken } from '../lib/api';
import PasswordInput from '../components/PasswordInput';
import { friendlyPasswordMessage } from '../lib/password';

interface ResetResponse {
  ok: boolean;
  token?: string;
  user?: unknown;
  tenant?: unknown;
}

export default function ResetPassword() {
  const [params] = useSearchParams();
  const nav = useNavigate();
  const token = params.get('token') || '';
  const [password, setPassword] = useState('');
  const [password2, setPassword2] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(false);

  // If no token in URL, this page is unusable.
  useEffect(() => {
    if (!token)
      setError(
        'This reset link is missing its token. Use the link from your email or from the forgot-password page.',
      );
  }, [token]);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setErrorCode(undefined);
    if (!token) {
      setError('Reset token missing.');
      return;
    }
    if (password.length < 10) {
      setError('Your new password must be at least 10 characters.');
      return;
    }
    if (password !== password2) {
      setError('New password and confirmation do not match.');
      return;
    }
    setLoading(true);
    try {
      const body = await api<ResetResponse>(
        'POST',
        '/api/v1/auth/reset',
        { token, new_password: password },
        false,
      );
      if (body.token) {
        // Auto-login the user so they land on the dashboard instead of the sign-in page.
        setToken(body.token);
        nav('/dashboard');
      } else {
        setError('Password was reset. Please sign in with your new password.');
      }
    } catch (cause: any) {
      if (cause instanceof ApiError) {
        const code = (cause as ApiError & { code?: string }).code;
        // Backend returns 401 with code 'invalid_token' (or similar) for bad tokens.
        if (cause.status === 401) {
          setError(
            'This reset link is invalid or has expired. Request a new one from the forgot-password page.',
          );
        } else if (code?.startsWith('password_')) {
          setErrorCode(code);
          setError(friendlyPasswordMessage(code, cause.friendlyMessage));
        } else {
          setError(cause.friendlyMessage);
        }
      } else {
        setError(cause?.message || 'Could not reset your password.');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <span className="auth-brand-mark">S</span>
        <span>
          <strong>StackWatch</strong>
          <small>Self-hosted infrastructure platform</small>
        </span>
      </Link>
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Account recovery</span>
          <h1>Set a new password</h1>
          <p>Choose a new password for your workspace account. The link expires in 1 hour.</p>
        </header>
        <form onSubmit={onSubmit} noValidate>
          <label className="auth-pwd-label">
            <span>New password</span>
            <PasswordInput
              value={password}
              onChange={(v) => {
                setPassword(v);
                setError(null);
                setErrorCode(undefined);
              }}
              errorCode={errorCode}
              autoFocus
              required
            />
            <small>Minimum 10 characters. Use a passphrase you don&apos;t reuse elsewhere.</small>
          </label>
          <label>
            <span>Confirm new password</span>
            <input
              type="password"
              value={password2}
              onChange={(e) => {
                setPassword2(e.target.value);
                setError(null);
              }}
              placeholder="Type your new password again"
              autoComplete="new-password"
              required
              minLength={10}
            />
          </label>
          {error && <div className="auth-error">{error}</div>}
          <button type="submit" className="auth-button-primary" disabled={loading || !token}>
            {loading ? 'Saving…' : 'Set new password'}
          </button>
          <p className="auth-switch">
            <Link to="/login">← Back to sign in</Link>
          </p>
        </form>
      </div>
      <p className="auth-foot">
        Reset tokens are single-use and expire in 1 hour. If yours has expired, request a new one.
      </p>
    </div>
  );
}
