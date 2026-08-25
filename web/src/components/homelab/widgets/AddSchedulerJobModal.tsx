import { useState } from 'react';
import { ApiError } from '../../../lib/api';
import {
  SCHEDULER_ACTION_KINDS,
  SCHEDULER_CONTENT_TYPES,
  SCHEDULER_HEADER_BLACKLIST,
  SCHEDULER_MAX_BODY_BYTES,
  SCHEDULER_MAX_HEADERS,
  type SchedulerFormState,
} from './SchedulerWidget.types';

/**
 * AddSchedulerJobModal — Tier 10 Phase 9 (H10 — Scheduler).
 *
 * The "+ New job" modal extracted from SchedulerWidget so
 * the main widget stays under the 400-LOC cap. Owns the
 * form state (name, action_kind, url, method, headers,
 * body, content-type, schedule) and the POST /scheduler/jobs
 * call.
 *
 * Client-side mirrors the server-side guardrails so the
 * user gets immediate feedback:
 *   - URL must start with http:// or https://
 *   - body max 4KB
 *   - header name regex [A-Za-z0-9-_]{1,64}
 *   - headers exclude Host/Cookie/Authorization/etc.
 *   - schedule non-empty
 *
 * Validation runs server-side; this component reflects the
 * API error in a friendly message.
 */

interface AddSchedulerJobModalProps {
  form: SchedulerFormState;
  setForm: (next: SchedulerFormState) => void;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: () => void;
}

