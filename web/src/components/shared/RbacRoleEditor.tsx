import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * RbacRoleEditor — Tier 9.3 (Phase 3) Datadog-style editor for a
 * single RBAC role returned by GET /api/v1/enterprise/rbac/roles.
 *
 * Mirrors the rbacRoleRow JSON shape (id, name, description,
 * permissions, is_builtin). Used both for creating custom roles
 * (id is empty on first render) and for editing existing custom
 * roles. Built-in roles are rendered READ-ONLY — the spec is
 * strict that built-ins (Admin / Operator / Viewer / Billing) can
 * never be modified or deleted by tenant admins.
 *
 * Layout (Datadog-style form):
 *   - Banner: "Built-in role — read-only" when is_builtin=true
 *   - Name input (text, required, max 64 chars)
 *   - Description input (textarea, max 256 chars)
 *   - Permissions multi-select grouped by category:
 *       Servers / Dashboards / Alerts / Incidents / SSO / SCIM
 *       / RBAC / Audit / Compliance / Org
 *     Each category header is a small caps eyebrow; each
 *     permission is a checkbox in `<resource>:<verb>` format.
 *   - Save / Cancel buttons
 *
 * Props:
 *   role        — the role being edited (id empty = new role)
 *   onSave      — called with the validated payload
 *   onCancel    — close without saving
 *   busy        — disable inputs during in-flight save
 *
 * The permissions list is hard-coded to mirror the backend's
 * builtinPermissions allowlist (handlers_rbac_types.go). If the
 * backend adds a new permission, this list must be updated — same
 * convention as SsoProviderForm's scope list.
 */

// Permission groups — order matters for stable UI rendering.
// Each group: { label, permissions: ["resource:verb", ...] }
const PERMISSION_GROUPS: { label: string; permissions: string[] }[] = [
  {
    label: 'Servers',
    permissions: ['servers:read', 'servers:write'],
  },
  {
    label: 'Dashboards',
    permissions: ['dashboards:read', 'dashboards:write'],
  },
  {
    label: 'Alerts',
    permissions: ['alerts:read', 'alerts:write'],
  },
  {
    label: 'Incidents',
    permissions: ['incidents:read', 'incidents:write'],
  },
  {
    label: 'SSO',
    permissions: ['sso:read', 'sso:write'],
  },
  {
    label: 'SCIM',
    permissions: ['scim:read', 'scim:write'],
  },
  {
    label: 'RBAC',
    permissions: ['rbac:read', 'rbac:write'],
  },
  {
    label: 'Audit',
    permissions: ['audit:read', 'audit:write'],
  },
  {
    label: 'Compliance',
    permissions: ['compliance:read', 'compliance:write'],
  },
  {
    label: 'Org',
    permissions: ['org:read', 'org:write'],
  },
];

export interface RbacRoleDraft {
  name: string;
  description: string;
  permissions: string[];
}

interface RbacRoleEditorProps {
  role?: {
    id?: string;
    name: string;
    description?: string;
    permissions: string[];
    is_builtin?: boolean;
  };
  onSave: (draft: RbacRoleDraft) => void;
  onCancel?: () => void;
  busy?: boolean;
}

function initialFromRole(role?: RbacRoleEditorProps['role']): RbacRoleDraft {
  return {
    name: role?.name ?? '',
    description: role?.description ?? '',
    permissions: role?.permissions ? [...role.permissions] : [],
  };
}

