import {
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import KpiCard from './shared/KpiCard';
import ComplianceReportCard, {
  ComplianceReportRow,
} from './shared/ComplianceReportCard';
import {
  motion,
  buttonSpring,
  kpiStagger,
  useReducedMotion,
} from '../lib/motion';
import {
  GenerateReportForm,
  Modal,
  ScheduleReportForm,
  handleDeleteScheduleSubmit,
  handleGenerateReportSubmit,
  handleScheduleReportSubmit,
  initialGenerateForm,
  initialScheduleForm,
} from './ComplianceSectionModals';
import {
  GenerateReportFormView,
  ScheduleReportFormView,
} from './ComplianceSectionFormViews';
import { downloadReport } from './ComplianceSectionHelpers';
import ComplianceScheduleList from './ComplianceScheduleList';

/**
 * ComplianceSection — Tier 9.5 (Phase 5) UI surface for the
 * Compliance Reports page section. Mirrors the SsoSection /
 * ScimSection / RbacSection / AuditSection pattern so the future
 * EnterprisePage (Phase 6) can compose this in without going
 * over the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: total reports / framework breakdown /
 *     completed today, driven by kpiStagger for entrance.
 *   - Two tabs: Reports (list of compliance_reports) and
 *     Schedules (list of compliance_schedules).
 *   - Top-right action buttons: "Generate report" (always) +
 *     "Schedule report" (always). Both open modal forms.
 *   - Reports tab renders a stack of ComplianceReportCards.
 *   - Schedules tab renders a simple list with delete buttons.
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring) —
 * no new variants.
 */

// Schedule row shape returned by GET /compliance/schedules.
// Mirrors complianceScheduleRow in handlers_compliance_types.go.
export interface ComplianceScheduleRow {
  id: string;
  tenant_id: string;
  framework: string;
  frequency: 'monthly' | 'quarterly' | 'yearly' | string;
  recipients: string[];
  enabled: boolean;
  next_run_at: string;
  created_at: string;
}

interface ComplianceSectionProps {
  reports: ComplianceReportRow[];
  schedules: ComplianceScheduleRow[];
  busy?: boolean;
  onError: (msg: string) => void;
  onReportsChanged: () => void;
  onSchedulesChanged: () => void;
  // The session token to use for browser-native downloads. We
  // can't use the api() helper because Content-Disposition
  // bodies don't deserialize as JSON. Provided by the parent
  // (EnterprisePage / dev console) which owns auth header
  // construction.
  sessionToken?: string | null;
}

type Tab = 'reports' | 'schedules';

export default function ComplianceSection({
  reports,
  schedules,
  busy = false,
  onError,
  onReportsChanged,
  onSchedulesChanged,
  sessionToken,
}: ComplianceSectionProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;
  const [tab, setTab] = useState<Tab>('reports');
  const [showGenerate, setShowGenerate] = useState(false);
  const [showSchedule, setShowSchedule] = useState(false);
  const [generateForm, setGenerateForm] =
    useState<GenerateReportForm>(initialGenerateForm);
  const [scheduleForm, setScheduleForm] =
    useState<ScheduleReportForm>(initialScheduleForm);

  // Sorted: running/pending first (operator wants to see them),
  // then completed, then failed; within each bucket by
  // created_at DESC.
  const sortedReports = useMemo(() => {
    const bucketOrder: Record<string, number> = {
      running: 0,
      pending: 1,
      failed: 2,
      completed: 3,
    };
    return [...reports].sort((a, b) => {
      const ba = bucketOrder[a.status] ?? 9;
      const bb = bucketOrder[b.status] ?? 9;
      if (ba !== bb) return ba - bb;
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }, [reports]);

  // KPI computations:
  //   total      — count of all reports
  //   frameworks — number of distinct frameworks represented
  //   today      — count of reports with created_at within last 24h
  const totalReports = reports.length;
  const distinctFrameworks = useMemo(
    () => new Set(reports.map((r) => r.framework)).size,
    [reports],
  );
  const reportsLast24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    return reports.filter((r) => new Date(r.created_at).getTime() >= cutoff)
      .length;
  }, [reports]);

  const handleGenerate = useCallback(
    async (e: FormEvent) => {
      await handleGenerateReportSubmit({
        e,
        generateForm,
        setSectionBusy,
        onError,
        onComplete: () => {
          setShowGenerate(false);
          setGenerateForm(initialGenerateForm());
          onReportsChanged();
        },
      });
    },
    [generateForm, onError, onReportsChanged],
  );

  const handleSchedule = useCallback(
    async (e: FormEvent) => {
      await handleScheduleReportSubmit({
        e,
        scheduleForm,
        setSectionBusy,
        onError,
        onComplete: () => {
          setShowSchedule(false);
          setScheduleForm(initialScheduleForm());
          onSchedulesChanged();
        },
      });
    },
    [scheduleForm, onError, onSchedulesChanged],
  );

  const handleDeleteSchedule = useCallback(
    async (scheduleId: string) => {
      await handleDeleteScheduleSubmit({
        scheduleId,
        setSectionBusy,
        onError,
        onComplete: onSchedulesChanged,
      });
    },
    [onError, onSchedulesChanged],
  );

  // Close modals on Escape.
  useEffect(() => {
    if (!showGenerate && !showSchedule) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setShowGenerate(false);
        setShowSchedule(false);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showGenerate, showSchedule]);

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
          label="Total reports"
          value={totalReports}
          status={totalReports > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
        <KpiCard
          label="Frameworks"
          value={distinctFrameworks}
          status={distinctFrameworks > 0 ? 'up' : 'neutral'}
          accent="cyan"
        />
        <KpiCard
          label="Last 24h"
          value={reportsLast24h}
          status={reportsLast24h > 0 ? 'up' : 'neutral'}
          accent="green"
        />
      </motion.div>

      <section className="dash-section">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'baseline',
            gap: 8,
          }}
        >
          <div>
            <span className="dash-eyebrow">Compliance Reports</span>
            <h2 className="dash-section-title">Evidence Packages</h2>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => setShowGenerate(true)}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={isBusy}
            >
              + Generate report
            </motion.button>
            <motion.button
              type="button"
              className="sw-button sw-button-secondary"
              onClick={() => setShowSchedule(true)}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={isBusy}
            >
              + Schedule report
            </motion.button>
          </div>
        </div>

        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Generate evidence packages for SOC2 / ISO 27001 / HIPAA / PCI / GDPR
          auditors. Reports assemble real DB counts (audit log / anomaly events
          / active users) into a downloadable artifact.
        </p>

        {/* Tab strip */}
        <div
          role="tablist"
          aria-label="Compliance views"
          style={{
            display: 'flex',
            gap: 4,
            borderBottom: '1px solid var(--border)',
            marginTop: 12,
            marginBottom: 12,
          }}
        >
          {(
            [
              { id: 'reports' as const, label: `Reports (${reports.length})` },
              {
                id: 'schedules' as const,
                label: `Schedules (${schedules.length})`,
              },
            ]
          ).map((t) => (
            <motion.button
              key={t.id}
              type="button"
              role="tab"
              aria-selected={tab === t.id}
              onClick={() => setTab(t.id)}
              className={`sw-button sw-button-secondary`}
              style={{
                borderBottom:
                  tab === t.id ? '2px solid var(--accent)' : '2px solid transparent',
                borderRadius: 0,
                opacity: tab === t.id ? 1 : 0.65,
              }}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
            >
              {t.label}
            </motion.button>
          ))}
        </div>

        {tab === 'reports' ? (
          <div className="threat-card-list">
            {sortedReports.length === 0 ? (
              <div
                className="threat-card"
                style={{
                  textAlign: 'center',
                  padding: 24,
                  color: 'var(--text-muted)',
                }}
              >
                <strong style={{ color: 'var(--text)' }}>
                  No compliance reports yet
                </strong>
                <p style={{ margin: '8px 0 16px', fontSize: 12 }}>
                  Click <strong>+ Generate report</strong> to assemble a fresh
                  evidence package for an auditor.
                </p>
              </div>
            ) : (
              sortedReports.map((r) => (
                <ComplianceReportCard
                  key={r.id}
                  report={r}
                  busy={isBusy}
                  onDownload={(id) =>
                    void downloadReport(id, sessionToken, onError)
                  }
                />
              ))
            )}
          </div>
        ) : (
          <ComplianceScheduleList
            schedules={schedules}
            busy={isBusy}
            onDelete={(id) => void handleDeleteSchedule(id)}
          />
        )}
      </section>

      {showGenerate ? (
        <Modal title="Generate compliance report" onClose={() => setShowGenerate(false)}>
          <GenerateReportFormView
            generateForm={generateForm}
            setGenerateForm={setGenerateForm}
            busy={isBusy}
            onSubmit={handleGenerate}
            onClose={() => setShowGenerate(false)}
          />
        </Modal>
      ) : null}

      {showSchedule ? (
        <Modal title="Schedule compliance report" onClose={() => setShowSchedule(false)}>
          <ScheduleReportFormView
            scheduleForm={scheduleForm}
            setScheduleForm={setScheduleForm}
            busy={isBusy}
            onSubmit={handleSchedule}
            onClose={() => setShowSchedule(false)}
          />
        </Modal>
      ) : null}
    </>
  );
}
