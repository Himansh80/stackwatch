import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { login, setToken } from '../lib/api';
import Button from '../components/shared/Button';
import Input from '../components/shared/Input';
import BrandLogo from '../components/shared/BrandLogo';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const nav = useNavigate();

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const resp = await login(email.trim(), password);
      setToken(resp.token);
      nav('/dashboard');
    } catch (err: any) {
      setError(err?.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-shell">
      <Link className="auth-brand" to="/">
        <BrandLogo variant="mark" size={36} />
        <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
      </Link>
      <div className="auth-card">
        <header>
          <span className="auth-eyebrow">Welcome back</span>
          <h1>Sign in</h1>
          <p>Use your StackWatch workspace email to connect to the control plane.</p>
        </header>
        <form onSubmit={onSubmit}>
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
            autoFocus
            fullWidth
          />
          <Input
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Your password"
            required
            minLength={8}
            fullWidth
          />
          {error && <div className="auth-error">{error}</div>}
          <Button
            type="submit"
            variant="primary"
            fullWidth
            loading={loading}
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </Button>
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