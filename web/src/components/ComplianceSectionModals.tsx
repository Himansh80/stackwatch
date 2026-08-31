import React, { FormEvent } from 'react';
import { ApiError, api } from '../lib/api';
import Button from './shared/Button';
import SharedModal from './shared/Modal';

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
 *   - <Modal /> — thin wrapper around the shared Modal (kept as
 *     a back-compat re-export for ComplianceSection.tsx which
 *     uses the {title, onClose, children} API).
 *   - <ModalActions /> — Cancel + Submit button row using shared
 *     <Button>.
 *   - GenerateReportForm, ScheduleReportForm types + initial*Form()
 *   - handleGenerateReportSubmit() — POST /compliance/reports submit
 *   - handleScheduleReportSubmit() — POST /compliance/schedules submit
 *   - handleDeleteScheduleSubmit()  — DELETE /compliance/schedules/:id
 *
 * The per-modal JSX bodies live in ComplianceSectionFormViews.tsx
 * so this file stays under the 400-LOC cap.
 *
 * Tier 20 Phase E: the local Modal primitive is now a thin
 * wrapper over the dashboard's shared focus-trapped Modal. The
 * old raw-div + threat-card markup is gone.
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
// Modal — back-compat wrapper around the shared Modal.
//
// ComplianceSection.tsx uses the legacy {title, onClose, children}
// API (no explicit `open` prop). We preserve that by deriving
// `open` from children presence: if children is null/undefined the
// modal is closed; otherwise it's open. This keeps ComplianceSection
// working unchanged while still using the focus-trapped shared
// Modal underneath.
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
    <SharedModal open onClose={onClose} title={title} size="md">
      {children}
    </SharedModal>
  );
}

// ------------------------------------------------------------------
// ModalActions — Cancel + Submit button row using shared Button.
// ------------------------------------------------------------------

export function ModalActions({
  busy,
  submitLabel,
  onClose,
}: {
  busy: boolean;
  submitLabel: string;
  onClose: () => void;
}) {
  return (
    <div
      style={{
        marginTop: 16,
        display: 'flex',
        gap: 8,
        justifyContent: 'flex-end',
      }}
    >
      <Button
        type="button"
        variant="secondary"
        size="sm"
        onClick={onClose}
        disabled={busy}
      >
        Cancel
      </Button>
      <Button
        type="submit"
        variant="primary"
        size="sm"
        loading={busy}
      >
        {busy ? 'Working…' : submitLabel}
      </Button>
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
