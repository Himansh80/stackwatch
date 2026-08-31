import { FormEvent, useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ApiError, api, setToken } from '../lib/api';
import PasswordInput from '../components/PasswordInput';
import PasswordField from '../components/PasswordField';
import Button from '../components/shared/Button';
import BrandLogo from '../components/shared/BrandLogo';
import { friendlyPasswordMessage } from '../lib/password';
import {
  motion,
  cardEntrance,
  staggerFormRows,
  formRow,
  EASE_OUT,
  useReducedMotion,
} from '../lib/motion';

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

  useEffect(() => {
    if (!token)
      setError(
        'This reset link is missing its token. Use the link from your email or from the forgot-password page.',
      );
  }, [token]);

  const reduce = useReducedMotion();

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
        setToken(body.token);
        nav('/dashboard');
      } else {
        setError('Password was reset. Please sign in with your new password.');
      }
    } catch (cause: any) {
      if (cause instanceof ApiError) {
        const code = (cause as ApiError & { code?: string }).code;
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
        <BrandLogo variant="mark" size={36} />
        <span>
          <strong>StackWatch</strong>
          <small>Self-hosted infrastructure platform</small>
        </span>
      </Link>
      <motion.div
        className="auth-card"
        initial={reduce ? false : "hidden"}
        animate="show"
        variants={cardEntrance}
      >
        <motion.header variants={formRow}>
          <span className="auth-eyebrow">Account recovery</span>
          <h1>Set a new password</h1>
          <p>Choose a new password for your workspace account. The link expires in 1 hour.</p>
        </motion.header>
        <motion.form
          onSubmit={onSubmit}
          noValidate
          initial={reduce ? false : "hidden"}
          animate="show"
          variants={staggerFormRows}
        >
          <motion.div className="auth-pwd-label" variants={formRow}>
            <span className="auth-pwd-label-text">New password</span>
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
            <small className="auth-pwd-help">Minimum 10 characters. Use a passphrase you don't reuse elsewhere.</small>
          </motion.div>
          <motion.div variants={formRow}>
            <PasswordField
              value={password2}
              onChange={(v) => { setPassword2(v); setError(null); }}
              placeholder="Type your new password again"
              autoComplete="new-password"
              required
              minLength={10}
            />
          </motion.div>
          {error && (
            <motion.div
              className="auth-error"
              variants={formRow}
              initial={reduce ? false : { opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.18, ease: EASE_OUT }}
              role="alert"
            >
              {error}
            </motion.div>
          )}
          <motion.div variants={formRow}>
            <Button
              type="submit"
              variant="primary"
              fullWidth
              loading={loading || !token}
              disabled={loading || !token}
            >
              {loading ? 'Saving…' : 'Set new password'}
            </Button>
          </motion.div>
          <motion.p
            className="auth-switch"
            variants={formRow}
            initial={reduce ? false : 'hidden'}
            animate="show"
          >
            <Link to="/login">← Back to sign in</Link>
          </motion.p>
        </motion.form>
      </motion.div>
      <p className="auth-foot">
        Reset tokens are single-use and expire in 1 hour. If yours has expired, request a new one.
      </p>
    </div>
  );
}