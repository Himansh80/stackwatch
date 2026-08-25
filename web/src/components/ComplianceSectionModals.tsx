import React, { FormEvent } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../lib/motion';
import { ApiError, api } from '../lib/api';

/**
 * ComplianceSectionModals — Tier 9.5 (Phase 5) shared primitives
 * + the submit-handler plumbing used by ComplianceSection.
 *
 * Why a sibling file:
 *   The ComplianceSection (ComplianceSection.tsx) is rendered
 *   against the page's tiered layout; the modal bodies have
 *   enough form fields + handlers that they would push the
 *   section over the 400-LOC cap. Splitting them keeps both
 *   files under the limit while preserving the JSX tree
 *   ownership (the parent owns which modal is open; this file
 *   owns the primitives + the submit behavior;
 *   ComplianceSectionFormViews.tsx owns the per-form JSX).
 *
 * Exposes:
 *   - <Modal />, <Field />, <ModalActions /> — modal primitives
 *   - GenerateReportForm, ScheduleReportForm types + initial*Form()
 *   - handleGenerateReportSubmit() — POST /compliance/reports submit
 *   - handleScheduleReportSubmit() — POST /compliance/schedules submit
 *   - handleDeleteScheduleSubmit()  — DELETE /compliance/schedules/:id
 *
 * The per-modal JSX bodies live in ComplianceSectionFormViews.tsx
 * so this file stays under the 400-LOC cap.
 *
 * Motion: reuses existing exports (buttonSpring) — no new variants.
 * Tokens: reuses --surface / --border / --text / --text-muted /
 * --accent / --red / --green / --amber / .threat-card-* etc. — no
 * new CSS.
 */

export type ComplianceFramework =
  | 'soc2'
  | 'iso27001'
  | 'hipaa'
  | 'pci'
  | 'gdpr';

export type ComplianceFrequency = 'monthly' | 'quarterly' | 'yearly';

export interface GenerateReportForm {
  framework: ComplianceFramework;
  periodStart: string;
  periodEnd: string;
}

export interface ScheduleReportForm {
  framework: ComplianceFramework;
  frequency: ComplianceFrequency;
  recipients: string;
  nextRunAt: string;
  enabled: boolean;
}

export function initialGenerateForm(): GenerateReportForm {
  const now = new Date();
  const start = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  return {
    framework: 'soc2',
    periodStart: start.toISOString().slice(0, 16),
    periodEnd: now.toISOString().slice(0, 16),
  };
}

export function initialScheduleForm(): ScheduleReportForm {
  const next = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000);
  return {
    framework: 'soc2',
    frequency: 'monthly',
    recipients: '',
    nextRunAt: next.toISOString().slice(0, 16),
    enabled: true,
  };
}

// ------------------------------------------------------------------
// Modal primitives — mirror AuditSectionModals.tsx so the visual
// language matches the rest of the Tier 9 surface.
// ------------------------------------------------------------------

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={title}
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.45)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="threat-card"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 'min(560px, 92vw)',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        <div className="threat-card-top">
          <strong className="threat-card-type">{title}</strong>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="threat-card-resolve-btn"
            style={{ marginLeft: 'auto' }}
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label
      style={{
        display: 'block',
        marginTop: 12,
        fontSize: 12,
        color: 'var(--text-muted)',
      }}
    >
      {label}
      <div style={{ marginTop: 4 }}>{children}</div>
    </label>
  );
}

export function ModalActions({
  busy,
  submitLabel,
  onClose,
}: {
  busy: boolean;
  submitLabel: string;
  onClose: () => void;
}) {
  const reduce = useReducedMotion();
  return (
    <div
      style={{
        marginTop: 16,
        display: 'flex',
        gap: 8,
        justifyContent: 'flex-end',
      }}
    >
      <motion.button
        type="button"
        className="sw-button sw-button-secondary"
        onClick={onClose}
        whileHover={reduce ? undefined : buttonSpring.whileHover}
        whileTap={reduce ? undefined : buttonSpring.whileTap}
        transition={buttonSpring.transition}
        disabled={busy}
      >
        Cancel
      </motion.button>
      <motion.button
        type="submit"
        className="empty-state-cta"
        whileHover={reduce ? undefined : buttonSpring.whileHover}
        whileTap={reduce ? undefined : buttonSpring.whileTap}
        transition={buttonSpring.transition}
        disabled={busy}
      >
        {busy ? 'Working…' : submitLabel}
      </motion.button>
    </div>
  );
}

