import { useCallback, useMemo, useState } from 'react';
import KpiCard from './shared/KpiCard';
import RbacRoleEditor, { RbacRoleDraft } from './shared/RbacRoleEditor';
import RbacAssignmentsPanel, {
  RbacAssignmentRow,
  RbacRoleRef,
} from './RbacAssignmentsPanel';
import { ApiError, api } from '../lib/api';
import { motion, buttonSpring, kpiStagger, useReducedMotion } from '../lib/motion';

/**
 * RbacSection — Tier 9.3 (Phase 3) UI surface for the Roles &
 * Permissions page section. Mirrors the SsoSection / ScimSection
 * pattern so the future EnterprisePage (Phase 6) can compose this
 * in without going over the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: total roles / custom roles / users with custom
 *     roles (using kpiStagger for entrance).
 *   - Tabbed role list (Built-in | Custom). Each tab is a button
 *     row; clicking switches the filter.
 *   - Each role row: name + description + permission count + edit
 *     button (custom only) + delete button (custom only).
 *   - RbacAssignmentsPanel handles user↔role binding UI (kept in
 *     a separate file to keep this file under the 400-LOC cap).
 *
 * Motion: reuses existing exports (kpiStagger, buttonSpring) — no
 * new variants.
 */

export interface RbacRoleRow {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  permissions: string[];
  is_builtin: boolean;
  created_at: string;
}

