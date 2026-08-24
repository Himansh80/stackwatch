/**
 * PipelineCard — compact card showing one cicd_pipelines row.
 *
 * Used on CicdPage → Pipelines tab. Renders:
 *   - provider pill (GH/GL/JK/CI)
 *   - repo (mono) + branch (mono, muted)
 *   - commit SHA (7 chars, mono)
 *   - status pill (success=green, failed=red, running=amber, pending/cancelled=muted)
 *   - duration (formatted: "1m 23s")
 *   - started-at relative time ("2 min ago")
 *   - deployment_count badge if non-zero
 *
 * Design tokens only (--green, --amber, --red, --green-soft, --muted, etc).
 * No hex literals. Reuses .dash-status and .dash-status-* tones.
 */

export interface PipelineCardProps {
  pipeline: {
    id: string;
    provider: 'github' | 'gitlab' | 'jenkins' | 'circleci' | string;
    repo: string;
    branch?: string | null;
    commit_sha?: string | null;
    status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | string;
    started_at: string;
    finished_at?: string | null;
    duration_ms?: number;
  };
  deploymentCount?: number;
}

const PROVIDER_ICON: Record<string, string> = {
  github: 'GH',
  gitlab: 'GL',
  jenkins: 'JK',
  circleci: 'CI',
};

function providerLabel(p: string): string {
  return PROVIDER_ICON[p.toLowerCase()] ?? p.slice(0, 2).toUpperCase();
}

function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '—';
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ${sec % 60}s`;
  const hr = Math.floor(min / 60);
  return `${hr}h ${min % 60}m`;
}

function shortSHA(s: string | null | undefined): string {
  if (!s) return '—';
  return s.length > 7 ? s.slice(0, 7) : s;
}

function relativeTime(iso: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function statusTone(status: string): string {
  switch (status.toLowerCase()) {
    case 'success':
      return 'dash-status dash-status-good';
    case 'failed':
      return 'dash-status dash-status-bad';
    case 'running':
      return 'dash-status dash-status-warn';
    case 'cancelled':
    case 'pending':
    default:
      return 'dash-status dash-status-muted';
  }
}

export default function PipelineCard({ pipeline, deploymentCount = 0 }: PipelineCardProps) {
  const prov = (pipeline.provider || '').toLowerCase();
  const dur = formatDuration(pipeline.duration_ms ?? 0);
  return (
    <article className="pipeline-card">
      <div className="pipeline-card-top">
        <span className="pipeline-card-provider" data-provider={prov}>
          {providerLabel(prov)}
        </span>
        <strong className="pipeline-card-repo">
          <code>{pipeline.repo || '—'}</code>
        </strong>
        {pipeline.branch ? (
          <span className="pipeline-card-branch">
            <code>{pipeline.branch}</code>
          </span>
        ) : null}
        <code className="pipeline-card-sha" title={pipeline.commit_sha || ''}>
          {shortSHA(pipeline.commit_sha)}
        </code>
        <span className="pipeline-card-time" title={pipeline.started_at}>
          {relativeTime(pipeline.started_at)}
        </span>
      </div>
      <div className="pipeline-card-meta">
        <span className={statusTone(pipeline.status)}>
          <span className="dash-status-dot" aria-hidden="true" />
          {pipeline.status}
        </span>
        <span className="pipeline-card-duration" title={`Duration ${dur}`}>
          {dur}
        </span>
        {deploymentCount > 0 ? (
          <span className="pipeline-card-deploy-pill">
            <span className="pipeline-card-deploy-label">deploys</span>
            <strong>{deploymentCount}</strong>
          </span>
        ) : null}
      </div>
    </article>
  );
}