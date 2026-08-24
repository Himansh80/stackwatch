import { useCallback, useEffect, useMemo, useState } from 'react';
import EmptyState from './shared/EmptyState';
import KpiCard from './shared/KpiCard';
import NoiseRuleEditor, { NoiseRule } from './shared/NoiseRuleEditor';
import SnoozeHistoryPanel, { SnoozeRow } from './shared/SnoozeHistoryPanel';
import { ApiError, api } from '../lib/api';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../lib/motion';

/**
 * NoiseReductionSection — Tier 8.4 (Phase 4) UI surface for the
 * Alert Noise Reduction page section. Mirrors the Phase 2+3
 * extracted-section pattern so the IntelligencePage stays under
 * the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: rules active / suppressed today / snoozes active
 *   - Rules list (each row: name / pattern / window / channels /
 *     enabled toggle + delete button)
 *   - "New rule" button → NoiseRuleEditor modal (POST /noise/rules)
 *   - SnoozeHistoryPanel (extracted) — snooze log + snooze modal
 *
 * Motion: existing exports (kpiStagger, buttonSpring) — no new
 * variants. Reuses existing tokens (--surface, --border, --accent,
 * --green, --amber, --red) via inline styles.
 */

export interface NoiseRuleRow {
  id: string;
  name: string;
  fingerprint_pattern: string;
  suppression_window_seconds: number;
  channels: string[];
  enabled: boolean;
  created_at: string;
}