// ------------------------------------------------------------------
// Submit handlers
// ------------------------------------------------------------------

interface GenerateArgs {
  e: FormEvent;
  generateForm: GenerateReportForm;
  setSectionBusy: (b: boolean) => void;
  onError: (msg: string) => void;
  onComplete: () => void;
}

// handleGenerateReportSubmit posts the generate request and
// triggers a refresh of the parent's list. The actual
// report-generation is asynchronous on the backend
// (POST /compliance/reports returns 201 + row id; the worker
// goroutine fills in status='completed' + artifact_path).
export async function handleGenerateReportSubmit({
  e,
  generateForm,
  setSectionBusy,
  onError,
  onComplete,
}: GenerateArgs): Promise<void> {
  e.preventDefault();
  const ps = new Date(generateForm.periodStart);
  const pe = new Date(generateForm.periodEnd);
  if (!ps.getTime() || !pe.getTime() || pe <= ps) {
    onError('period_end must be after period_start.');
    return;
  }
  setSectionBusy(true);
  try {
    await api('POST', '/api/v1/enterprise/compliance/reports', {
      framework: generateForm.framework,
      period_start: ps.toISOString(),
      period_end: pe.toISOString(),
    });
    onComplete();
  } catch (cause) {
    onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
  } finally {
    setSectionBusy(false);
  }
}

interface ScheduleArgs {
  e: FormEvent;
  scheduleForm: ScheduleReportForm;
  setSectionBusy: (b: boolean) => void;
  onError: (msg: string) => void;
  onComplete: () => void;
}

// handleScheduleReportSubmit parses the recipients string
// (comma-separated email list), validates next_run_at, and POSTs
// to /api/v1/enterprise/compliance/schedules. The future cron
// sweep reads this row.
export async function handleScheduleReportSubmit({
  e,
  scheduleForm,
  setSectionBusy,
  onError,
  onComplete,
}: ScheduleArgs): Promise<void> {
  e.preventDefault();
  const next = new Date(scheduleForm.nextRunAt);
  if (!next.getTime()) {
    onError('next_run_at is required.');
    return;
  }
  // Split comma-separated recipients into a trimmed, deduped array.
  const recipients = scheduleForm.recipients
    .split(',')
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
  if (recipients.length === 0) {
    onError('At least one recipient email is required.');
    return;
  }
  setSectionBusy(true);
  try {
    await api('POST', '/api/v1/enterprise/compliance/schedules', {
      framework: scheduleForm.framework,
      frequency: scheduleForm.frequency,
      recipients,
      enabled: scheduleForm.enabled,
      next_run_at: next.toISOString(),
    });
    onComplete();
  } catch (cause) {
    onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
  } finally {
    setSectionBusy(false);
  }
}

interface DeleteArgs {
  scheduleId: string;
  setSectionBusy: (b: boolean) => void;
  onError: (msg: string) => void;
  onComplete: () => void;
}

// handleDeleteScheduleSubmit hits DELETE /compliance/schedules/:id
// and triggers a refresh of the parent's list. Returns nothing
// (parent owns the success snackbar / refresh).
export async function handleDeleteScheduleSubmit({
  scheduleId,
  setSectionBusy,
  onError,
  onComplete,
}: DeleteArgs): Promise<void> {
  setSectionBusy(true);
  try {
    await api('DELETE', `/api/v1/enterprise/compliance/schedules/${scheduleId}`);
    onComplete();
  } catch (cause) {
    onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
  } finally {
    setSectionBusy(false);
  }
}

export const inputStyle: React.CSSProperties = {
  display: 'block',
  width: '100%',
  padding: 8,
  background: 'var(--surface-2)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-md)',
  color: 'var(--text)',
  fontFamily: 'inherit',
};
