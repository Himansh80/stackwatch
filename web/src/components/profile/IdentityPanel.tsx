interface IdentityRow {
  key: string;
  label: string;
  value: string;
  mono: boolean;
}

interface IdentityPanelProps {
  rows: IdentityRow[];
  onCopy: (value: string, field: string) => void;
  copiedField: string | null;
}

/**
 * IdentityPanel — left card on the Profile page. Shows the user's
 * technical identifiers (user_id, tenant_id, workspace slug, email)
 * with a copy button per row.
 */
export default function IdentityPanel({ rows, onCopy, copiedField }: IdentityPanelProps) {
  return (
    <article className="dash-panel prof-panel">
      <div className="dash-panel-head">
        <div>
          <span className="dash-eyebrow">Identity</span>
          <h3>Technical identifiers</h3>
        </div>
        <span className="dash-panel-context">Needed for API calls and scripts</span>
      </div>
      <div className="prof-id-list">
        {rows.map((row) => (
          <div key={row.key} className="prof-id-row">
            <span className="prof-id-label">{row.label}</span>
            <span className={`prof-id-value ${row.mono ? 'prof-id-mono' : ''}`}>
              {row.value || '—'}
            </span>
            <button
              type="button"
              className="prof-id-copy"
              onClick={() => row.value && onCopy(row.value, row.key)}
              disabled={!row.value}
              aria-label={`Copy ${row.label}`}
              title={row.value ? 'Copy to clipboard' : 'No value to copy'}
            >
              {copiedField === row.key ? '✓ Copied' : '⧉ Copy'}
            </button>
          </div>
        ))}
      </div>
    </article>
  );
}

/**
 * Build the standard 4-row identity list from a Profile object.
 * Helper exported so the parent page doesn't have to repeat the
 * row config every time the profile loads.
 */
export function buildIdentityRows(profile: { user_id: string; tenant_id: string; tenant_slug: string; email: string }): IdentityRow[] {
  return [
    { key: 'user_id', label: 'User ID', value: profile.user_id, mono: true },
    { key: 'tenant_id', label: 'Tenant ID', value: profile.tenant_id, mono: true },
    { key: 'tenant_slug', label: 'Workspace slug', value: profile.tenant_slug, mono: true },
    { key: 'email', label: 'Email address', value: profile.email, mono: false },
  ];
}

/**
 * Copy `text` to the clipboard. Falls back to a manual selection
 * if the async Clipboard API isn't available (older browsers, http
 * origins without user gesture).
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
    const range = document.createRange();
    const sel = window.getSelection();
    const el = document.createElement('textarea');
    el.value = text;
    el.style.position = 'fixed';
    el.style.opacity = '0';
    document.body.appendChild(el);
    range.selectNodeContents(el);
    if (sel) { sel.removeAllRanges(); sel.addRange(range); }
    document.execCommand('copy');
    document.body.removeChild(el);
    if (sel) sel.removeAllRanges();
    return true;
  } catch {
    return false;
  }
}
