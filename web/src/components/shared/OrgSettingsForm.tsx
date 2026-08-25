import { FormEvent, useEffect, useState } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * OrgSettingsForm — Tier 9.6 (Phase 6) Datadog-style settings form
 * for a single org row. Mounted inside EnterpriseOrgsSection's
 * expanded settings panel and rendered after a user clicks
 * "Edit settings" on a parent or child org.
 *
 * Shape: 3 visible fields (theme / default_dashboard / default_landing)
 * plus an open-ended JSON textarea for arbitrary custom keys (the
 * backend column is a free-form JSONB blob — see migrations/
 * 040_enterprise.sql). The form holds a working copy locally so
 * the parent can re-render without losing unsaved input.
 *
 * Tokens: reuses --surface / --border / --accent / --green / --red
 * via the existing .threat-card, .empty-state-cta, and .form-input
 * classes — no new CSS.
 *
 * Motion: buttonSpring on Save/Cancel. No new variants.
 */

export interface OrgSettings {
  theme?: 'light' | 'dark' | 'auto' | string;
  default_dashboard?: string;
  default_landing?: string;
  [key: string]: unknown;
}

export interface OrgSettingsFormProps {
  org: {
    id: string;
    name: string;
    settings?: OrgSettings | null;
  };
  /** Dashboards available in the default-dashboard dropdown. */
  dashboardOptions?: string[];
  /** Fired with the parsed new settings blob when the user clicks Save. */
  onSave: (settings: OrgSettings) => void;
  /** Fired when the user clicks Cancel. */
  onCancel?: () => void;
  busy?: boolean;
  /** Optional error string from the most recent save attempt. */
  errorMessage?: string;
}

const DEFAULT_DASHBOARD_OPTIONS = [
  'overview',
  'apm',
  'logs',
  'rum',
  'synthetics',
  'security',
  'cspm',
  'cicd',
  'database',
  'incidents',
  'notebooks',
  'intelligence',
  'enterprise',
];

const THEMES: ReadonlyArray<{ value: string; label: string }> = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'auto', label: 'Auto (follow OS)' },
];

export default function OrgSettingsForm({
  org,
  dashboardOptions,
  onSave,
  onCancel,
  busy = false,
  errorMessage,
}: OrgSettingsFormProps) {
  const reduce = useReducedMotion();
  const initial = (org.settings ?? {}) as OrgSettings;
  const dashboards = dashboardOptions ?? DEFAULT_DASHBOARD_OPTIONS;

  const [theme, setTheme] = useState<string>(initial.theme ?? 'auto');
  const [defaultDashboard, setDefaultDashboard] = useState<string>(
    initial.default_dashboard ?? 'overview',
  );
  const [defaultLanding, setDefaultLanding] = useState<string>(
    initial.default_landing ?? '/dashboard',
  );
  const [customJson, setCustomJson] = useState<string>(() => {
    // Strip the three known keys so the user sees only their custom
    // additions in the textarea.
    const cloned: Record<string, unknown> = { ...initial };
    delete cloned.theme;
    delete cloned.default_dashboard;
    delete cloned.default_landing;
    const keys = Object.keys(cloned);
    if (keys.length === 0) return '';
    try {
      return JSON.stringify(cloned, null, 2);
    } catch {
      return '';
    }
  });

  // Reset the local state when the org prop changes (parent org
  // owner switches which org is being edited).
  useEffect(() => {
    setTheme(initial.theme ?? 'auto');
    setDefaultDashboard(initial.default_dashboard ?? 'overview');
    setDefaultLanding(initial.default_landing ?? '/dashboard');
    const cloned: Record<string, unknown> = { ...initial };
    delete cloned.theme;
    delete cloned.default_dashboard;
    delete cloned.default_landing;
    setCustomJson(
      Object.keys(cloned).length === 0
        ? ''
        : (() => {
            try {
              return JSON.stringify(cloned, null, 2);
            } catch {
              return '';
            }
          })(),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [org.id]);

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    let custom: Record<string, unknown> = {};
    const trimmed = customJson.trim();
    if (trimmed) {
      try {
        const parsed = JSON.parse(trimmed);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          custom = parsed as Record<string, unknown>;
        }
      } catch {
        // Surface a friendly error and don't submit. The parent
        // doesn't get an errorMessage prop back here, so we use a
        // quick alert — UI feedback only.
        window.alert('Custom settings must be a JSON object.');
        return;
      }
    }
    onSave({
      ...custom,
      theme,
      default_dashboard: defaultDashboard,
      default_landing: defaultLanding,
    });
  };

  return (
    <form className="threat-card" onSubmit={handleSubmit} aria-label={`Settings for ${org.name}`}>
      <div className="threat-card-top">
        <strong className="threat-card-type">Org settings — {org.name}</strong>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: 12,
          marginTop: 12,
        }}
      >
        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Theme</span>
          <select
            className="form-input"
            value={theme}
            onChange={(e) => setTheme(e.target.value)}
            disabled={busy}
          >
            {THEMES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Default dashboard</span>
          <select
            className="form-input"
            value={defaultDashboard}
            onChange={(e) => setDefaultDashboard(e.target.value)}
            disabled={busy}
          >
            {dashboards.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Default landing path</span>
          <input
            className="form-input"
            type="text"
            value={defaultLanding}
            onChange={(e) => setDefaultLanding(e.target.value)}
            placeholder="/dashboard"
            disabled={busy}
          />
        </label>
      </div>

      <label
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 4,
          fontSize: 12,
          marginTop: 12,
        }}
      >
        <span style={{ color: 'var(--muted)' }}>
          Custom settings (JSON object — open-ended for future fields)
        </span>
        <textarea
          className="form-input"
          value={customJson}
          onChange={(e) => setCustomJson(e.target.value)}
          placeholder='{ "alert_email": "ops@example.com" }'
          rows={4}
          style={{ fontFamily: 'ui-monospace, SFMono-Regular, monospace', fontSize: 12 }}
          disabled={busy}
        />
      </label>

      {errorMessage ? (
        <div
          role="alert"
          style={{
            marginTop: 8,
            padding: '6px 10px',
            background: 'rgba(239, 68, 68, 0.08)',
            border: '1px solid rgba(239, 68, 68, 0.4)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--red, #ef4444)',
            fontSize: 12,
          }}
        >
          {errorMessage}
        </div>
      ) : null}

      <div style={{ display: 'flex', gap: 8, marginTop: 12, justifyContent: 'flex-end' }}>
        {onCancel ? (
          <motion.button
            type="button"
            className="empty-state-cta"
            onClick={onCancel}
            disabled={busy}
            style={{ background: 'transparent', border: '1px solid var(--border)' }}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
          >
            Cancel
          </motion.button>
        ) : null}
        <motion.button
          type="submit"
          className="empty-state-cta"
          disabled={busy}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
        >
          {busy ? 'Saving…' : 'Save settings'}
        </motion.button>
      </div>
    </form>
  );
}