interface NoiseReductionSectionProps {
  rules: NoiseRuleRow[];
  snoozes: SnoozeRow[];
  busy: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

function windowLabel(seconds: number): string {
  if (seconds >= 604800) return '7d';
  if (seconds >= 86400) return '24h';
  if (seconds >= 21600) return '6h';
  if (seconds >= 3600) return '1h';
  if (seconds >= 900) return '15m';
  if (seconds >= 300) return '5m';
  return `${seconds}s`;
}

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

export default function NoiseReductionSection({
  rules,
  snoozes,
  busy,
  onError,
  onChanged,
}: NoiseReductionSectionProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  // New-rule modal state. The snooze modal lives inside the
  // SnoozeHistoryPanel sub-component (extracted to keep this file
  // under the 400-LOC cap).
  const [ruleModal, setRuleModal] = useState(false);

  // KPIs derived locally from the rules + snoozes lists.
  const counts = useMemo(() => {
    const activeRules = rules.filter((r) => r.enabled).length;
    const activeSnoozes = snoozes.filter((s) => s.active).length;
    return {
      activeRules,
      suppressed: activeSnoozes,
      activeSnoozes,
    };
  }, [rules, snoozes]);

  // Toggle a rule's enabled state — Phase 4 has no PATCH endpoint
  // so we recreate with the flipped `enabled` flag (suffixed with
  // " (paused)" when disabling so the audit trail is clear).
  const toggleRule = useCallback(
    async (rule: NoiseRuleRow) => {
      setSectionBusy(true);
      try {
        await api('POST', '/api/v1/noise/rules', {
          name: rule.enabled ? rule.name + ' (paused)' : rule.name.replace(/ \(paused\)$/, ''),
          fingerprint_pattern: rule.fingerprint_pattern,
          suppression_window_seconds: rule.suppression_window_seconds,
          channels: rule.channels,
          enabled: !rule.enabled,
        });
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  const deleteRule = useCallback(
    async (rule: NoiseRuleRow) => {
      if (!window.confirm(`Delete noise rule "${rule.name}"?`)) return;
      setSectionBusy(true);
      try {
        await api('DELETE', `/api/v1/noise/rules/${rule.id}`);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  const saveRule = useCallback(
    async (rule: NoiseRule) => {
      setSectionBusy(true);
      try {
        await api('POST', '/api/v1/noise/rules', {
          name: rule.name,
          fingerprint_pattern: rule.fingerprint_pattern,
          suppression_window_seconds: rule.suppression_window_seconds,
          channels: rule.channels,
          enabled: rule.enabled,
        });
        setRuleModal(false);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  // Close rule modal on Escape.
  useEffect(() => {
    if (!ruleModal) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setRuleModal(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [ruleModal]);

  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{ marginTop: 16 }}
      >
        <KpiCard
          label="Rules active"
          value={counts.activeRules}
          status={counts.activeRules > 0 ? 'up' : 'neutral'}
          accent={counts.activeRules > 0 ? 'green' : 'cyan'}
        />
        <KpiCard
          label="Suppressed today"
          value={counts.suppressed}
          status={counts.suppressed > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
        <KpiCard
          label="Snoozes active"
          value={counts.activeSnoozes}
          status={counts.activeSnoozes > 0 ? 'stale' : 'neutral'}
          accent={counts.activeSnoozes > 0 ? 'amber' : 'cyan'}
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Alert Noise Reduction</span>
        <h2 className="dash-section-title">Suppression rules</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Operator-authored rules that suppress duplicate alerts by fingerprint
          pattern (glob: <code>*cpu*</code>, <code>disk.fill.*</code>). Each rule
          applies for its <em>suppression window</em> and only to the selected
          channels (empty = all).
        </p>
        {rules.length === 0 ? (
          <EmptyState
            illustration={<span style={{ fontSize: 36 }}>⌬</span>}
            headline="No noise rules yet"
            subhead="Create a rule to suppress duplicates of a known-noisy metric. Rules match alerts by fingerprint pattern and apply for the suppression window."
          />
        ) : (
          <div className="threat-card-list">
            {rules.map((r) => (
              <div key={r.id} className="threat-card">
                <div className="threat-card-top">
                  <span
                    className={`dash-status ${
                      r.enabled ? 'dash-status-up' : 'dash-status-stale'
                    }`}
                  >
                    <span className="dash-status-dot" aria-hidden="true" />
                    {r.enabled ? 'enabled' : 'paused'}
                  </span>
                  <strong className="threat-card-type">{r.name}</strong>
                  <span className="threat-card-time" title={r.created_at}>
                    {new Date(r.created_at).toLocaleString()}
                  </span>
                </div>
                <p className="threat-card-desc">
                  pattern <code style={{ color: 'var(--accent)' }}>{r.fingerprint_pattern}</code> ·
                  window <strong>{windowLabel(r.suppression_window_seconds)}</strong> ·
                  channels <strong>{r.channels.length === 0 ? 'all' : r.channels.join(', ')}</strong>
                </p>
                <div className="threat-card-meta">
                  <button
                    type="button"
                    className="threat-card-resolve-btn"
                    onClick={() => void toggleRule(r)}
                    disabled={isBusy}
                  >
                    {r.enabled ? 'Pause' : 'Enable'}
                  </button>
                  <button
                    type="button"
                    className="threat-card-resolve-btn"
                    onClick={() => void deleteRule(r)}
                    disabled={isBusy}
                    style={{ color: 'var(--red)' }}
                  >
                    Delete
                  </button>
                  <span style={{ marginLeft: 'auto', fontSize: 11, color: 'var(--text-muted)' }}>
                    {shortId(r.id)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}

        <div style={{ marginTop: 12 }}>
          <motion.button
            type="button"
            className="empty-state-cta"
            onClick={() => setRuleModal(true)}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={isBusy}
          >
            + New rule
          </motion.button>
        </div>
      </section>

      {/* Snooze log + modal (extracted to SnoozeHistoryPanel so this
          file stays under the 400-LOC cap). */}
      <SnoozeHistoryPanel
        snoozes={snoozes}
        busy={isBusy}
        onError={onError}
        onChanged={onChanged}
      />

      {ruleModal ? (
        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
          <div className="slow-query-explain-modal-head">
            <strong>New noise rule</strong>
            <button
              type="button"
              className="slow-query-explain-close"
              onClick={() => setRuleModal(false)}
              aria-label="Close"
            >
              ✕
            </button>
          </div>
          <NoiseRuleEditor
            onSave={saveRule}
            onCancel={() => setRuleModal(false)}
            busy={isBusy}
          />
        </div>
      ) : null}
    </>
  );
}
