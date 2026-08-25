import React, { FormEvent } from 'react';
import {
  motion,
  buttonSpring,
  useReducedMotion,
} from '../lib/motion';
import { ApiError, api } from '../lib/api';

/**
 * AuditSectionModals — Tier 9.4 (Phase 4) shared primitives + the
 * submit-handler plumbing used by AuditSection.
 *
 * Why a sibling file:
 *   The AuditSection (AuditSection.tsx) is rendered against the
 *   page's tiered layout; the modal bodies have enough form fields
 *   + handlers that they would push the section over the 400-LOC
 *   cap. Splitting them keeps both files under the limit while
 *   preserving the JSX tree ownership (the parent owns which
 *   modal is open; this file owns the primitives + the submit
 *   behavior; AuditSectionFormViews.tsx owns the per-form JSX).
 *
 * Exposes:
 *   - <Modal />, <Field />, <ModalActions /> — modal primitives
 *   - ArchiveForm, ExportForm types + initial*Form() defaults
 *   - handleCreateArchiveSubmit() — POST /audit/archive submit
 *   - handleExportSubmit()         — POST /audit/export submit
 *
 * The per-modal JSX bodies live in AuditSectionFormViews.tsx so
 * this file stays under the 400-LOC cap.
 *
 * Motion: reuses existing exports (buttonSpring) — no new variants.
 * Tokens: reuses --surface / --border / --text / --text-muted /
 * --accent / --red / --green / --amber / .threat-card-* etc. — no
 * new CSS.
 */

export type ExportFormat = 'csv' | 'json';

export interface ArchiveForm {
  periodStart: string; // yyyy-MM-ddTHH:mm (datetime-local)
  periodEnd: string;
}

export interface ExportForm {
  eventType: string;
  startTime: string;
  endTime: string;
  format: ExportFormat;
  limit: number;
}

export function initialArchiveForm(): ArchiveForm {
  const now = new Date();
  const start = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  return {
    periodStart: start.toISOString().slice(0, 16),
    periodEnd: now.toISOString().slice(0, 16),
  };
}

export function initialExportForm(): ExportForm {
  const now = new Date();
  const start = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  return {
    eventType: '',
    startTime: start.toISOString().slice(0, 16),
    endTime: now.toISOString().slice(0, 16),
    format: 'csv',
    limit: 1000,
  };
}

// ------------------------------------------------------------------
// Modal primitives
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

interface CreateArchiveArgs {
  e: FormEvent;
  archiveForm: ArchiveForm;
  setSectionBusy: (b: boolean) => void;
  onError: (msg: string) => void;
  onComplete: () => void;
}

export async function handleCreateArchiveSubmit({
  e,
  archiveForm,
  setSectionBusy,
  onError,
  onComplete,
}: CreateArchiveArgs): Promise<void> {
  e.preventDefault();
  const ps = new Date(archiveForm.periodStart);
  const pe = new Date(archiveForm.periodEnd);
  if (!ps.getTime() || !pe.getTime() || pe <= ps) {
    onError('period_end must be after period_start.');
    return;
  }
  setSectionBusy(true);
  try {
    await api('POST', '/api/v1/enterprise/audit/archive', {
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

interface ExportArgs {
  e: FormEvent;
  exportForm: ExportForm;
  setSectionBusy: (b: boolean) => void;
  sessionToken: string | null;
  onError: (msg: string) => void;
  onComplete: () => void;
}

// handleExportSubmit posts the export request and turns the binary
// response into a browser-native download via a temporary <a>.
// Streams via fetch + blob; the auth header is supplied by the
// section's sessionToken (passed in by the parent page).
export async function handleExportSubmit({
  e,
  exportForm,
  setSectionBusy,
  sessionToken,
  onError,
  onComplete,
}: ExportArgs): Promise<void> {
  e.preventDefault();
  const st = new Date(exportForm.startTime);
  const et = new Date(exportForm.endTime);
  if (!st.getTime() || !et.getTime() || et <= st) {
    onError('end_time must be after start_time.');
    return;
  }
  if (!sessionToken) {
    onError('Session token missing — cannot initiate download.');
    return;
  }
  const payload = {
    event_type: exportForm.eventType.trim(),
    start_time: st.toISOString(),
    end_time: et.toISOString(),
    format: exportForm.format,
    limit: Math.max(1, Math.min(1_000_000, exportForm.limit || 1000)),
  };
  setSectionBusy(true);
  try {
    const res = await fetch('/api/v1/enterprise/audit/export', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${sessionToken}`,
      },
      body: JSON.stringify(payload),
    });
    if (!res.ok) {
      throw new Error(`Export failed (${res.status}): ${await res.text()}`);
    }
    const blob = await res.blob();
    const dlUrl = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = dlUrl;
    a.download = `audit-export-${Date.now()}.${exportForm.format}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(dlUrl);
    onComplete();
  } catch (cause) {
    onError((cause as Error).message ?? String(cause));
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

