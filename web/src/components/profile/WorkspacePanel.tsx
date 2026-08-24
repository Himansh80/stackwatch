interface StatCardProps {
  label: string;
  value: string;
  hint: string;
  tone: 'cyan' | 'indigo' | 'green' | 'amber' | 'red';
}

function StatCard({ label, value, hint, tone }: StatCardProps) {
  return (
    <div className={`prof-stat prof-stat-${tone}`}>
      <span className="prof-stat-label">{label}</span>
      <strong className="prof-stat-value">{value}</strong>
      <span className="prof-stat-hint">{hint}</span>
    </div>
  );
}

interface WorkspacePanelProps {
  tenantName: string;
  tenantPlan: string;
  tenantSlug: string;
  status: string;
  role: string;
  tone: 'admin' | 'operator' | 'viewer' | 'neutral';
}

/**
 * WorkspacePanel — right card on the first Profile row. Shows a 2x2
 * stat grid (Plan / Workspace / Status / Role) for the user's
 * workspace.
 */
export default function WorkspacePanel({
  tenantName,
  tenantPlan,
  tenantSlug,
  status,
  role,
  tone,
}: WorkspacePanelProps) {
  return (
    <article className="dash-panel prof-panel">
      <div className="dash-panel-head">
        <div>
          <span className="dash-eyebrow">Workspace</span>
          <h3>{tenantName || 'Your workspace'}</h3>
        </div>
        <span className="prof-plan-badge">{tenantPlan || 'free'}</span>
      </div>
      <div className="prof-workspace-body">
        <StatCard label="Plan" value={tenantPlan || 'free'} hint="Subscription tier" tone="cyan" />
        <StatCard label="Workspace" value={tenantName || '—'} hint={`slug · ${tenantSlug || '—'}`} tone="indigo" />
        <StatCard label="Status" value={status || 'unknown'} hint="Account state" tone={status === 'active' ? 'green' : 'amber'} />
        <StatCard label="Role" value={role || '—'} hint="Permission level" tone={tone === 'admin' ? 'amber' : 'cyan'} />
      </div>
    </article>
  );
}
