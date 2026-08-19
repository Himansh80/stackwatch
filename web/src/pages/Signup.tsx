import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { setToken } from '../lib/api';

interface SignupError {
  field?: string;
  message: string;
}

export default function Signup() {
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [tenantName, setTenantName] = useState('');
  const [error, setError] = useState<SignupError | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch('/api/v1/auth/signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: email.trim(),
          password,
          full_name: fullName.trim(),
          tenant_name: tenantName.trim(),
        }),
      });
      const text = await res.text();
      if (!res.ok) {
        let message = text;
        try {
          const parsed = JSON.parse(text);
          message = parsed.error || parsed.message || text;
        } catch {
          // fall back to raw text
        }
        setError({ message: message || `Signup failed (${res.status})` });
        return;
      }
      const body = JSON.parse(text);
      if (!body.token) {
        setError({ message: 'Signup succeeded but no token was returned.' });
        return;
      }
      setToken(body.token);
      nav('/dashboard');
    } catch (cause: any) {
      setError({ message: cause?.message || 'Unable to reach the signup endpoint.' });
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
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Create your workspace</span>
          <h1>Start free</h1>
          <p>One workspace, one admin user, no credit card. You can run a hosted free tier or download the binary and self-host.</p>
        </header>
        <form onSubmit={onSubmit}>
          <label>
            <span>Workspace name</span>
            <input
              type="text"
              value={tenantName}
              onChange={(e) => setTenantName(e.target.value)}
              placeholder="Bareilly Homelab"
              required
              autoFocus
            />
            <small>Shared by everyone in your team. You can rename later.</small>
          </label>
          <label>
            <span>Your name</span>
            <input
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="Himan Shukla"
              required
            />
          </label>
          <label>
            <span>Work email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
            />
          </label>
          <label>
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="At least 8 characters"
              required
              minLength={8}
            />
            <small>Minimum 8 characters. Use a passphrase you don&apos;t reuse elsewhere.</small>
          </label>
          {error && <div className="auth-error">{error.message}</div>}
          <button type="submit" className="auth-button-primary" disabled={loading}>
            {loading ? 'Creating workspace...' : 'Create free account'}
          </button>
        </form>
        <p className="auth-switch">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </div>
      <p className="auth-foot">
        By creating an account you agree to run StackWatch on systems you own or are authorized to monitor.      </p>
    </div>
  );
}
