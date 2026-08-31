import { FormEvent, useState } from 'react';
import { ApiError, api } from '../../lib/api';
import Button from '../shared/Button';
import Input from '../shared/Input';
import Select from '../shared/Select';

export interface LogArchive {
  id: string;
  name: string;
  destination_type: string;
  destination_config: unknown;
  enabled: boolean;
  created_at: string;
}
export interface LogRehydration {
  id: string;
  archive_id: string;
  archive_name: string;
  start_time: string;
  end_time: string;
  status: string;
  progress_pct: number;
  created_at: string;
}

interface LogArchivesTabProps {
  archives: LogArchive[];
  rehydrations: LogRehydration[];
  onChanged: () => void | Promise<void>;
  setError: (msg: string) => void;
}

const DESTINATION_OPTIONS = [
  { value: 's3', label: 'S3' },
  { value: 'gcs', label: 'GCS' },
  { value: 'azure', label: 'Azure' },
  { value: 'local', label: 'Local' },
];

export default function LogArchivesTab({ archives, rehydrations, onChanged, setError }: LogArchivesTabProps) {
  const [archForm, setArchForm] = useState({ name: '', destination_type: 's3', bucket: '', region: '', path: '' });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [openFor, setOpenFor] = useState<string | null>(null);
  const [range, setRange] = useState({
    start: new Date(Date.now() - 7 * 86400_000).toISOString().slice(0, 16),
    end: new Date().toISOString().slice(0, 16),
  });

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setErr('');
    if (!archForm.name.trim()) {
      setErr('Name is required.');
      return;
    }
    setBusy(true);
    try {
      let cfg: Record<string, string> = {};
      if (archForm.destination_type === 'local') cfg = { path: archForm.path };
      else if (archForm.destination_type === 's3') cfg = { bucket: archForm.bucket, region: archForm.region };
      else cfg = { bucket: archForm.bucket };
      await api('POST', '/api/v1/logs/archives', {
        name: archForm.name.trim(),
        destination_type: archForm.destination_type,
        destination_config: cfg,
      });
      setArchForm({ name: '', destination_type: 's3', bucket: '', region: '', path: '' });
      await onChanged();
    } catch (cause) {
      setErr(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const rehydrate = async (archive: LogArchive) => {
    try {
      await api('POST', `/api/v1/logs/archives/${archive.id}/rehydrate`, {
        start_time: new Date(range.start).toISOString(),
        end_time: new Date(range.end).toISOString(),
      });
      setOpenFor(null);
      await onChanged();
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    }
  };

  return (
    <>
      <section className="dash-section">
        <span className="dash-eyebrow">New archive</span>
        <h2 className="dash-section-title">Create an archive destination</h2>
        <form className="log-archive-form" onSubmit={submit}>
          <Input
            label="Name"
            type="text"
            value={archForm.name}
            onChange={(e) => setArchForm({ ...archForm, name: e.target.value })}
            maxLength={128}
            required
          />
          <Select
            label="Destination"
            value={archForm.destination_type}
            onChange={(e) => setArchForm({ ...archForm, destination_type: e.target.value })}
            options={DESTINATION_OPTIONS}
          />
          {archForm.destination_type === 'local' ? (
            <Input
              label="Path"
              type="text"
              value={archForm.path}
              onChange={(e) => setArchForm({ ...archForm, path: e.target.value })}
              placeholder="/srv/log-archive"
            />
          ) : (
            <>
              <Input
                label="Bucket"
                type="text"
                value={archForm.bucket}
                onChange={(e) => setArchForm({ ...archForm, bucket: e.target.value })}
                placeholder="my-bucket"
              />
              {archForm.destination_type === 's3' ? (
                <Input
                  label="Region"
                  type="text"
                  value={archForm.region}
                  onChange={(e) => setArchForm({ ...archForm, region: e.target.value })}
                  placeholder="us-east-1"
                />
              ) : null}
            </>
          )}
          {err ? <p className="apm-form-error">{err}</p> : null}
          <div className="apm-form-actions">
            <Button type="submit" variant="primary" loading={busy}>
              {busy ? 'Saving…' : 'Save'}
            </Button>
          </div>
        </form>
      </section>

      <section className="dash-section">
        <span className="dash-eyebrow">Configured</span>
        <h2 className="dash-section-title">Archive destinations</h2>
        {archives.length === 0 ? (
          <p className="dash-section-lede">No archives configured.</p>
        ) : (
          <div className="log-archive-grid">
            {archives.map((a) => (
              <div key={a.id} className="log-archive-card">
                <strong>{a.name}</strong>
                <span className="log-archive-type">{a.destination_type}</span>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => setOpenFor(openFor === a.id ? null : a.id)}
                >
                  {openFor === a.id ? 'Cancel' : 'Rehydrate'}
                </Button>
                {openFor === a.id ? (
                  <div className="log-rehydrate-form">
                    <Input
                      label="Start"
                      type="datetime-local"
                      value={range.start}
                      onChange={(e) => setRange({ ...range, start: e.target.value })}
                    />
                    <Input
                      label="End"
                      type="datetime-local"
                      value={range.end}
                      onChange={(e) => setRange({ ...range, end: e.target.value })}
                    />
                    <Button
                      type="button"
                      variant="primary"
                      size="sm"
                      onClick={() => void rehydrate(a)}
                    >
                      Queue rehydration
                    </Button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="dash-section">
        <span className="dash-eyebrow">Rehydrations</span>
        <h2 className="dash-section-title">Recent rehydration jobs</h2>
        {rehydrations.length === 0 ? (
          <p className="dash-section-lede">No rehydrations queued.</p>
        ) : (
          <table className="dash-table">
            <thead>
              <tr><th>Archive</th><th>Start</th><th>End</th><th>Status</th><th>Progress</th></tr>
            </thead>
            <tbody>
              {rehydrations.map((r) => (
                <tr key={r.id}>
                  <td>{r.archive_name}</td>
                  <td>{new Date(r.start_time).toLocaleString()}</td>
                  <td>{new Date(r.end_time).toLocaleString()}</td>
                  <td>{r.status}</td>
                  <td>{r.progress_pct}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
