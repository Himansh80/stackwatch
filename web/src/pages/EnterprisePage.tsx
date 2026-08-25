import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import SsoSection from '../components/SsoSection';
import type { SsoProviderRow } from '../components/shared/SsoProviderCard';
import ScimSection from '../components/ScimSection';
import type { ScimTokenRow } from '../components/shared/ScimTokenTable';
import RbacSection from '../components/RbacSection';
import type { RbacRoleRow } from '../components/RbacSection';
import type { RbacAssignmentRow } from '../components/RbacAssignmentsPanel';
import AuditSection from '../components/AuditSection';
import type { AuditArchiveRow } from '../components/shared/AuditArchiveCard';
import ComplianceSection from '../components/ComplianceSection';
import type {
  ComplianceScheduleRow,
} from '../components/ComplianceSection';
import type { ComplianceReportRow } from '../components/shared/ComplianceReportCard';
import EnterpriseOrgsSection from '../components/EnterpriseOrgsSection';
import type { EnterpriseOrgRow } from '../components/EnterpriseOrgsSection';
import KpiCard from '../components/shared/KpiCard';
import { motion, buttonSpring, kpiStagger, pageEnter, useReducedMotion } from '../lib/motion';

/**
 * EnterprisePage — Tier 9 (D9) Security & Enterprise surface at /enterprise.
 *
 * Phase 6 — THE FINAL PHASE of Tier 9. The page is now a true 6-tab
 * dispatcher that wires data loads once on mount and renders one
 * of the six extracted sections per active tab:
 *
 *   - SSO         → SsoSection              (Phase 1)
 *   - SCIM        → ScimSection             (Phase 2)
 *   - RBAC        → RbacSection             (Phase 3)
 *   - Audit       → AuditSection            (Phase 4)
 *   - Compliance  → ComplianceSection       (Phase 5)
 *   - Orgs        → EnterpriseOrgsSection   (Phase 6 — NEW)
 *
 * Header layout:
 *   - Title: "Enterprise"
 *   - Subtitle: "SSO + SCIM + RBAC + audit + compliance + org hierarchy"
 *   - Right side: Export button (placeholder — wires to a future
 *     /enterprise/export endpoint that bundles all surfaces into a
 *     single audit-trail JSON; for now it's a no-op toast).
 *   - KPI strip across the top: 6 top-level counts (active SSO
 *     providers / SCIM tokens / RBAC roles / audit archives /
 *     compliance reports / orgs) using kpiStagger for entrance.
 *
 * Motion: pageEnter on the page; kpiStagger on the KPI strip;
 * buttonSpring on the Export CTA. Reuses existing tokens.
 *
 * This is the LAST page added to Tier 9. After this ships, the
 * Tier 9 surface is COMPLETE — 32 routes, 13 DB tables, and 6
 * UI surfaces wired into a single dashboard.
 */

type Tab = 'sso' | 'scim' | 'rbac' | 'audit' | 'compliance' | 'orgs';

const TAB_LABELS: Record<Tab, string> = {
  sso: 'SSO',
  scim: 'SCIM',
  rbac: 'RBAC',
  audit: 'Audit',
  compliance: 'Compliance',
  orgs: 'Orgs',
};

interface SsoConnectionsResp {
  connections?: { id: string; provider_id: string; provider_name?: string; subject: string; created_at: string; last_used_at?: string | null }[];
}
interface SsoListResp { providers?: SsoProviderRow[]; total?: number; }
interface ScimListResp { tokens?: ScimTokenRow[]; total?: number; }
interface ScimSyncLogResp { entries?: { status?: string }[]; total?: number; }
interface RbacListResp { roles?: RbacRoleRow[]; total?: number; }
interface AuditListResp { archives?: AuditArchiveRow[]; total?: number; }
interface ComplianceReportsResp { reports?: ComplianceReportRow[]; total?: number; }
interface ComplianceSchedulesResp { schedules?: ComplianceScheduleRow[]; total?: number; }
interface OrgsListResp { orgs?: EnterpriseOrgRow[]; total?: number; }

