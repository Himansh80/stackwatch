import { FormEvent } from 'react';
import {
  ArchiveForm,
  ExportForm,
  ExportFormat,
  Field,
  ModalActions,
  inputStyle,
} from './AuditSectionModals';

/**
 * AuditSectionFormViews — Tier 9.4 (Phase 4) per-form modal JSX.
 *
 * Sibling of AuditSectionModals.tsx (which owns the modal
 * primitives + submit handlers). Splitting keeps the per-form
 * field layout (often the biggest LOC growth point) out of the
 * submit-handler file and respects the 400-LOC modular cap.
 *
 * Each view is dumb: it owns zero state of its own. The parent
 * supplies the form values + a setter (typed as a functional
 * updater so the parent can stay React-idiomatic with
 * `setExportForm(f => …)`).
 */

export function ArchiveFormView({
  archiveForm,
  setArchiveForm,
  busy,
  onSubmit,
  onClose,
}: {
  archiveForm: ArchiveForm;
  setArchiveForm: (updater: (f: ArchiveForm) => ArchiveForm) => void;
  busy: boolean;
  onSubmit: (e: FormEvent) => void;
  onClose: () => void;
}) {
  return (
    <form onSubmit={onSubmit} aria-label="Create audit archive">
      <Field label="Period start (UTC)">
        <input
          type="datetime-local"
          value={archiveForm.periodStart}
          onChange={(e) =>
            setArchiveForm((f) => ({ ...f, periodStart: e.target.value }))
          }
          required
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <Field label="Period end (UTC)">
        <input
          type="datetime-local"
          value={archiveForm.periodEnd}
          onChange={(e) =>
            setArchiveForm((f) => ({ ...f, periodEnd: e.target.value }))
          }
          required
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <ModalActions
        busy={busy}
        submitLabel="Start archive job"
        onClose={onClose}
      />
    </form>
  );
}

export function ExportFormView({
  exportForm,
  setExportForm,
  busy,
  onSubmit,
  onClose,
}: {
  exportForm: ExportForm;
  setExportForm: (updater: (f: ExportForm) => ExportForm) => void;
  busy: boolean;
  onSubmit: (e: FormEvent) => void;
  onClose: () => void;
}) {
  return (
    <form onSubmit={onSubmit} aria-label="Export audit events">
      <Field label="Event type filter (optional, e.g. sso.callback)">
        <input
          type="text"
          value={exportForm.eventType}
          placeholder="leave empty for all events"
          maxLength={64}
          onChange={(e) =>
            setExportForm((f) => ({ ...f, eventType: e.target.value }))
          }
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <Field label="Start (UTC)">
        <input
          type="datetime-local"
          value={exportForm.startTime}
          onChange={(e) =>
            setExportForm((f) => ({ ...f, startTime: e.target.value }))
          }
          required
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <Field label="End (UTC)">
        <input
          type="datetime-local"
          value={exportForm.endTime}
          onChange={(e) =>
            setExportForm((f) => ({ ...f, endTime: e.target.value }))
          }
          required
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <Field label="Format">
        <select
          value={exportForm.format}
          onChange={(e) =>
            setExportForm((f) => ({
              ...f,
              format: e.target.value as ExportFormat,
            }))
          }
          disabled={busy}
          style={inputStyle}
        >
          <option value="csv">CSV (RFC 4180, with header)</option>
          <option value="json">NDJSON (one event per line)</option>
        </select>
      </Field>
      <Field label="Row limit (1 — 1,000,000)">
        <input
          type="number"
          value={exportForm.limit}
          min={1}
          max={1_000_000}
          onChange={(e) =>
            setExportForm((f) => ({
              ...f,
              limit: parseInt(e.target.value, 10) || 1000,
            }))
          }
          disabled={busy}
          style={inputStyle}
        />
      </Field>
      <ModalActions busy={busy} submitLabel="Download" onClose={onClose} />
    </form>
  );
}