export default function RbacRoleEditor({
  role,
  onSave,
  onCancel,
  busy = false,
}: RbacRoleEditorProps) {
  const reduce = useReducedMotion();
  const [draft, setDraft] = useState<RbacRoleDraft>(() => initialFromRole(role));

  // Re-sync when the role prop changes (e.g., user clicks "Edit"
  // on a different role while this editor is still mounted).
  useEffect(() => {
    setDraft(initialFromRole(role));
  }, [role]);

  const isReadOnly = !!role?.is_builtin;
  const isNew = !role?.id;

  const selectedCount = draft.permissions.length;
  const totalCount = useMemo(
    () => PERMISSION_GROUPS.reduce((sum, g) => sum + g.permissions.length, 0),
    [],
  );

  const togglePermission = useCallback((perm: string) => {
    setDraft((d) => ({
      ...d,
      permissions: d.permissions.includes(perm)
        ? d.permissions.filter((p) => p !== perm)
        : [...d.permissions, perm],
    }));
  }, []);

  const handleSubmit = useCallback(
    (e: FormEvent) => {
      e.preventDefault();
      if (isReadOnly) return;
      const trimmedName = draft.name.trim();
      if (!trimmedName) return;
      onSave({
        name: trimmedName,
        description: draft.description.trim(),
        permissions: [...draft.permissions].sort(),
      });
    },
    [draft, isReadOnly, onSave],
  );

  return (
    <form className="threat-card" onSubmit={handleSubmit} aria-label="Edit RBAC role">
      {isReadOnly ? (
        <div
          className="dash-status dash-status-stale"
          style={{ marginBottom: 12, fontSize: 12 }}
          role="note"
        >
          <span className="dash-status-dot" aria-hidden="true" />
          Built-in role — read-only. Create a new role if you need to modify permissions.
        </div>
      ) : null}

      <div className="threat-card-top">
        <strong className="threat-card-type">
          {isNew ? 'New custom role' : isReadOnly ? `Built-in: ${role?.name}` : `Edit: ${role?.name}`}
        </strong>
        <span
          className="dash-status dash-status-up"
          style={{ marginLeft: 'auto' }}
          title={`${selectedCount} of ${totalCount} permissions`}
        >
          <span className="dash-status-dot" aria-hidden="true" />
          {selectedCount} perm{selectedCount === 1 ? '' : 's'}
        </span>
      </div>

      <label
        style={{
          display: 'block',
          marginTop: 12,
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        Name
        <input
          type="text"
          value={draft.name}
          onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
          placeholder="ReadOnly, NightOps, DevOps, …"
          maxLength={64}
          required
          disabled={isReadOnly}
          style={{
            display: 'block',
            width: '100%',
            marginTop: 4,
            padding: 8,
            background: 'var(--surface-2)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--text)',
            opacity: isReadOnly ? 0.6 : 1,
          }}
        />
      </label>

      <label
        style={{
          display: 'block',
          marginTop: 12,
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        Description
        <textarea
          value={draft.description}
          onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}
          placeholder="What is this role for? (optional, max 256 chars)"
          maxLength={256}
          rows={2}
          disabled={isReadOnly}
          style={{
            display: 'block',
            width: '100%',
            marginTop: 4,
            padding: 8,
            background: 'var(--surface-2)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--text)',
            fontFamily: 'inherit',
            resize: 'vertical',
            opacity: isReadOnly ? 0.6 : 1,
          }}
        />
      </label>

      <div
        style={{
          marginTop: 12,
          fontSize: 12,
          color: 'var(--text-muted)',
        }}
      >
        Permissions
      </div>

      <div
        style={{
          marginTop: 6,
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
          gap: 8,
        }}
      >
        {PERMISSION_GROUPS.map((group) => (
          <div
            key={group.label}
            style={{
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              padding: 8,
            }}
          >
            <div
              style={{
                fontSize: 10,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
                color: 'var(--accent)',
                marginBottom: 4,
              }}
            >
              {group.label}
            </div>
            {group.permissions.map((perm) => {
              const checked = draft.permissions.includes(perm);
              return (
                <label
                  key={perm}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 12,
                    cursor: isReadOnly ? 'default' : 'pointer',
                    padding: '2px 0',
                    opacity: isReadOnly ? 0.6 : 1,
                  }}
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => togglePermission(perm)}
                    disabled={isReadOnly}
                    aria-label={`Permission ${perm}`}
                  />
                  <code style={{ color: checked ? 'var(--text)' : 'var(--text-muted)' }}>
                    {perm}
                  </code>
                </label>
              );
            })}
          </div>
        ))}
      </div>

      {!isReadOnly ? (
        <div
          style={{
            marginTop: 16,
            display: 'flex',
            gap: 8,
            justifyContent: 'flex-end',
          }}
        >
          {onCancel ? (
            <motion.button
              type="button"
              className="sw-button sw-button-secondary"
              onClick={onCancel}
              whileHover={reduce ? undefined : buttonSpring.whileHover}
              whileTap={reduce ? undefined : buttonSpring.whileTap}
              transition={buttonSpring.transition}
              disabled={busy}
            >
              Cancel
            </motion.button>
          ) : null}
          <motion.button
            type="submit"
            className="empty-state-cta"
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={busy || !draft.name.trim()}
          >
            {busy ? 'Saving…' : isNew ? 'Create role' : 'Save changes'}
          </motion.button>
        </div>
      ) : null}
    </form>
  );
}
