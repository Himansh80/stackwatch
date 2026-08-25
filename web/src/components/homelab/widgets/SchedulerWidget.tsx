import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import AddSchedulerJobModal from './AddSchedulerJobModal';
import SchedulerJobDetailModal from './SchedulerJobDetailModal';
import {
  SCHEDULER_CONTENT_TYPES,
  asSchedulerJobs,
  asSchedulerRuns,
  schedulerRelativeTime,
  schedulerStatusColor,
  type SchedulerFormState,
  type SchedulerJob,
  type SchedulerRun,
} from './SchedulerWidget.types';

/**
 * SchedulerWidget — Tier 10 Phase 9 (H10 — Task Scheduler).
 *
 * The scheduler tab content on the HomelabPage. Calls:
 *   GET    /api/v1/homelab/scheduler/jobs
 *   POST   /api/v1/homelab/scheduler/jobs
 *   PATCH  /api/v1/homelab/scheduler/jobs/:id
 *   DELETE /api/v1/homelab/scheduler/jobs/:id
 *   GET    /api/v1/homelab/scheduler/runs
 *   POST   /api/v1/homelab/scheduler/jobs/:id/run
 *
 * Lists every job with status pill + next_run + last_run
 * + schedule + last_run_status. Click job → details modal
 * with headers + body + recent runs. "+ New job" opens the
 * modal in AddSchedulerJobModal.tsx so the main widget
 * stays under the 400-LOC cap.
 *
 * SECURITY: the modal's client-side validation mirrors the
 * server-side allowlist (URL prefix, body cap, header
 * blacklist, schedule required). The server-side
 * handlers_homelab_scheduler_validation.go is the source
 * of truth — anything that slips past the client is
 * still rejected.
 *
 * Motion: pageEnter on the wrapper (existing token, no
 * new variants).
 */

const emptyForm: SchedulerFormState = {
  name: '',
  action_kind: 'http_get',
  url: '',
  method: 'GET',
  headerRows: [],
  body: '',
  contentType: SCHEDULER_CONTENT_TYPES[0],
  schedule: '0 * * * *',
};

