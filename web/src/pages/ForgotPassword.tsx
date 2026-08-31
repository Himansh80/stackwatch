import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api } from '../lib/api';
import Button from '../components/shared/Button';
import Input from '../components/shared/Input';
import BrandLogo from '../components/shared/BrandLogo';
import {
  motion,
  AnimatePresence,
  cardEntrance,
  staggerFormRows,
  formRow,
  EASE_OUT,
  useReducedMotion,
} from '../lib/motion';

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

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

  const reduce = useReducedMotion();

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

  const resetLink = response?.reset_url
    ? `${window.location.origin}${response.reset_url}`
    : null;

  const showForm = !response;

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <BrandLogo variant="mark" size={36} />
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <motion.div
        className="auth-card"
        initial={reduce ? false : "hidden"}
        animate="show"
        variants={cardEntrance}
      >
        <motion.header variants={formRow}>
          <span className="auth-eyebrow">Account recovery</span>
          <h1>{showForm ? 'Reset your password' : 'Check your inbox'}</h1>
          {showForm ? (
            <p>Enter the email tied to your workspace. We'll start the reset.</p>
          ) : (
            <p>{response?.message || 'If that email exists, a reset link has been generated.'}</p>
          )}
        </motion.header>
        <AnimatePresence mode="wait" initial={false}>
          {showForm ? (
            <motion.form
              key="forgot-form"
              onSubmit={onSubmit}
              noValidate
              initial={reduce ? false : "hidden"}
              animate="show"
              exit={reduce ? { opacity: 0 } : { opacity: 0, y: -6, transition: { duration: 0.16 } }}
              variants={staggerFormRows}
            >
              <motion.div variants={formRow}>
                <Input
                  label="Email"
                  type="email"
                  value={email}
                  onChange={(e) => { setEmail(e.target.value); setError(null); }}
                  placeholder="you@example.com"
                  required
                  autoFocus
                  autoComplete="email"
                  fullWidth
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
                  loading={loading}
                >
                {loading ? 'Working…' : 'Reset password'}
              </Button>
              </motion.div>
              <motion.p className="auth-switch" variants={formRow}>
                <Link to="/login">← Back to sign in</Link>
              </motion.p>
            </motion.form>
          ) : (
            <motion.div
              key="forgot-result"
              className="auth-success"
              initial={reduce ? false : { opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={reduce ? { opacity: 0 } : { opacity: 0, y: -8, transition: { duration: 0.16 } }}
              transition={{ duration: 0.32, ease: EASE_OUT }}
            >
              {resetLink ? (
                <>
                  <p className="auth-success-note">
                    Click the button below to open the reset page and set a new password. The link expires in 1 hour.
                  </p>
                  <div className="auth-success-actions">
                    <Button
                      variant="primary"
                      onClick={() => goToReset(resetLink)}
                    >
                      Open reset page
                    </Button>
                    <Button
                      variant="ghost"
                      onClick={() => copyLink(resetLink)}
                    >
                      Copy link
                    </Button>
                  </div>
                  <pre className="auth-success-pre">{resetLink}</pre>
                  <details className="auth-success-details">
                    <summary>Use a different email?</summary>
                    <form onSubmit={(e) => { e.preventDefault(); setResponse(null); setEmail(''); }} className="auth-success-resend">
                      <Button type="submit" variant="ghost">Send a fresh reset link</Button>
                    </form>
                  </details>
                </>
              ) : (
                <>
                  <p className="auth-success-note">
                    Check your email for a link to set a new password. The link expires in 1 hour.
                  </p>
                  <details className="auth-success-details">
                    <summary>Didn't get the email?</summary>
                    <form onSubmit={(e) => { e.preventDefault(); setResponse(null); setEmail(''); }} className="auth-success-resend">
                      <Button type="submit" variant="ghost">Send a fresh reset link</Button>
                    </form>
                  </details>
                </>
              )}
              <motion.p
                className="auth-switch"
                initial={reduce ? false : { opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.32, delay: 0.1, ease: EASE_OUT }}
              >
                <Link to="/login">← Back to sign in</Link>
              </motion.p>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
      <p className="auth-foot">
        Forgot which email you used? Ask your workspace admin to look it up from the People page.
      </p>
    </div>
  );
}