export default function AddSchedulerJobModal({
  form,
  setForm,
  busy,
  error,
  onClose,
  onSubmit,
}: AddSchedulerJobModalProps) {
  // Local validation: pre-flight before letting POST fly.
  const [localError, setLocalError] = useState('');

  function clientValidate(): string {
    if (!form.name.trim()) return 'Name is required.';
    if (!form.url.trim()) return 'URL is required.';
    if (!form.url.trim().startsWith('http://') && !form.url.trim().startsWith('https://')) {
      return 'URL must start with http:// or https://';
    }
    if (form.action_kind === 'http_post' && form.body.length > SCHEDULER_MAX_BODY_BYTES) {
      return `Body must be ${SCHEDULER_MAX_BODY_BYTES} bytes or fewer.`;
    }
    if (form.headerRows.length > SCHEDULER_MAX_HEADERS) {
      return `At most ${SCHEDULER_MAX_HEADERS} headers allowed.`;
    }
    const re = /^[A-Za-z0-9-_]{1,64}$/;
    for (const h of form.headerRows) {
      if (!h.name.trim()) continue;
      if (!re.test(h.name.trim())) return `Header name "${h.name}" must match [A-Za-z0-9-_]{1,64}`;
      const lower = h.name.trim().toLowerCase();
      if (SCHEDULER_HEADER_BLACKLIST.some((b) => b.toLowerCase() === lower)) {
        return `Header "${h.name}" is not allowed (forbidden: ${SCHEDULER_HEADER_BLACKLIST.join(', ')})`;
      }
    }
    if (!form.schedule.trim()) return 'Schedule is required.';
    return '';
  }

  function handleSubmit() {
    const v = clientValidate();
    if (v) {
      setLocalError(v);
      return;
    }
    setLocalError('');
    onSubmit();
  }

  void ApiError;
  // Suppress unused import warning while keeping the import
  // for future API-error surface updates.
  void setLocalError;
  void setLocalError;

  return (
    <div
      className="homelab-modal-backdrop"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="homelab-modal-panel"
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-2)',
          borderRadius: 8,
          padding: 24,
          minWidth: 420,
          maxWidth: 560,
          width: '95%',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        <h3 style={{ marginTop: 0 }}>New scheduled job</h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Name</span>
            <input
              type="text"
              className="homelab-search-input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Garage door ping"
              maxLength={100}
            />
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Action</span>
            <select
              className="homelab-search-input"
              value={form.action_kind}
              onChange={(e) => {
                const ak = e.target.value as typeof form.action_kind;
                const method = ak === 'http_post' ? 'POST' : 'GET';
                setForm({ ...form, action_kind: ak, method });
              }}
            >
              {SCHEDULER_ACTION_KINDS.map((a) => (
                <option key={a} value={a}>
                  {a}
                </option>
              ))}
            </select>
          </label>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>URL</span>
            <input
              type="url"
              className="homelab-search-input"
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
              placeholder="https://probe.example.com/heartbeat"
            />
            <small style={{ color: 'var(--muted)', fontSize: 11 }}>
              Public endpoints only — private/loopback addresses are rejected.
            </small>
          </label>
          {form.action_kind === 'http_post' && (
            <>
              <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <span style={{ fontSize: 12, fontWeight: 600 }}>Content-Type</span>
                <select
                  className="homelab-search-input"
                  value={form.contentType}
                  onChange={(e) => setForm({ ...form, contentType: e.target.value })}
                >
                  {SCHEDULER_CONTENT_TYPES.map((c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  ))}
                </select>
              </label>
              <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                <span style={{ fontSize: 12, fontWeight: 600 }}>Body</span>
                <textarea
                  className="homelab-search-input"
                  value={form.body}
                  onChange={(e) => setForm({ ...form, body: e.target.value })}
                  placeholder='{"event":"ping"}'
                  rows={4}
                  maxLength={SCHEDULER_MAX_BODY_BYTES}
                />
                <small style={{ color: 'var(--muted)', fontSize: 11 }}>
                  {form.body.length} / {SCHEDULER_MAX_BODY_BYTES} bytes
                </small>
              </label>
            </>
          )}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>
              Headers <small style={{ color: 'var(--muted)', fontWeight: 400 }}>(max {SCHEDULER_MAX_HEADERS}; Host/Cookie/Authorization blocked)</small>
            </span>
            {form.headerRows.map((h, i) => (
              <div key={i} style={{ display: 'flex', gap: 6 }}>
                <input
                  type="text"
                  className="homelab-search-input"
                  value={h.name}
                  onChange={(e) => {
                    const rows = [...form.headerRows];
                    rows[i] = { ...rows[i], name: e.target.value };
                    setForm({ ...form, headerRows: rows });
                  }}
                  placeholder="X-Signature"
                  style={{ flex: 1 }}
                />
                <input
                  type="text"
                  className="homelab-search-input"
                  value={h.value}
                  onChange={(e) => {
                    const rows = [...form.headerRows];
                    rows[i] = { ...rows[i], value: e.target.value };
                    setForm({ ...form, headerRows: rows });
                  }}
                  placeholder="value"
                  style={{ flex: 2 }}
                />
                <button
                  type="button"
                  className="empty-state-cta"
                  onClick={() => {
                    const rows = form.headerRows.filter((_, j) => j !== i);
                    setForm({ ...form, headerRows: rows });
                  }}
                >
                  ×
                </button>
              </div>
            ))}
            {form.headerRows.length < SCHEDULER_MAX_HEADERS && (
              <button
                type="button"
                className="empty-state-cta"
                onClick={() =>
                  setForm({
                    ...form,
                    headerRows: [...form.headerRows, { name: '', value: '' }],
                  })
                }
              >
                + Add header
              </button>
            )}
          </div>
          <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600 }}>Schedule (cron, 5 fields)</span>
            <input
              type="text"
              className="homelab-search-input"
              value={form.schedule}
              onChange={(e) => setForm({ ...form, schedule: e.target.value })}
              placeholder="0 3 * * 0"
            />
            <small style={{ color: 'var(--muted)', fontSize: 11 }}>
              minute hour day-of-month month day-of-week — e.g. <code>*/5 * * * *</code> every 5 min
            </small>
          </label>
          {(localError || error) ? (
            <div className="dash-error" role="alert">
              {localError || error}
            </div>
          ) : null}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button
              type="button"
              className="empty-state-cta"
              onClick={handleSubmit}
              disabled={busy}
            >
              {busy ? 'Creating…' : 'Create job'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
