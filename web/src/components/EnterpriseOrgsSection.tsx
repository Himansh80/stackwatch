import { FormEvent, useMemo, useState } from 'react';
import { ApiError, api } from '../lib/api';
import KpiCard from './shared/KpiCard';
import OrgSettingsForm, { OrgSettings } from './shared/OrgSettingsForm';
import {
  CreateOrgForm,
  createOrgFormInitial,
} from './EnterpriseOrgsSectionHelpers';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../lib/motion';

/**
 * EnterpriseOrgsSection — Tier 9.6 (Phase 6) UI surface for the
 * Enterprise Tenants + Org Settings tab on EnterprisePage.
 *
 * Mirrors the SsoSection / ScimSection / RbacSection / AuditSection /
 * ComplianceSection pattern: a self-contained tab body that takes
 * loaded data + callbacks as props. The parent (EnterprisePage)
 * owns the data-fetch lifecycle; this section only handles
 * presentation + the create-sub-org / update-settings mutations
 * it owns.
 *
 * Layout:
 *   - 3 KpiCards: total orgs / sub-orgs / users across orgs
 *   - "+ Create sub-org" button (top right) → CreateOrgForm
 *   - Tree view of org hierarchy: parents shown first, children
 *     indented under their parent. Click an org to expand its
 *     settings panel (OrgSettingsForm).
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring). No
 * new variants. Tokens: --surface / --border / --accent / --green
 * / --red via existing .threat-card / .empty-state-cta / .form-input
 * classes — no new CSS.
 */

export interface EnterpriseOrgRow {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  parent_org_id?: string | null;
  settings?: Record<string, unknown> | null;
  user_count?: number;
  child_count?: number;
  created_at: string;
  updated_at: string;
}

