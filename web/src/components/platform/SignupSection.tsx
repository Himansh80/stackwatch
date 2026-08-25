import { FormEvent, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError, api, setToken } from '../../lib/api';
import PasswordInput from '../../components/PasswordInput';
import {
  motion,
  pageEnter,
  kpiEnter,
  kpiStagger,
  useReducedMotion,
} from '../../lib/motion';

// Tier 11 Phase 3 — Self-Service Signup (PL3).
//
// SignupSection renders the public, Datadog-style self-service
// signup wizard. It is the section counterpart to web/src/pages/
// Signup.tsx (the standalone /signup route) — a 3-step wizard
// suitable for embedding inside any marketing-style page (a
// /pricing CTA, a footer "Get started" button, etc.) where the
// caller wants a more guided flow than the bare page offers.
//
// 3 steps:
//   1. Form       — email + password + full_name + organization_name.
//                   Server-side validation is the source of truth
//                   (HTML5 minlength is a hint, not a gate).
//   2. Verify     — paste the verification_token returned by step 1
//                   (in dev mode; SMTP lands in PL5). On success the
//                   backend mints a JWT so step 3 is reached already
//                   logged-in.
//   3. Welcome    — success animation + "Go to dashboard" button.
//
// Motion: pageEnter on the wrapper; kpiStagger on the step dots;
// kpiEnter on each dot. No new motion variants — reuses the
// existing primitives from src/lib/motion.tsx.
//
// Data sources:
//   POST /api/v1/platform/signup        — step 1 → 2 transition
//   POST /api/v1/platform/signup/verify — step 2 → 3 transition
//   POST /api/v1/platform/signup/resend — re-mint token (step 2 only)
//
// Auth: every step runs against the PUBLIC endpoints — no JWT
// needed (a freshly-typed email has no StackWatch credentials
// yet). Step 3 stores the freshly-issued JWT in localStorage
// via setToken() so the redirect lands the user logged-in.

type Step = 'form' | 'verify' | 'welcome';

interface SignupForm {
  email: string;
  password: string;
  full_name: string;
  organization_name: string;
}

interface SignupResp {
  id: string;
  email: string;
  status: string;
  verification_token?: string;
  next_step: string;
}

interface VerifyResp {
  token: string;
  tenant_id: string;
  user_id: string;
  email: string;
  role: string;
}

const EMPTY_FORM: SignupForm = {
  email: '',
  password: '',
  full_name: '',
  organization_name: '',
};

const STEPS: { id: Step; label: string }[] = [
  { id: 'form', label: 'Your details' },
  { id: 'verify', label: 'Verify email' },
  { id: 'welcome', label: 'Welcome' },
];

