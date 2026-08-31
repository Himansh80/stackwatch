import { FormEvent, useState } from 'react';
import { ApiError, api } from '../../lib/api';
import Button from '../shared/Button';
import Input from '../shared/Input';

export interface LogRetention {
  id: string;
  service: string;
  hot_days: number;
  cold_days: number;
  enabled: boolean;
  created_at: string;
}

interface LogRetentionTabProps {
  policies: LogRetention[];
  onChanged: () => void | Promise<void>;
}

/**
 * Tier 20 Phase G: refactored raw inputs + sw-button class to
 * shared Input + Button primitives.
 */
export default function LogRetentionTab({ policies, onChanged }: LogRetentionTabProps) {
  const [form, setForm] = useState({ service: '', hot_days: 7, cold_days: 90 });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setErr('');
    if (!form.service.trim()) {
      setErr('Service is required.');
      return;
    }
    setBusy(true);
    try {
      await api('POST', '/api/v1/logs/retention', {
        service: form.service.trim(),
        hot_days: form.hot_days,
        cold_days: form.cold_days,
      });
      setForm({ service: '', hot_days: 7, cold_days: 90 });
      await onChanged();
    } catch (cause) {
      setErr(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <section className="dash-section">
        <span className="dash-eyebrow">New policy</span>
        <h2 className="dash-section-title">Set a retention window</h2>
        <form className="log-retention-form" onSubmit={submit}>
          <Input
            label="Service"
            type="text"
            value={form.service}
            onChange={(e) => setForm({ ...form, service: e.target.value })}
            placeholder="api"
            maxLength={128}
            required
          />
          <Input
            label="Hot days"
            type="number"
            min={1}
            value={form.hot_days}
            onChange={(e) => setForm({ ...form, hot_days: Number(e.target.value) })}
          />
          <Input
            label="Cold days"
            type="number"
            min={1}
            value={form.cold_days}
            onChange={(e) => setForm({ ...form, cold_days: Number(e.target.value) })}
          />
          {err ? <p className="apm-form-error">{err}</p> : null}
          <div className="apm-form-actions">
            <Button type="submit" variant="primary" size="sm" disabled={busy} loading={busy}>
              {busy ? 'Saving…' : 'Save policy'}
            </Button>
          </div>
        </form>
      </section>
      <section className="dash-section">
        <span className="dash-eyebrow">Configured</span>
        <h2 className="dash-section-title">Retention policies</h2>
        {policies.length === 0 ? (
          <p className="dash-section-lede">No retention policies configured.</p>
        ) : (
          <table className="dash-table">
            <thead>
              <tr><th>Service</th><th>Hot</th><th>Cold</th><th>Enabled</th></tr>
            </thead>
            <tbody>
              {policies.map((r) => (
                <tr key={r.id}>
                  <td><strong>{r.service}</strong></td>
                  <td>{r.hot_days}d</td>
                  <td>{r.cold_days}d</td>
                  <td>{r.enabled ? 'yes' : 'no'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