interface EnterpriseOrgsSectionProps {
  orgs: EnterpriseOrgRow[];
  busy?: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

interface OrgNode {
  org: EnterpriseOrgRow;
  children: OrgNode[];
  depth: number;
}

// buildTree walks the flat orgs list and returns a forest of OrgNodes
// keyed by parent_org_id. Orphan orgs (parent_org_id points to a row
// we don't have — shouldn't happen since the API filters by
// tenant_id) get attached to the top level as a defensive fallback.
function buildTree(orgs: EnterpriseOrgRow[]): OrgNode[] {
  const byID = new Map<string, OrgNode>();
  for (const o of orgs) byID.set(o.id, { org: o, children: [], depth: 0 });
  const roots: OrgNode[] = [];
  for (const o of orgs) {
    const node = byID.get(o.id)!;
    const parent = o.parent_org_id ? byID.get(o.parent_org_id) : null;
    if (parent) {
      node.depth = parent.depth + 1;
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

// flattenTree returns a DFS-ordered list with depth info so the JSX
// can render with an indent multiplier per row.
function flattenTree(roots: OrgNode[]): OrgNode[] {
  const out: OrgNode[] = [];
  const walk = (n: OrgNode) => {
    out.push(n);
    for (const c of n.children) walk(c);
  };
  for (const r of roots) walk(r);
  return out;
}

export default function EnterpriseOrgsSection({
  orgs,
  busy = false,
  onError,
  onChanged,
}: EnterpriseOrgsSectionProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState(createOrgFormInitial);
  const [expandedID, setExpandedID] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string>('');

  const { tree, flat } = useMemo(() => {
    const t = buildTree(orgs);
    return { tree: t, flat: flattenTree(t) };
  }, [orgs]);

  const totalOrgs = orgs.length;
  const totalSubOrgs = orgs.filter((o) => o.parent_org_id).length;
  const totalUsers = orgs.reduce((sum, o) => sum + (o.user_count ?? 0), 0);

  const handleCreate = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSectionBusy(true);
    setSaveError('');
    try {
      const body: Record<string, unknown> = {
        name: form.name.trim(),
        slug: form.slug.trim().toLowerCase(),
      };
      const pid = form.parent_orgId.trim();
      if (pid) body.parent_org_id = pid;
      await api('POST', '/api/v1/enterprise/orgs', body);
      setForm(createOrgFormInitial());
      setShowCreate(false);
      onChanged();
    } catch (cause) {
      const msg =
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setSaveError(msg);
      onError(msg);
    } finally {
      setSectionBusy(false);
    }
  };

  const handleSaveSettings = async (
    orgID: string,
    newSettings: OrgSettings,
  ) => {
    setSectionBusy(true);
    setSaveError('');
    try {
      await api('PATCH', `/api/v1/enterprise/orgs/${orgID}/settings`, {
        settings: newSettings,
      });
      onChanged();
    } catch (cause) {
      const msg =
        cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setSaveError(msg);
      onError(msg);
    } finally {
      setSectionBusy(false);
    }
  };

  const expandedOrg = expandedID ? orgs.find((o) => o.id === expandedID) : null;

  return (
    <div>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
      >
        <KpiCard label="Total orgs" value={totalOrgs} accent="indigo"
          status={totalOrgs > 0 ? 'up' : 'neutral'} />
        <KpiCard label="Sub-orgs" value={totalSubOrgs} accent="cyan"
          status={totalSubOrgs > 0 ? 'up' : 'neutral'} />
        <KpiCard label="Users across orgs" value={totalUsers} accent="green"
          status={totalUsers > 0 ? 'up' : 'neutral'} />
      </motion.div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 12 }}>
        <motion.button
          type="button"
          className="empty-state-cta"
          onClick={() => {
            setShowCreate((v) => !v);
            setSaveError('');
          }}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
          disabled={isBusy}
        >
          {showCreate ? 'Close' : '+ Create sub-org'}
        </motion.button>
      </div>

      {showCreate ? (
        <CreateOrgForm
          form={form}
          setForm={setForm}
          orgs={orgs}
          busy={isBusy}
          errorMessage={saveError}
          onSubmit={(e) => void handleCreate(e)}
        />
      ) : null}

      <div style={{ marginTop: 16 }}>
        {flat.length === 0 ? (
          <div className="threat-card"
            style={{ textAlign: 'center', padding: 24, color: 'var(--muted)' }}>
            No orgs yet — click <strong>Create sub-org</strong> above to add
            your first org.
          </div>
        ) : (
          flat.map((node) => {
            const indent = node.depth * 16;
            const isExpanded = expandedID === node.org.id;
            return (
              <div key={node.org.id}>
                <div
                  className="threat-card"
                  style={{ marginLeft: indent, marginTop: 8, cursor: 'pointer' }}
                  onClick={() => {
                    setExpandedID(isExpanded ? null : node.org.id);
                    setSaveError('');
                  }}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      setExpandedID(isExpanded ? null : node.org.id);
                    }
                  }}
                  aria-expanded={isExpanded}
                  aria-label={`Toggle settings for ${node.org.name}`}
                >
                  <div className="threat-card-top">
                    <span className="dash-sev dash-sev-low"
                      style={{ background: 'var(--accent-soft)' }}>
                      {node.depth === 0 ? 'Top-level' : `Child of depth ${node.depth}`}
                    </span>
                    <strong className="threat-card-type">{node.org.name}</strong>
                    <span style={{ marginLeft: 8, color: 'var(--muted)', fontSize: 12 }}>
                      {node.org.slug}
                    </span>
                    <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--muted)' }}>
                      {node.org.user_count ?? 0} users · {node.org.child_count ?? 0} sub-orgs
                      <span style={{ marginLeft: 8 }}>{isExpanded ? '▾' : '▸'}</span>
                    </span>
                  </div>
                </div>
                {isExpanded && expandedOrg ? (
                  <div style={{ marginLeft: indent + 16, marginTop: 4 }}>
                    <OrgSettingsForm
                      org={{
                        id: expandedOrg.id,
                        name: expandedOrg.name,
                        settings: (expandedOrg.settings as OrgSettings | null) ?? null,
                      }}
                      onSave={(s) => void handleSaveSettings(expandedOrg.id, s)}
                      onCancel={() => setExpandedID(null)}
                      busy={isBusy}
                      errorMessage={saveError}
                    />
                  </div>
                ) : null}
              </div>
            );
          })
        )}
        {tree.length > 0 ? (
          <div style={{ marginTop: 8, fontSize: 11, color: 'var(--muted)', textAlign: 'right' }}>
            {tree.length} top-level org{tree.length === 1 ? '' : 's'}
            {totalSubOrgs > 0 ? ` · ${totalSubOrgs} sub-org${totalSubOrgs === 1 ? '' : 's'}` : ''}
          </div>
        ) : null}
      </div>
    </div>
  );
}