export default function EnterprisePage() {
  const logout = useLogout();
  const reduce = useReducedMotion();
  const [tab, setTab] = useState<Tab>('sso');
  const [error, setError] = useState('');
  const [exportBusy, setExportBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const sessionToken = getToken();

  // --- Shared data state — loaded once on mount, fed to whichever
  // section owns the active tab. The other tabs' sections won't
  // render but the data is still in memory so switching tabs is
  // instant. Same pattern as IntelligencePage (Tier 8 Phase 5).
  const [ssoProviders, setSsoProviders] = useState<SsoProviderRow[]>([]);
  const [ssoConnections, setSsoConnections] = useState<SsoConnectionsResp['connections']>([]);
  const [scimTokens, setScimTokens] = useState<ScimTokenRow[]>([]);
  const [scimRecentErrors, setScimRecentErrors] = useState<number>(0);
  const [rbacRoles, setRbacRoles] = useState<RbacRoleRow[]>([]);
  const [rbacAssignments] = useState<RbacAssignmentRow[]>([]);
  const [auditArchives, setAuditArchives] = useState<AuditArchiveRow[]>([]);
  const [complianceReports, setComplianceReports] = useState<ComplianceReportRow[]>([]);
  const [complianceSchedules, setComplianceSchedules] = useState<ComplianceScheduleRow[]>([]);
  const [orgs, setOrgs] = useState<EnterpriseOrgRow[]>([]);

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view Enterprise.');
      return false;
    }
    return true;
  };

  // Single load-all call. Mirrors IntelligencePage (Tier 8 Phase 5).
  // We fetch every section's data up front so tab switches are
  // instant — one bigger initial response vs 6 small lazy calls.
  const loadData = useCallback(async () => {
    if (!requireAuth()) return;
    setBusy(true);
    try {
      const [
        sso, conn, scimTokensResp, scimLog,
        rbacR, audit, cReports, cSchedules, orgsResp,
      ] = await Promise.all([
        api<SsoListResp>('GET', '/api/v1/enterprise/sso/providers'),
        api<SsoConnectionsResp>('GET', '/api/v1/enterprise/sso/connections'),
        api<ScimListResp>('GET', '/api/v1/enterprise/scim/tokens'),
        api<ScimSyncLogResp>('GET', '/api/v1/enterprise/scim/sync-log?limit=50'),
        api<RbacListResp>('GET', '/api/v1/enterprise/rbac/roles'),
        api<AuditListResp>('GET', '/api/v1/enterprise/audit/archives'),
        api<ComplianceReportsResp>('GET', '/api/v1/enterprise/compliance/reports?limit=100'),
        api<ComplianceSchedulesResp>('GET', '/api/v1/enterprise/compliance/schedules'),
        api<OrgsListResp>('GET', '/api/v1/enterprise/orgs'),
      ]);
      setSsoProviders(sso.providers || []);
      setSsoConnections(conn.connections || []);
      setScimTokens(scimTokensResp.tokens || []);
      setScimRecentErrors(
        (scimLog.entries || []).filter((e) => e.status === 'error').length,
      );
      setRbacRoles(rbacR.roles || []);
      // No GET /rbac/assignments endpoint exists — the RbacSection
      // already handles an empty assignments list (its KPI then
      // shows 0 users with custom assignments, which is the
      // truthful state until a list endpoint is added).
      // (rbacAssignments is set to [] above and never reassigned.)
      setAuditArchives(audit.archives || []);
      setComplianceReports(cReports.reports || []);
      setComplianceSchedules(cSchedules.schedules || []);
      setOrgs(orgsResp.orgs || []);
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    setError('');
    void loadData();
  }, [loadData]);

  // Export button handler — Phase 6 placeholder. The future
  // /enterprise/export endpoint (Tier 9.x post-Phase 6) will bundle
  // all 6 surfaces into a single audit-trail JSON. For now we
  // surface a friendly "coming soon" message via a window.confirm.
  const onExport = useCallback(async () => {
    if (!requireAuth()) return;
    setExportBusy(true);
    try {
      // Placeholder — wires to a future /enterprise/export endpoint.
      window.alert(
        'Enterprise export is a Tier 9.x post-Phase 6 feature. ' +
          'For now, use the per-surface Export buttons inside each tab.',
      );
    } finally {
      setTimeout(() => setExportBusy(false), 600);
    }
  }, []);

  // Header KPI strip — one count per Tier 9 surface, computed from
  // the loaded state. These are the page-level "fleet enterprise
  // posture at a glance" KPIs the spec calls out as User Story 6.2.
  const headerKpis = useMemo(() => {
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    return {
      activeSso: ssoProviders.filter((p) => p.enabled).length,
      scimTokens: scimTokens.length,
      rbacRoles: rbacRoles.length,
      auditArchives: auditArchives.length,
      complianceReportsToday: (complianceReports || []).filter(
        (r) => new Date(r.created_at).getTime() >= oneDayAgo,
      ).length,
      orgs: orgs.length,
    };
  }, [ssoProviders, scimTokens, rbacRoles, auditArchives, complianceReports, orgs]);

  return (
    <div className="dash-app">
      <AppSidebar
        active="enterprise"
        onLogout={logout}
        show={[
          'dashboard', 'billing', 'profile', 'settings',
          'proxmox', 'truenas',
          'incidents', 'notebooks', 'intelligence', 'enterprise',
        ]}
      />
      <main className="dash-main">
        <header className="dash-topbar">
          <div className="dash-greeting">
            <span className="dash-greeting-eyebrow">Enterprise</span>
            <div className="dash-greeting-row">
              <strong className="dash-greeting-text">
                SSO + SCIM + RBAC + audit + compliance + org hierarchy
              </strong>
              <span className="dash-greeting-clock">
                <span className="dash-greeting-clock-time">
                  {headerKpis.activeSso} SSO · {headerKpis.scimTokens} SCIM ·
                  {headerKpis.rbacRoles} roles · {headerKpis.auditArchives} archives ·
                  {headerKpis.orgs} orgs
                </span>
              </span>
            </div>
          </div>
          <div className="dash-top-actions">
            <motion.button
              type="button"
              className="empty-state-cta"
              onClick={() => void onExport()}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy || exportBusy}
              title="Enterprise export (placeholder — future Tier 9.x endpoint)"
            >
              {exportBusy ? 'Exporting…' : '⤓ Export'}
            </motion.button>
          </div>
        </header>

        <motion.div
          className="dash-page"
          initial="hidden"
          animate="show"
          variants={pageEnter}
        >
          {error ? (
            <div className="dash-error" role="alert">{error}</div>
          ) : null}

          {/* Top KPI strip — fleet enterprise posture at a glance. */}
          <motion.div
            className="dash-metric-strip"
            initial="hidden"
            animate="show"
            variants={kpiStagger}
          >
            <KpiCard label="Active SSO providers" value={headerKpis.activeSso}
              status={headerKpis.activeSso > 0 ? 'up' : 'neutral'}
              accent={headerKpis.activeSso > 0 ? 'green' : 'indigo'} />
            <KpiCard label="SCIM tokens" value={headerKpis.scimTokens}
              status={headerKpis.scimTokens > 0 ? 'up' : 'neutral'}
              accent={headerKpis.scimTokens > 0 ? 'cyan' : 'indigo'} />
            <KpiCard label="RBAC roles" value={headerKpis.rbacRoles}
              status={headerKpis.rbacRoles > 0 ? 'up' : 'neutral'}
              accent={headerKpis.rbacRoles > 0 ? 'amber' : 'indigo'} />
            <KpiCard label="Audit archives" value={headerKpis.auditArchives}
              status={headerKpis.auditArchives > 0 ? 'up' : 'neutral'}
              accent={headerKpis.auditArchives > 0 ? 'cyan' : 'indigo'} />
            <KpiCard label="Compliance reports (24h)" value={headerKpis.complianceReportsToday}
              status={headerKpis.complianceReportsToday > 0 ? 'up' : 'neutral'}
              accent={headerKpis.complianceReportsToday > 0 ? 'amber' : 'indigo'} />
            <KpiCard label="Orgs" value={headerKpis.orgs}
              status={headerKpis.orgs > 0 ? 'up' : 'neutral'}
              accent={headerKpis.orgs > 0 ? 'green' : 'indigo'} />
          </motion.div>

          {/* 6-tab dispatcher (SSO / SCIM / RBAC / Audit / Compliance / Orgs). */}
          <div className="logs-tabs" role="tablist" style={{ marginTop: 16 }}>
            {(['sso', 'scim', 'rbac', 'audit', 'compliance', 'orgs'] as Tab[]).map(
              (t) => (
                <button
                  key={t}
                  role="tab"
                  type="button"
                  aria-selected={tab === t}
                  className={`logs-tab ${tab === t ? 'logs-tab-active' : ''}`}
                  onClick={() => setTab(t)}
                >
                  {TAB_LABELS[t]}
                </button>
              ),
            )}
          </div>

          {tab === 'sso' ? (
            <SsoSection
              providers={ssoProviders}
              connections={ssoConnections || []}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
            />
          ) : null}

          {tab === 'scim' ? (
            <ScimSection
              tokens={scimTokens}
              usersCount={0}
              recentErrors={scimRecentErrors}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
            />
          ) : null}

          {tab === 'rbac' ? (
            <RbacSection
              roles={rbacRoles}
              assignments={rbacAssignments}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
            />
          ) : null}

          {tab === 'audit' ? (
            <AuditSection
              archives={auditArchives}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
              sessionToken={sessionToken}
            />
          ) : null}

          {tab === 'compliance' ? (
            <ComplianceSection
              reports={complianceReports}
              schedules={complianceSchedules}
              busy={busy}
              onError={setError}
              onReportsChanged={() => void loadData()}
              onSchedulesChanged={() => void loadData()}
              sessionToken={sessionToken}
            />
          ) : null}

          {tab === 'orgs' ? (
            <EnterpriseOrgsSection
              orgs={orgs}
              busy={busy}
              onError={setError}
              onChanged={() => void loadData()}
            />
          ) : null}
        </motion.div>
      </main>
    </div>
  );
}