export default function SchedulerWidget({ config: _config }: { config?: { feed_id?: string } }) {
  void _config;
  const [jobs, setJobs] = useState<SchedulerJob[]>([]);
  const [runs, setRuns] = useState<SchedulerRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<SchedulerFormState>(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState('');
  const [detailJob, setDetailJob] = useState<SchedulerJob | null>(null);
  const [busyJobId, setBusyJobId] = useState<string | null>(null);
  const [capRemaining, setCapRemaining] = useState(25);

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your scheduler.');
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const resp = await api<unknown>('GET', '/api/v1/homelab/scheduler/jobs');
      const data = asSchedulerJobs(resp);
      setJobs(data.jobs);
      setCapRemaining(data.cap_remaining);
      setError('');
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const sortedJobs = useMemo(
    () =>
      [...jobs].sort((a, b) => {
        // Enabled first, then by next_run_at asc (soonest first).
        if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
        const an = Date.parse(a.next_run_at ?? '') || Number.POSITIVE_INFINITY;
        const bn = Date.parse(b.next_run_at ?? '') || Number.POSITIVE_INFINITY;
        return an - bn;
      }),
    [jobs],
  );

  async function createJob() {
    setBusy(true);
    setFormError('');
    try {
      const headers: Record<string, string> = {};
      for (const r of form.headerRows) {
        const name = r.name.trim();
        if (!name) continue;
        headers[name] = r.value;
      }
      if (form.action_kind === 'http_post' && form.body.trim()) {
        headers['Content-Type'] = form.contentType;
      }
      const body: Record<string, unknown> = {
        name: form.name.trim(),
        action_kind: form.action_kind,
        url: form.url.trim(),
        method: form.method,
        headers,
        body: form.action_kind === 'http_post' ? form.body : undefined,
        schedule: form.schedule.trim(),
      };
      await api<unknown>('POST', '/api/v1/homelab/scheduler/jobs', body);
      setModalOpen(false);
      setForm(emptyForm);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setFormError(msg);
    } finally {
      setBusy(false);
    }
  }

  async function deleteJob(id: string) {
    if (!confirm('Delete this job and its run history?')) return;
    setBusyJobId(id);
    try {
      await api<unknown>('DELETE', `/api/v1/homelab/scheduler/jobs/${id}`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setBusyJobId(null);
    }
  }

  async function toggleEnabled(job: SchedulerJob) {
    setBusyJobId(job.id);
    try {
      await api<unknown>('PATCH', `/api/v1/homelab/scheduler/jobs/${job.id}`, {
        enabled: !job.enabled,
      });
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setBusyJobId(null);
    }
  }

  async function runNow(job: SchedulerJob) {
    setBusyJobId(job.id);
    try {
      await api<unknown>('POST', `/api/v1/homelab/scheduler/jobs/${job.id}/run`);
      await load();
      if (detailJob?.id === job.id) {
        await openDetail(job);
      }
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setBusyJobId(null);
    }
  }

  async function openDetail(job: SchedulerJob) {
    setDetailJob(job);
    try {
      const resp = await api<unknown>(
        'GET',
        `/api/v1/homelab/scheduler/runs?job_id=${encodeURIComponent(job.id)}&limit=50`,
      );
      const data = asSchedulerRuns(resp);
      setRuns(data.runs);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
      setRuns([]);
    }
  }

  return (
    <motion.div initial="hidden" animate="show" variants={pageEnter}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
          flexWrap: 'wrap',
          gap: 8,
        }}
      >
        <div>
          <h2 style={{ margin: 0 }}>Scheduler</h2>
          <small style={{ color: 'var(--muted)' }}>
            HTTP probes only — {capRemaining} / 25 jobs remaining
          </small>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            type="button"
            className="empty-state-cta"
            onClick={() => void load()}
            disabled={loading}
          >
            {loading ? 'Loading…' : 'Refresh'}
          </button>
          <button
            type="button"
            className="empty-state-cta"
            onClick={() => setModalOpen(true)}
            disabled={capRemaining <= 0}
          >
            + New job
          </button>
        </div>
      </div>

      {error ? (
        <div className="dash-error" role="alert" style={{ marginBottom: 12 }}>
          {error}
        </div>
      ) : null}

      {!loading && jobs.length === 0 ? (
        <div
          style={{
            padding: 24,
            background: 'var(--surface-2)',
            border: '1px dashed var(--border)',
            borderRadius: 8,
          }}
        >
          <h3 style={{ marginTop: 0 }}>No scheduled jobs yet</h3>
          <p style={{ color: 'var(--muted)' }}>
            Create HTTP probes that fire on a cron schedule.
          </p>
          <p style={{ color: 'var(--muted)', fontSize: 13 }}>
            <strong>Security:</strong> only <code>http_get</code> and <code>http_post</code>{' '}
            actions are allowed — no shell access. URLs must resolve to public IPs; headers
            can&apos;t include Host/Cookie/Authorization. Per-user cap is 25 jobs.
          </p>
          <button
            type="button"
            className="empty-state-cta"
            onClick={() => setModalOpen(true)}
          >
            + Create your first job
          </button>
        </div>
      ) : null}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {sortedJobs.map((job) => (
          <div
            key={job.id}
            style={{
              padding: 12,
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              opacity: job.enabled ? 1 : 0.6,
              cursor: 'pointer',
            }}
            onClick={() => void openDetail(job)}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontWeight: 600 }}>{job.name || '(unnamed)'}</div>
                <small style={{ color: 'var(--muted)' }}>
                  <code>{job.method}</code> {job.url} • <code>{job.schedule}</code>
                </small>
              </div>
              <span
                style={{
                  background: schedulerStatusColor(job.last_run_status),
                  color: 'white',
                  padding: '2px 8px',
                  borderRadius: 4,
                  fontSize: 11,
                  fontWeight: 600,
                }}
              >
                {job.last_run_status ?? 'pending'}
              </span>
              <button
                type="button"
                className="empty-state-cta"
                onClick={(e) => {
                  e.stopPropagation();
                  void toggleEnabled(job);
                }}
                disabled={busyJobId === job.id}
              >
                {job.enabled ? 'Pause' : 'Resume'}
              </button>
              <button
                type="button"
                className="empty-state-cta"
                onClick={(e) => {
                  e.stopPropagation();
                  void runNow(job);
                }}
                disabled={busyJobId === job.id}
              >
                Run
              </button>
              <button
                type="button"
                className="empty-state-cta"
                onClick={(e) => {
                  e.stopPropagation();
                  void deleteJob(job.id);
                }}
                disabled={busyJobId === job.id}
              >
                Delete
              </button>
            </div>
            <div style={{ marginTop: 6, display: 'flex', gap: 12, flexWrap: 'wrap' }}>
              <small style={{ color: 'var(--muted)' }}>
                next: {schedulerRelativeTime(job.next_run_at)}
              </small>
              <small style={{ color: 'var(--muted)' }}>
                last: {schedulerRelativeTime(job.last_run_at)}
              </small>
              {job.last_run_error ? (
                <small style={{ color: 'var(--error, #ef4444)' }}>
                  {job.last_run_error.slice(0, 80)}
                </small>
              ) : null}
            </div>
          </div>
        ))}
      </div>

      {modalOpen && (
        <AddSchedulerJobModal
          form={form}
          setForm={setForm}
          busy={busy}
          error={formError}
          onClose={() => setModalOpen(false)}
          onSubmit={() => void createJob()}
        />
      )}

      {detailJob && (
        <SchedulerJobDetailModal
          job={detailJob}
          runs={runs}
          onClose={() => setDetailJob(null)}
        />
      )}
    </motion.div>
  );
}
