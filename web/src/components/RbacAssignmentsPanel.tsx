import { useCallback, useEffect, useState } from 'react';
import { ApiError, api } from '../lib/api';
import { motion, buttonSpring, useReducedMotion } from '../lib/motion';

/**
 * RbacAssignmentsPanel — Tier 9.3 (Phase 3) extracted sub-section
 * for the user↔role assignments surface. Lives in its own file so
 * RbacSection.tsx stays under the 400-LOC cap.
 *
 * Layout:
 *   - User picker (lazy-loaded /api/v1/users) + Role picker
 *   - Assign button
 *   - List of current assignments (cards) with Unassign buttons
 *
 * Owns the local "which user/role is selected" state so the parent
 * doesn't need to thread it through. Calls `onChanged()` after
 * successful mutations so the parent can refetch.
 */

export interface RbacAssignmentRow {
  id: string;
  tenant_id: string;
  user_id: string;
  role_id: string;
  role_name?: string;
  assigned_at: string;
}

export interface RbacRoleRef {
  id: string;
  name: string;
  is_builtin: boolean;
}

interface RbacAssignmentsPanelProps {
  assignments: RbacAssignmentRow[];
  roles: RbacRoleRef[];
  busy?: boolean;
  onError: (msg: string) => void;
  onChanged: () => void;
}

interface UserOption {
  id: string;
  email: string;
  full_name?: string;
}

interface UsersListResponse {
  users: UserOption[];
  total?: number;
}

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

export default function RbacAssignmentsPanel({
  assignments,
  roles,
  busy = false,
  onError,
  onChanged,
}: RbacAssignmentsPanelProps) {
  const reduce = useReducedMotion();
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  const [users, setUsers] = useState<UserOption[]>([]);
  const [usersLoaded, setUsersLoaded] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [selectedRoleId, setSelectedRoleId] = useState('');

  // Lazy-load user list once on mount.
  useEffect(() => {
    if (usersLoaded) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await api<UsersListResponse>('GET', '/api/v1/users');
        if (!cancelled) {
          setUsers(res.users || []);
          setUsersLoaded(true);
        }
      } catch (cause) {
        if (!cancelled) {
          onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
          setUsersLoaded(true);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [usersLoaded, onError]);

  const handleAssign = useCallback(async () => {
    if (!selectedUserId || !selectedRoleId) {
      onError('Select a user and a role before assigning.');
      return;
    }
    setSectionBusy(true);
    try {
      await api('POST', `/api/v1/enterprise/rbac/users/${selectedUserId}/roles`, {
        role_id: selectedRoleId,
      });
      setSelectedUserId('');
      setSelectedRoleId('');
      onChanged();
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  }, [selectedUserId, selectedRoleId, onChanged, onError]);

  const handleUnassign = useCallback(
    async (a: RbacAssignmentRow) => {
      setSectionBusy(true);
      try {
        await api(
          'DELETE',
          `/api/v1/enterprise/rbac/users/${a.user_id}/roles/${a.role_id}`,
        );
        onChanged();
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onChanged, onError],
  );

  return (
    <>
      <div
        className="threat-card"
        style={{ display: 'flex', gap: 8, alignItems: 'flex-end', flexWrap: 'wrap' }}
      >
        <label style={{ flex: '1 1 200px', fontSize: 12, color: 'var(--text-muted)' }}>
          User
          <select
            value={selectedUserId}
            onChange={(e) => setSelectedUserId(e.target.value)}
            disabled={isBusy || users.length === 0}
            style={{
              display: 'block',
              width: '100%',
              marginTop: 4,
              padding: 8,
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              color: 'var(--text)',
            }}
          >
            <option value="">
              {users.length === 0 ? 'Loading users…' : 'Select a user'}
            </option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.full_name ? `${u.full_name} (${u.email})` : u.email}
              </option>
            ))}
          </select>
        </label>
        <label style={{ flex: '1 1 200px', fontSize: 12, color: 'var(--text-muted)' }}>
          Role
          <select
            value={selectedRoleId}
            onChange={(e) => setSelectedRoleId(e.target.value)}
            disabled={isBusy}
            style={{
              display: 'block',
              width: '100%',
              marginTop: 4,
              padding: 8,
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-md)',
              color: 'var(--text)',
            }}
          >
            <option value="">Select a role</option>
            {roles.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
                {r.is_builtin ? ' (built-in)' : ''}
              </option>
            ))}
          </select>
        </label>
        <motion.button
          type="button"
          className="empty-state-cta"
          onClick={() => void handleAssign()}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
          disabled={isBusy || !selectedUserId || !selectedRoleId}
          style={{ alignSelf: 'flex-end' }}
        >
          Assign
        </motion.button>
      </div>

      {assignments.length > 0 ? (
        <div className="threat-card-list" style={{ marginTop: 12 }}>
          {assignments.map((a) => {
            const user = users.find((u) => u.id === a.user_id);
            return (
              <article key={a.id} className="threat-card">
                <div className="threat-card-top">
                  <span className="dash-sev dash-sev-low">ASSIGNMENT</span>
                  <strong className="threat-card-type">
                    {user
                      ? user.full_name
                        ? `${user.full_name} (${user.email})`
                        : user.email
                      : a.user_id}
                  </strong>
                  <span className="dash-sev dash-sev-medium" style={{ marginLeft: 'auto' }}>
                    {a.role_name ?? a.role_id}
                  </span>
                </div>
                <div className="threat-card-meta">
                  <button
                    type="button"
                    className="threat-card-resolve-btn"
                    onClick={() => void handleUnassign(a)}
                    aria-label={`Unassign ${a.role_name ?? a.role_id}`}
                    style={{ color: 'var(--red)' }}
                    disabled={isBusy}
                  >
                    Unassign
                  </button>
                  <span
                    style={{
                      marginLeft: 'auto',
                      fontSize: 11,
                      color: 'var(--text-muted)',
                    }}
                    title={a.assigned_at}
                  >
                    assigned {relativeTime(a.assigned_at)}
                  </span>
                </div>
              </article>
            );
          })}
        </div>
      ) : null}
    </>
  );
}