interface RbacSectionProps {
  roles: RbacRoleRow[];
  assignments: RbacAssignmentRow[];
  busy?: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

type Tab = 'builtin' | 'custom';

function relativeTime(iso: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Math.max(0, Date.now() - t);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

export default function RbacSection({
  roles,
  assignments,
  busy = false,
  onError,
  onChanged,
}: RbacSectionProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  const [tab, setTab] = useState<Tab>('builtin');
  const [editing, setEditing] = useState<RbacRoleRow | null>(null);
  const [creating, setCreating] = useState(false);

  const builtins = useMemo(() => roles.filter((r) => r.is_builtin), [roles]);
  const customs = useMemo(() => roles.filter((r) => !r.is_builtin), [roles]);
  const visible = tab === 'builtin' ? builtins : customs;

  const usersWithAssignments = useMemo(() => {
    const set = new Set<string>();
    assignments.forEach((a) => set.add(a.user_id));
    return set.size;
  }, [assignments]);

  const roleRefs = useMemo<RbacRoleRef[]>(
    () => roles.map((r) => ({ id: r.id, name: r.name, is_builtin: r.is_builtin })),
    [roles],
  );

  const handleDeleteRole = useCallback(
    async (role: RbacRoleRow) => {
      if (role.is_builtin) return;
      if (
        !window.confirm(
          `Delete custom role "${role.name}"? All user assignments to this role will be revoked. (cannot be undone)`,
        )
      ) {
        return;
      }
      setSectionBusy(true);
      try {
        await api('DELETE', `/api/v1/enterprise/rbac/roles/${role.id}`);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  const handleSaveRole = useCallback(
    async (draft: RbacRoleDraft) => {
      const isEdit = editing != null && !creating;
      setSectionBusy(true);
      try {
        if (isEdit) {
          await api('PATCH', `/api/v1/enterprise/rbac/roles/${editing.id}`, {
            name: draft.name,
            description: draft.description,
            permissions: draft.permissions,
          });
        } else {
          await api('POST', '/api/v1/enterprise/rbac/roles', {
            name: draft.name,
            description: draft.description,
            permissions: draft.permissions,
          });
        }
        setEditing(null);
        setCreating(false);
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [editing, creating, onChanged, onError],
  );

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
          label="Total roles"
          value={roles.length}
          status={roles.length > 0 ? 'up' : 'neutral'}
          accent="indigo"
        />
        <KpiCard
          label="Custom roles"
          value={customs.length}
          status={customs.length > 0 ? 'up' : 'neutral'}
          accent={customs.length > 0 ? 'green' : 'cyan'}
        />
        <KpiCard
          label="Users with custom roles"
          value={usersWithAssignments}
          status={usersWithAssignments > 0 ? 'up' : 'neutral'}
          accent="cyan"
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Roles &amp; Permissions</span>
        <h2 className="dash-section-title">Roles</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Built-in roles (Admin / Operator / Viewer / Billing) cannot
          be modified. Create a custom role to grant specific
          permissions to a subset of users.
        </p>

        <div
          style={{
            display: 'flex',
            gap: 8,
            marginTop: 8,
            marginBottom: 12,
            borderBottom: '1px solid var(--border)',
          }}
        >
          {(['builtin', 'custom'] as Tab[]).map((t) => {
            const count = t === 'builtin' ? builtins.length : customs.length;
            const active = tab === t;
            return (
              <motion.button
                key={t}
                type="button"
                onClick={() => setTab(t)}
                whileHover={reduce ? undefined : buttonSpring.whileHover}
                whileTap={reduce ? undefined : buttonSpring.whileTap}
                transition={buttonSpring.transition}
                style={{
                  background: 'transparent',
                  border: 'none',
                  borderBottom: active ? '2px solid var(--accent)' : '2px solid transparent',
                  color: active ? 'var(--text)' : 'var(--text-muted)',
                  padding: '6px 12px',
                  fontSize: 13,
                  cursor: 'pointer',
                  textTransform: 'capitalize',
                  fontWeight: active ? 600 : 400,
                }}
                aria-pressed={active}
              >
                {t} <span style={{ opacity: 0.7 }}>({count})</span>
              </motion.button>
            );
          })}
          <motion.button
            type="button"
            className="empty-state-cta"
            style={{ marginLeft: 'auto', marginBottom: 8 }}
            onClick={() => {
              setCreating(true);
              setEditing(null);
            }}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={isBusy}
          >
            + New custom role
          </motion.button>
        </div>

        <div className="threat-card-list">
          {visible.length === 0 ? (
            <div
              className="threat-card"
              style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}
            >
              <strong style={{ color: 'var(--text)' }}>
                {tab === 'builtin'
                  ? 'No built-in roles seeded yet'
                  : 'No custom roles yet'}
              </strong>
              <p style={{ margin: '8px 0 0', fontSize: 12 }}>
                {tab === 'builtin'
                  ? 'Built-in roles lazy-seed on first GET. Try refreshing the page.'
                  : 'Click "+ New custom role" to create one.'}
              </p>
            </div>
          ) : (
            visible.map((role) => (
              <article key={role.id} className="threat-card">
                <div className="threat-card-top">
                  <span
                    className={`dash-sev ${role.is_builtin ? 'dash-sev-low' : 'dash-sev-medium'}`}
                  >
                    {role.is_builtin ? 'BUILT-IN' : 'CUSTOM'}
                  </span>
                  <strong className="threat-card-type">{role.name}</strong>
                  <span
                    className="dash-status dash-status-up"
                    title={`${role.permissions.length} permissions`}
                  >
                    <span className="dash-status-dot" aria-hidden="true" />
                    {role.permissions.length} perm{role.permissions.length === 1 ? '' : 's'}
                  </span>
                </div>
                {role.description ? (
                  <p className="threat-card-desc">{role.description}</p>
                ) : null}
                <div className="threat-card-meta">
                  {!role.is_builtin ? (
                    <>
                      <button
                        type="button"
                        className="threat-card-resolve-btn"
                        onClick={() => {
                          setEditing(role);
                          setCreating(false);
                        }}
                        aria-label={`Edit custom role ${role.name}`}
                        disabled={isBusy}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="threat-card-resolve-btn"
                        onClick={() => void handleDeleteRole(role)}
                        aria-label={`Delete custom role ${role.name}`}
                        style={{ color: 'var(--red)' }}
                        disabled={isBusy}
                      >
                        Delete
                      </button>
                    </>
                  ) : null}
                  <span
                    style={{
                      marginLeft: 'auto',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                    title={role.created_at}
                  >
                    created {relativeTime(role.created_at)}
                  </span>
                </div>
              </article>
            ))
          )}
        </div>
      </section>

      <section className="dash-section">
        <span className="dash-eyebrow">User role assignments</span>
        <h2 className="dash-section-title">Assign roles to users</h2>
        <RbacAssignmentsPanel
          assignments={assignments}
          roles={roleRefs}
          busy={isBusy}
          onError={onError}
          onChanged={onChanged}
        />
      </section>

      {(creating || editing) ? (
        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
          <div className="slow-query-explain-modal-head">
            <strong>{creating ? 'New custom role' : `Edit ${editing?.name}`}</strong>
            <button
              type="button"
              className="slow-query-explain-close"
              onClick={() => {
                setCreating(false);
                setEditing(null);
              }}
              aria-label="Close"
              disabled={isBusy}
            >
              ✕
            </button>
          </div>
          <div style={{ padding: 16 }}>
            <RbacRoleEditor
              role={editing ?? undefined}
              onSave={(d) => void handleSaveRole(d)}
              onCancel={() => {
                setCreating(false);
                setEditing(null);
              }}
              busy={isBusy}
            />
          </div>
        </div>
      ) : null}
    </>
  );
}
