import { FormEvent, useEffect, useState } from 'react';
import Button from './Button';
import Input from './Input';
import Select from './Select';
import Textarea from './Textarea';

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
        <Select
          label="Theme"
          value={theme}
          onChange={(e) => setTheme(e.target.value)}
          disabled={busy}
          options={THEMES}
          fullWidth
        />

        <Select
          label="Default dashboard"
          value={defaultDashboard}
          onChange={(e) => setDefaultDashboard(e.target.value)}
          disabled={busy}
          options={dashboards.map((d) => ({ value: d, label: d }))}
          fullWidth
        />

        <Input
          label="Default landing path"
          type="text"
          value={defaultLanding}
          onChange={(e) => setDefaultLanding(e.target.value)}
          placeholder="/dashboard"
          disabled={busy}
          fullWidth
        />

        <Textarea
          label="Custom settings (JSON object — open-ended for future fields)"
          value={customJson}
          onChange={(e) => setCustomJson(e.target.value)}
          placeholder='{ "alert_email": "ops@example.com" }'
          rows={4}
          disabled={busy}
          fullWidth
          style={{
            fontFamily: 'ui-monospace, SFMono-Regular, monospace',
            fontSize: 12,
          }}
        />
      </div>

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
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onCancel}
            disabled={busy}
          >
            Cancel
          </Button>
        ) : null}
        <Button
          type="submit"
          variant="primary"
          size="sm"
          disabled={busy}
          loading={busy}
        >
          {busy ? 'Saving…' : 'Save settings'}
        </Button>
      </div>
    </form>
  );
}
