import { useEffect, useState, type FormEvent } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * NoiseRuleEditor — Tier 8.4 (Phase 4) Datadog-style editor for a
 * single alert-noise rule. Mirrors the noiseRuleRow JSON shape
 * returned by GET /api/v1/noise/rules. Edit mode is gated on the
 * presence of `rule.id`; otherwise the form is in create mode.
 *
 * Fields:
 *   - name              (text, required, max 64 chars)
 *   - fingerprint_pattern (text, required, monospace, supports
 *                          glob patterns like `*cpu*` or `disk.fill.*`)
 *   - suppression_window dropdown: 5m / 15m / 1h / 6h / 24h / 7d
 *   - channels          (checkboxes: email / slack / pagerduty / webhook)
 *   - enabled           (toggle, default true)
 *
 * Save / Cancel buttons are rendered inside the form so the parent
 * component can wrap them in any container (modal, inline panel,
 * etc.). Reuses existing tokens (--surface, --border, --accent,
 * --red, --green) via inline styles — no new CSS.
 */

export interface NoiseRule {
  id?: string;
  name: string;
  fingerprint_pattern: string;
  suppression_window_seconds: number;
  channels: string[];
  enabled: boolean;
}

interface NoiseRuleEditorProps {
  rule?: NoiseRule | null;
  onSave: (rule: NoiseRule) => void;
  onCancel?: () => void;
  busy?: boolean;
}

// suppression window presets (label, seconds).
const WINDOWS: Array<[string, number]> = [
  ['5m', 300],
  ['15m', 900],
  ['1h', 3600],
  ['6h', 21600],
  ['24h', 86400],
  ['7d', 604800],
];

const CHANNELS = ['email', 'slack', 'pagerduty', 'webhook'] as const;
type Channel = (typeof CHANNELS)[number];

function findWindow(seconds: number): string {
  for (const [label, s] of WINDOWS) {
    if (s === seconds) return label;
  }
  return '1h';
}

function windowSeconds(label: string): number {
  for (const [l, s] of WINDOWS) {
    if (l === label) return s;
  }
  return 3600;
}

export default function NoiseRuleEditor({
  rule,
  onSave,
  onCancel,
  busy,
}: NoiseRuleEditorProps) {
  const reduce = useReducedMotion();
  const isEdit = !!rule?.id;

  const [name, setName] = useState(rule?.name ?? '');
  const [pattern, setPattern] = useState(rule?.fingerprint_pattern ?? '');
  const [windowLabel, setWindowLabel] = useState(findWindow(rule?.suppression_window_seconds ?? 3600));
  const [channels, setChannels] = useState<string[]>(rule?.channels ?? []);
  const [enabled, setEnabled] = useState(rule?.enabled ?? true);
  const [error, setError] = useState('');

  // Re-sync local state when a different rule is passed in
  // (e.g. parent opens the editor for a different row).
  useEffect(() => {
    setName(rule?.name ?? '');
    setPattern(rule?.fingerprint_pattern ?? '');
    setWindowLabel(findWindow(rule?.suppression_window_seconds ?? 3600));
    setChannels(rule?.channels ?? []);
    setEnabled(rule?.enabled ?? true);
    setError('');
  }, [rule?.id, rule?.name, rule?.fingerprint_pattern, rule?.suppression_window_seconds, rule?.enabled, rule?.channels]);

  const toggleChannel = (ch: Channel) => {
    setChannels((prev) => (prev.includes(ch) ? prev.filter((c) => c !== ch) : [...prev, ch]));
  };

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmedName = name.trim();
    const trimmedPattern = pattern.trim();
    if (!trimmedName) {
      setError('Name is required.');
      return;
    }
    if (trimmedName.length > 64) {
      setError('Name must be 64 characters or fewer.');
      return;
    }
    if (!trimmedPattern) {
      setError('Fingerprint pattern is required.');
      return;
    }
    onSave({
      id: rule?.id,
      name: trimmedName,
      fingerprint_pattern: trimmedPattern,
      suppression_window_seconds: windowSeconds(windowLabel),
      channels,
      enabled,
    });
  };

  return (
    <form className="notebook-create-form" onSubmit={handleSubmit}>
      <label>
        <span>Rule name</span>
        <input
          type="text"
          required
          maxLength={64}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="suppress noisy CPU alerts"
          aria-label="Rule name"
        />
      </label>
      <label>
        <span>Fingerprint pattern (glob: <code>*cpu*</code>, <code>disk.fill.*</code>)</span>
        <input
          type="text"
          required
          maxLength={512}
          value={pattern}
          onChange={(e) => setPattern(e.target.value)}
          placeholder="*cpu*"
          aria-label="Fingerprint pattern"
          style={{
            fontFamily: 'var(--mono, ui-monospace, SFMono-Regular, Menlo, monospace)',
            fontSize: 12,
          }}
        />
      </label>
      <label>
        <span>Suppression window</span>
        <select
          value={windowLabel}
          onChange={(e) => setWindowLabel(e.target.value)}
          aria-label="Suppression window"
        >
          {WINDOWS.map(([label]) => (
            <option key={label} value={label}>{label}</option>
          ))}
        </select>
      </label>
      <div>
        <span style={{ display: 'block', marginBottom: 6, color: 'var(--text-muted)', fontSize: 12 }}>
          Channels (empty = all)
        </span>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
          {CHANNELS.map((ch) => (
            <label
              key={ch}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                fontSize: 12,
                color: 'var(--text)',
              }}
            >
              <input
                type="checkbox"
                checked={channels.includes(ch)}
                onChange={() => toggleChannel(ch)}
              />
              {ch}
            </label>
          ))}
        </div>
      </div>
      <label
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 8,
          fontSize: 13,
        }}
      >
        <input
          type="checkbox"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
        />
        Enabled
      </label>
      {error ? (
        <div className="dash-error" role="alert" style={{ fontSize: 12 }}>
          {error}
        </div>
      ) : null}
      <div className="incident-create-actions">
        {onCancel ? (
          <motion.button
            type="button"
            className="dash-icon-button"
            onClick={onCancel}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={busy}
          >
            Cancel
          </motion.button>
        ) : null}
        <motion.button
          type="submit"
          className="empty-state-cta"
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
          disabled={busy || !name.trim() || !pattern.trim()}
        >
          {isEdit ? 'Save rule' : 'Create rule'}
        </motion.button>
      </div>
    </form>
  );
}
