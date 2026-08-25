import { FormEvent } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../lib/motion';
import type { EnterpriseOrgRow } from './EnterpriseOrgsSection';

/**
 * EnterpriseOrgsSectionHelpers — sibling helpers for the Orgs tab
 * extracted to keep EnterpriseOrgsSection under the 400-LOC cap.
 *
 *   - CreateOrgForm: the "Create sub-org" form body
 *   - createOrgFormInitial: empty-form factory
 *
 * Mirrors ComplianceSectionHelpers / ComplianceSectionModals /
 * ComplianceSectionFormViews — same modular pattern Tier 8/9 uses
 * to keep section components slim.
 */

export interface CreateOrgFormState {
  name: string;
  slug: string;
  parent_orgId: string;
}

export function createOrgFormInitial(): CreateOrgFormState {
  return { name: '', slug: '', parent_orgId: '' };
}

interface CreateOrgFormProps {
  form: CreateOrgFormState;
  setForm: (updater: (prev: CreateOrgFormState) => CreateOrgFormState) => void;
  orgs: EnterpriseOrgRow[];
  busy: boolean;
  errorMessage: string;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
}

export function CreateOrgForm({
  form,
  setForm,
  orgs,
  busy,
  errorMessage,
  onSubmit,
}: CreateOrgFormProps) {
  const reduce = useReducedMotion();
  const topLevel = orgs.filter((o) => !o.parent_org_id);

  return (
    <form className="threat-card" onSubmit={onSubmit} style={{ marginTop: 12 }}>
      <div className="threat-card-top">
        <strong className="threat-card-type">Create org</strong>
      </div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: 12,
          marginTop: 12,
        }}
      >
        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Name</span>
          <input
            className="form-input"
            type="text"
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            required
            minLength={1}
            maxLength={255}
            disabled={busy}
            placeholder="Engineering"
          />
        </label>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Slug</span>
          <input
            className="form-input"
            type="text"
            value={form.slug}
            onChange={(e) =>
              setForm((f) => ({ ...f, slug: e.target.value.toLowerCase() }))
            }
            required
            minLength={1}
            maxLength={64}
            pattern="[a-z0-9][a-z0-9-]*[a-z0-9]"
            disabled={busy}
            placeholder="engineering"
          />
        </label>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12 }}>
          <span style={{ color: 'var(--muted)' }}>Parent org (optional)</span>
          <select
            className="form-input"
            value={form.parent_orgId}
            onChange={(e) => setForm((f) => ({ ...f, parent_orgId: e.target.value }))}
            disabled={busy}
          >
            <option value="">— top-level org —</option>
            {topLevel.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </label>
      </div>
      {errorMessage ? (
        <div
          role="alert"
          style={{
            marginTop: 8,
            padding: '6px 10px',
            background: 'rgba(239, 68, 68, 0.08)',
            border: '1px solid rgba(239, 68, 68, 0.4)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--red, #ef4444)',
            fontSize: 12,
          }}
        >
          {errorMessage}
        </div>
      ) : null}
      <div style={{ display: 'flex', gap: 8, marginTop: 12, justifyContent: 'flex-end' }}>
        <motion.button
          type="submit"
          className="empty-state-cta"
          disabled={busy || !form.name.trim() || !form.slug.trim()}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
        >
          {busy ? 'Creating…' : 'Create'}
        </motion.button>
      </div>
    </form>
  );
}
