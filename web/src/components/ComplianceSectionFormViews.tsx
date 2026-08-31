import { FormEvent } from 'react';
import Input from './shared/Input';
import Select, { type SelectOption } from './shared/Select';
import {
  ComplianceFramework,
  ComplianceFrequency,
  GenerateReportForm,
  ModalActions,
  ScheduleReportForm,
} from './ComplianceSectionModals';

/**
 * ComplianceSectionFormViews — Tier 9.5 (Phase 5) per-form modal JSX.
 *
 * Sibling of ComplianceSectionModals.tsx (which owns the modal
 * primitives + submit handlers). Splitting keeps the per-form
 * field layout (often the biggest LOC growth point) out of the
 * submit-handler file and respects the 400-LOC modular cap.
 *
 * Each view is dumb: it owns zero state of its own. The parent
 * supplies the form values + a setter (typed as a functional
 * updater so the parent can stay React-idiomatic with
 * `setGenerateForm(f => …)`).
 *
 * Tier 20 Phase E: replaced raw <input>/<select> markup with the
 * dashboard's shared <Input>/<Select> primitives. The old
 * `Field`/`inputStyle` helpers from ComplianceSectionModals are
 * gone — the shared Input/Select render their own labels + a11y.
 */

const FRAMEWORK_OPTIONS: SelectOption[] = [
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'pci', label: 'PCI DSS' },
  { value: 'gdpr', label: 'GDPR' },
];

const FREQUENCY_OPTIONS: SelectOption[] = [
  { value: 'monthly', label: 'Monthly (every 30 days)' },
  { value: 'quarterly', label: 'Quarterly (every 90 days)' },
  { value: 'yearly', label: 'Yearly (every 365 days)' },
];

export function GenerateReportFormView({
  generateForm,
  setGenerateForm,
  busy,
  onSubmit,
  onClose,
}: {
  generateForm: GenerateReportForm;
  setGenerateForm: (updater: (f: GenerateReportForm) => GenerateReportForm) => void;
  busy: boolean;
  onSubmit: (e: FormEvent) => void;
  onClose: () => void;
}) {
  return (
    <form onSubmit={onSubmit} aria-label="Generate compliance report">
      <Select
        label="Framework"
        value={generateForm.framework}
        onChange={(e) =>
          setGenerateForm((f) => ({
            ...f,
            framework: e.target.value as ComplianceFramework,
          }))
        }
        disabled={busy}
        options={FRAMEWORK_OPTIONS}
        fullWidth
      />
      <Input
        label="Period start (UTC)"
        type="datetime-local"
        value={generateForm.periodStart}
        onChange={(e) =>
          setGenerateForm((f) => ({ ...f, periodStart: e.target.value }))
        }
        required
        disabled={busy}
        fullWidth
      />
      <Input
        label="Period end (UTC)"
        type="datetime-local"
        value={generateForm.periodEnd}
        onChange={(e) =>
          setGenerateForm((f) => ({ ...f, periodEnd: e.target.value }))
        }
        required
        disabled={busy}
        fullWidth
      />
      <ModalActions
        busy={busy}
        submitLabel="Generate report"
        onClose={onClose}
      />
    </form>
  );
}

export function ScheduleReportFormView({
  scheduleForm,
  setScheduleForm,
  busy,
  onSubmit,
  onClose,
}: {
  scheduleForm: ScheduleReportForm;
  setScheduleForm: (updater: (f: ScheduleReportForm) => ScheduleReportForm) => void;
  busy: boolean;
  onSubmit: (e: FormEvent) => void;
  onClose: () => void;
}) {
  return (
    <form onSubmit={onSubmit} aria-label="Schedule recurring compliance report">
      <Select
        label="Framework"
        value={scheduleForm.framework}
        onChange={(e) =>
          setScheduleForm((f) => ({
            ...f,
            framework: e.target.value as ComplianceFramework,
          }))
        }
        disabled={busy}
        options={FRAMEWORK_OPTIONS}
        fullWidth
      />
      <Select
        label="Frequency"
        value={scheduleForm.frequency}
        onChange={(e) =>
          setScheduleForm((f) => ({
            ...f,
            frequency: e.target.value as ComplianceFrequency,
          }))
        }
        disabled={busy}
        options={FREQUENCY_OPTIONS}
        fullWidth
      />
      <Input
        label="Recipients (comma-separated email addresses)"
        type="text"
        value={scheduleForm.recipients}
        onChange={(e) =>
          setScheduleForm((f) => ({ ...f, recipients: e.target.value }))
        }
        placeholder="[email protected], [email protected]"
        disabled={busy}
        fullWidth
      />
      <Input
        label="Next run at (UTC)"
        type="datetime-local"
        value={scheduleForm.nextRunAt}
        onChange={(e) =>
          setScheduleForm((f) => ({ ...f, nextRunAt: e.target.value }))
        }
        required
        disabled={busy}
        fullWidth
      />
      <label
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          fontSize: 12,
          color: 'var(--text)',
          marginTop: 12,
        }}
      >
        <input
          type="checkbox"
          checked={scheduleForm.enabled}
          onChange={(e) =>
            setScheduleForm((f) => ({ ...f, enabled: e.target.checked }))
          }
          disabled={busy}
        />
        <span>Schedule is active (a future cron sweep will dispatch it)</span>
      </label>
      <ModalActions
        busy={busy}
        submitLabel="Create schedule"
        onClose={onClose}
      />
    </form>
  );
}