export default function SignupSection() {
  const reduce = useReducedMotion();
  const navigate = useNavigate();

  const [step, setStep] = useState<Step>('form');
  const [form, setForm] = useState<SignupForm>(EMPTY_FORM);
  const [token, setToken_] = useState<string>('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [resendBusy, setResendBusy] = useState(false);
  const [resendOk, setResendOk] = useState(false);

  // Clear the resend-success banner when the user types a new
  // token (the banner would be misleading otherwise).
  useEffect(() => {
    if (!token) {
      setResendOk(false);
    }
  }, [token]);

  // Step 1 submit: POST /signup, advance to verify.
  async function submitForm(ev: FormEvent) {
    ev.preventDefault();
    setBusy(true);
    setError('');
    try {
      const resp = await api<SignupResp>('POST', '/api/v1/platform/signup', form, false);
      if (resp.verification_token) {
        // Dev mode: auto-fill the token into the verify box
        // so the operator doesn't have to copy-paste. In a
        // real SMTP setup this branch wouldn't run — the
        // token would arrive via email.
        setToken_(resp.verification_token);
      }
      setStep('verify');
    } catch (e) {
      setError(e instanceof ApiError ? e.friendlyMessage : 'Signup failed.');
    } finally {
      setBusy(false);
    }
  }

  // Step 2 submit: POST /signup/verify, advance to welcome.
  async function submitToken(ev: FormEvent) {
    ev.preventDefault();
    setBusy(true);
    setError('');
    try {
      const resp = await api<VerifyResp>('POST', '/api/v1/platform/signup/verify', { token }, false);
      setToken(resp.token); // store JWT so redirect lands logged-in
      setStep('welcome');
    } catch (e) {
      setError(e instanceof ApiError ? e.friendlyMessage : 'Verification failed.');
    } finally {
      setBusy(false);
    }
  }

  // Step 2 helper: rotate the token via /signup/resend.
  async function resendVerification() {
    setResendBusy(true);
    setResendOk(false);
    setError('');
    try {
      const resp = await api<{ verification_token?: string }>(
        'POST',
        '/api/v1/platform/signup/resend',
        { email: form.email },
        false,
      );
      if (resp.verification_token) {
        setToken_(resp.verification_token);
      }
      setResendOk(true);
    } catch (e) {
      setError(e instanceof ApiError ? e.friendlyMessage : 'Resend failed.');
    } finally {
      setResendBusy(false);
    }
  }

  function resetWizard() {
    setForm(EMPTY_FORM);
    setToken_('');
    setError('');
    setResendOk(false);
    setStep('form');
  }

  // ---- Render ----

  return (
    <motion.section
      className="signup-section"
      initial={reduce ? false : 'hidden'}
      animate="show"
      variants={pageEnter}
      aria-label="Self-service signup"
    >
      <motion.ol
        className="signup-steps"
        variants={kpiStagger}
        initial={reduce ? false : 'hidden'}
        animate="show"
        aria-label="Signup progress"
      >
        {STEPS.map((s, i) => (
          <motion.li
            key={s.id}
            variants={kpiEnter}
            className={'signup-step ' + (s.id === step ? 'signup-step-active' : '')}
            aria-current={s.id === step ? 'step' : undefined}
          >
            <span className="signup-step-num" aria-hidden="true">{i + 1}</span>
            <span className="signup-step-label">{s.label}</span>
          </motion.li>
        ))}
      </motion.ol>

      {error && (
        <div className="signup-error" role="alert">{error}</div>
      )}

      {step === 'form' && (
        <form className="signup-form" onSubmit={submitForm} noValidate>
          <label className="signup-field">
            <span>Email</span>
            <input
              type="email"
              name="email"
              required
              autoComplete="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </label>

          <label className="signup-field">
            <span>Full name</span>
            <input
              type="text"
              name="full_name"
              required
              autoComplete="name"
              maxLength={128}
              value={form.full_name}
              onChange={(e) => setForm({ ...form, full_name: e.target.value })}
            />
          </label>

          <label className="signup-field">
            <span>Organization</span>
            <input
              type="text"
              name="organization_name"
              required
              maxLength={128}
              value={form.organization_name}
              onChange={(e) => setForm({ ...form, organization_name: e.target.value })}
            />
          </label>

          <div className="signup-field">
            <span>Password</span>
            <PasswordInput
              value={form.password}
              onChange={(next) => setForm({ ...form, password: next })}
              autoFocus
            />
          </div>

          <button type="submit" className="signup-submit" disabled={busy}>
            {busy ? 'Creating account…' : 'Create account'}
          </button>
          <p className="signup-meta">
            Already have an account? <a href="/login">Sign in</a>
          </p>
        </form>
      )}

      {step === 'verify' && (
        <form className="signup-form" onSubmit={submitToken} noValidate>
          <p className="signup-copy">
            We sent a verification token to <strong>{form.email}</strong>.
            Paste it below to activate your account.
          </p>
          <label className="signup-field">
            <span>Verification token</span>
            <input
              type="text"
              name="token"
              required
              minLength={16}
              autoFocus
              autoComplete="one-time-code"
              value={token}
              onChange={(e) => setToken_(e.target.value)}
            />
          </label>
          <div className="signup-verify-actions">
            <button type="submit" className="signup-submit" disabled={busy || token.length < 16}>
              {busy ? 'Verifying…' : 'Verify and sign in'}
            </button>
            <button
              type="button"
              className="signup-resend"
              onClick={resendVerification}
              disabled={resendBusy}
            >
              {resendBusy ? 'Resending…' : 'Resend token'}
            </button>
          </div>
          {resendOk && (
            <p className="signup-meta signup-meta-ok" role="status">
              A new token has been issued. Check your inbox (or the dev-mode response).
            </p>
          )}
          <button
            type="button"
            className="signup-back"
            onClick={() => setStep('form')}
            disabled={busy}
          >
            ← Back
          </button>
        </form>
      )}

      {step === 'welcome' && (
        <motion.div
          className="signup-welcome"
          initial={reduce ? false : { opacity: 0, scale: 0.96 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.35, ease: 'easeOut' }}
        >
          <div className="signup-check" aria-hidden="true">✓</div>
          <h2 className="signup-welcome-title">You&apos;re in!</h2>
          <p className="signup-copy">
            Your tenant <strong>{form.organization_name}</strong> is ready.
            We&apos;ve signed you in as <strong>{form.email}</strong>.
          </p>
          <div className="signup-welcome-actions">
            <button
              type="button"
              className="signup-submit"
              onClick={() => navigate('/dashboard')}
            >
              Go to dashboard
            </button>
            <button
              type="button"
              className="signup-resend"
              onClick={resetWizard}
            >
              Sign up another
            </button>
          </div>
        </motion.div>
      )}
    </motion.section>
  );
}
