import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api, setToken } from '../lib/api';
import PasswordInput from '../components/PasswordInput';
import { friendlyPasswordMessage } from '../lib/password';
import {
  motion,
  cardEntrance,
  staggerFormRows,
  formRow,
  buttonSpring,
  useReducedMotion,
} from '../lib/motion';

interface SignupError {
  message: string;
  code?: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function Signup() {
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [tenantName, setTenantName] = useState('');
  const [error, setError] = useState<SignupError | null>(null);
  const [loading, setLoading] = useState(false);

  // Reduced-motion: snap into the show state instead of running the entrance.
  const reduce = useReducedMotion();

  function clientValidate(): string | null {
    if (!tenantName.trim()) return 'Please enter a workspace name.';
    if (!fullName.trim()) return 'Please enter your name.';
    const trimmed = email.trim();
    if (!trimmed) return 'Please enter your work email.';
    if (!EMAIL_RE.test(trimmed)) return 'Please enter a valid email address.';
    if (password.length < 10) return 'Your password needs at least 10 characters.';
    return null;
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    const clientError = clientValidate();
    if (clientError) {
      setError({ message: clientError });
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const body = await api<{ token: string }>(
        'POST',
        '/api/v1/auth/signup',
        {
          email: email.trim(),
          password,
          full_name: fullName.trim(),
          tenant_name: tenantName.trim(),
        },
        false,
      );
      if (!body.token) {
        setError({ message: 'Signup succeeded but no token was returned.' });
        return;
      }
      setToken(body.token);
      nav('/dashboard');
    } catch (cause: any) {
      if (cause instanceof ApiError) {
        const code = (cause as ApiError & { code?: string }).code;
        setError({
          message: code?.startsWith('password_')
            ? friendlyPasswordMessage(code, cause.friendlyMessage)
            : cause.friendlyMessage,
          code,
        });
      } else {
        setError({ message: cause?.message || 'Unable to reach the signup endpoint.' });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <span className="auth-brand-mark">S</span>
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <motion.div
        className="auth-card"
        initial={reduce ? false : 'hidden'}
        animate="show"
        variants={cardEntrance}
      >
        <motion.header variants={formRow}>
          <span className="auth-eyebrow">Create your workspace</span>
          <h1>Start free</h1>
          <p>One workspace, one admin user, no credit card. You can run a hosted free tier or download the binary and self-host.</p>
        </motion.header>
        <motion.form
          onSubmit={onSubmit}
          initial={reduce ? false : 'hidden'}
          animate="show"
          variants={staggerFormRows}
        >
          <motion.label variants={formRow}>
            <span>Workspace name</span>
            <input
              type="text"
              value={tenantName}
              onChange={(e) => setTenantName(e.target.value)}
              placeholder="Your team or project name"
              required
              autoFocus
            />
            <small>Shared by everyone in your team. You can rename later.</small>
          </motion.label>
          <motion.label variants={formRow}>
            <span>Your name</span>
            <input
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="Your full name"
              required
            />
          </motion.label>
          <motion.label variants={formRow}>
            <span>Work email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
            />
          </motion.label>
          <motion.label className="auth-pwd-label" variants={formRow}>
            <span>Password</span>
            <PasswordInput
              value={password}
              onChange={setPassword}
              errorCode={error?.code}
              required
            />
            <small>Minimum 10 characters. Use a passphrase you don&apos;t reuse elsewhere.</small>
          </motion.label>
          {error && (
            <motion.div
              className="auth-error"
              variants={formRow}
              initial={reduce ? false : { opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
              role="alert"
            >
              {error.message}
            </motion.div>
          )}
          <motion.button
            type="submit"
            className="auth-button-primary"
            disabled={loading}
            whileHover={loading ? undefined : buttonSpring.whileHover}
            whileTap={loading ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
          >
            {loading ? 'Creating workspace...' : 'Create free account'}
          </motion.button>
        </motion.form>
        <motion.p
          className="auth-switch"
          variants={formRow}
          initial={reduce ? false : 'hidden'}
          animate="show"
        >
          Already have an account? <Link to="/login">Sign in</Link>
        </motion.p>
      </motion.div>
      <p className="auth-foot">
        By creating an account you agree to run StackWatch on systems you own or are authorized to monitor.
      </p>
    </div>
  );
}
