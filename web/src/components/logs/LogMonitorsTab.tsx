import { FormEvent, useState } from 'react';
import { ApiError, api } from '../../lib/api';
import Button from '../shared/Button';
import Input from '../shared/Input';
import Select from '../shared/Select';

export interface LogMonitor {
  id: string;
  name: string;
  query: string;
  threshold_count: number;
  threshold_window_seconds: number;
  severity: string;
  enabled: boolean;
  created_at: string;
}

interface LogMonitorsTabProps {
  monitors: LogMonitor[];
  onChanged: () => void | Promise<void>;
  setError: (msg: string) => void;
}

const SEVERITY_OPTIONS = [
  { value: 'warn', label: 'warn' },
  { value: 'error', label: 'error' },
  { value: 'info', label: 'info' },
];

export default function LogMonitorsTab({ monitors, onChanged, setError }: LogMonitorsTabProps) {
  const [monForm, setMonForm] = useState({ name: '', query: '', threshold_count: 10, threshold_window_seconds: 300, severity: 'warn' });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setErr('');
    if (!monForm.name.trim() || !monForm.query.trim()) {
      setErr('Name and query are required.');
      return;
    }
    setBusy(true);
    try {
      await api('POST', '/api/v1/logs/monitors', {
        name: monForm.name.trim(),
        query: monForm.query.trim(),
        threshold_count: monForm.threshold_count,
        threshold_window_seconds: monForm.threshold_window_seconds,
        severity: monForm.severity,
      });
      setMonForm({ name: '', query: '', threshold_count: 10, threshold_window_seconds: 300, severity: 'warn' });
      await onChanged();
    } catch (cause) {
      setErr(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const toggle = async (m: LogMonitor) => {
    try {
      await api('PUT', `/api/v1/logs/monitors/${m.id}`, { enabled: !m.enabled });
      await onChanged();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  };

  const remove = async (m: LogMonitor) => {
    if (!window.confirm(`Delete monitor "${m.name}"?`)) return;
    try {
      await api('DELETE', `/api/v1/logs/monitors/${m.id}`);
      await onChanged();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  };

  return (
    <>
      <section className="dash-section">
        <span className="dash-eyebrow">New monitor</span>
        <h2 className="dash-section-title">Create a log monitor</h2>
        <form className="log-monitor-form" onSubmit={submit}>
          <Input
            label="Name"
            type="text"
            value={monForm.name}
            onChange={(e) => setMonForm({ ...monForm, name: e.target.value })}
            placeholder="error-spike"
            maxLength={128}
            required
          />
          <Input
            label="Query"
            type="text"
            value={monForm.query}
            onChange={(e) => setMonForm({ ...monForm, query: e.target.value })}
            placeholder="level=error service=api"
            maxLength={1024}
            required
          />
          <Input
            label="Threshold count"
            type="number"
            min={1}
            value={monForm.threshold_count}
            onChange={(e) => setMonForm({ ...monForm, threshold_count: Number(e.target.value) })}
          />
          <Input
            label="Window (seconds)"
            type="number"
            min={10}
            value={monForm.threshold_window_seconds}
            onChange={(e) => setMonForm({ ...monForm, threshold_window_seconds: Number(e.target.value) })}
          />
          <Select
            label="Severity"
            value={monForm.severity}
            onChange={(e) => setMonForm({ ...monForm, severity: e.target.value })}
            options={SEVERITY_OPTIONS}
          />
          {err ? <p className="apm-form-error">{err}</p> : null}
          <div className="apm-form-actions">
            <Button type="submit" variant="primary" loading={busy}>
              {busy ? 'Creating…' : 'Create'}
            </Button>
          </div>
        </form>
      </section>
      <section className="dash-section">
        <span className="dash-eyebrow">Configured</span>
        <h2 className="dash-section-title">Log monitors</h2>
        {monitors.length === 0 ? (
          <p className="dash-section-lede">No monitors yet. Add one to start alerting on log patterns.</p>
        ) : (
          <table className="dash-table">
            <thead>
              <tr><th>Name</th><th>Query</th><th>Threshold</th><th>Severity</th><th>Enabled</th><th></th></tr>
            </thead>
            <tbody>
              {monitors.map((m) => (
                <tr key={m.id}>
                  <td><strong>{m.name}</strong></td>
                  <td><code>{m.query}</code></td>
                  <td>{m.threshold_count} / {m.threshold_window_seconds}s</td>
                  <td>{m.severity}</td>
                  <td>
                    <Button type="button" variant="secondary" size="sm" onClick={() => void toggle(m)}>
                      {m.enabled ? 'Disable' : 'Enable'}
                    </Button>
                  </td>
                  <td>
                    <Button type="button" variant="danger" size="sm" onClick={() => void remove(m)}>
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
