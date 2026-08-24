/**
 * ComplianceBar — horizontal stacked bar showing pass / fail / unknown
 * counts for a single compliance framework.
 *
 * Used on SecurityPage → Compliance tab. One widget per framework.
 * Bar segments are sized proportionally to (count / total). When total
 * is 0 the bar renders an empty state so the layout doesn't shift.
 *
 * Visual contract (tokens only):
 *   - pass    → var(--green) / var(--green-soft) / var(--green-text)
 *   - fail    → var(--red)   / var(--red-soft)   / var(--red-text)
 *   - unknown → var(--muted) / transparent       / var(--muted)
 *   - label uppercase 10px, counts 12px tabular-nums
 */
export interface ComplianceBarProps {
  framework: 'pci' | 'soc2' | 'gdpr' | 'hipaa' | string;
  total: number;
  pass: number;
  fail: number;
  unknown: number;
  /** Optional click handler (expand to rule list). */
  onClick?: () => void;
}

function pct(part: number, total: number): number {
  if (total <= 0) return 0;
  return Math.round((part / total) * 100);
}

export default function ComplianceBar({
  framework,
  total,
  pass,
  fail,
  unknown,
  onClick,
}: ComplianceBarProps) {
  const label = (framework || '').toUpperCase();
  const passPct = pct(pass, total);
  const failPct = pct(fail, total);
  const unknownPct = pct(unknown, total);
  return (
    <article
      className={`compliance-bar ${onClick ? 'compliance-bar-clickable' : ''}`}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onClick();
              }
            }
          : undefined
      }
    >
      <div className="compliance-bar-head">
        <strong className="compliance-bar-label">{label || '—'}</strong>
        <span className="compliance-bar-total">{total} rule{total === 1 ? '' : 's'}</span>
      </div>
      <div className="compliance-bar-track" aria-label={`${label} pass/fail breakdown`}>
        {total > 0 ? (
          <>
            {passPct > 0 ? (
              <span
                className="compliance-bar-seg compliance-bar-seg-pass"
                style={{ width: `${passPct}%` }}
                title={`${pass} pass (${passPct}%)`}
              />
            ) : null}
            {failPct > 0 ? (
              <span
                className="compliance-bar-seg compliance-bar-seg-fail"
                style={{ width: `${failPct}%` }}
                title={`${fail} fail (${failPct}%)`}
              />
            ) : null}
            {unknownPct > 0 ? (
              <span
                className="compliance-bar-seg compliance-bar-seg-unknown"
                style={{ width: `${unknownPct}%` }}
                title={`${unknown} unknown (${unknownPct}%)`}
              />
            ) : null}
          </>
        ) : (
          <span className="compliance-bar-empty">No rules defined</span>
        )}
      </div>
      <div className="compliance-bar-counts">
        <span className="compliance-bar-count compliance-bar-count-pass">
          <span className="compliance-bar-dot compliance-bar-dot-pass" aria-hidden="true" />
          {pass} pass
        </span>
        <span className="compliance-bar-count compliance-bar-count-fail">
          <span className="compliance-bar-dot compliance-bar-dot-fail" aria-hidden="true" />
          {fail} fail
        </span>
        <span className="compliance-bar-count compliance-bar-count-unknown">
          <span className="compliance-bar-dot compliance-bar-dot-unknown" aria-hidden="true" />
          {unknown} unknown
        </span>
      </div>
    </article>
  );
}