import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';
import type { RegionRow } from './RegionsSection';

// Tier 11 Phase 6 — Multi-Region / HA (PL6).
//
// Modal subcomponents for RegionsSection:
//
//   - RegionsDetailModal : click a row → show endpoint + last
//     health + latency in a small dialog. The handleClick row
//     pattern matches the rest of the platform/admin UI
//     (BackupSection row → modal, LimitsSection tenant → modal).
//
//   - RegionsAddModal : the create-region form. Controlled
//     inputs (parent owns the state) so the parent can also
//     own the submit + error surface. The regex + endpoint
//     validation lives in the parent's handleAdd so we don't
//     duplicate the regex across both layers.
//
// Both modals are intentionally small (under 100 LOC each) so
// this file stays well under the 400-LOC cap.

interface RegionsDetailModalProps {
  region: RegionRow | null;
  onClose: () => void;
}

export function RegionsDetailModal({ region, onClose }: RegionsDetailModalProps) {
  if (!region) return null;
  return (
    <div
      className="modal-backdrop"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
    >
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-semibold">{region.display_name}</h3>
        <p className="text-sm opacity-70 font-mono">{region.code}</p>
        <dl className="grid grid-cols-2 gap-2 text-sm mt-3">
          <dt className="opacity-60">Kind</dt>
          <dd>{region.region_kind}</dd>
          <dt className="opacity-60">Endpoint</dt>
          <dd className="font-mono break-all">{region.endpoint_url}</dd>
          <dt className="opacity-60">Last status</dt>
          <dd>{region.last_health_status || 'unknown'}</dd>
          <dt className="opacity-60">Latency</dt>
          <dd>
            {region.last_health_latency_ms != null
              ? `${region.last_health_latency_ms} ms`
              : '—'}
          </dd>
          <dt className="opacity-60">Last error</dt>
          <dd>{region.last_health_error || '—'}</dd>
          <dt className="opacity-60">Last probe</dt>
          <dd>
            {region.last_health_at
              ? new Date(region.last_health_at).toLocaleString()
              : 'never'}
          </dd>
        </dl>
        <button type="button" className="btn-secondary mt-4" onClick={onClose}>
          Close
        </button>
      </div>
    </div>
  );
}

interface RegionsAddModalProps {
  open: boolean;
  creating: boolean;
  addCode: string;
  addName: string;
  addKind: string;
  addUrl: string;
  addErr: string;
  onChangeCode: (v: string) => void;
  onChangeName: (v: string) => void;
  onChangeKind: (v: string) => void;
  onChangeUrl: (v: string) => void;
  onSubmit: () => void;
  onClose: () => void;
}

export function RegionsAddModal({
  open,
  creating,
  addCode,
  addName,
  addKind,
  addUrl,
  addErr,
  onChangeCode,
  onChangeName,
  onChangeKind,
  onChangeUrl,
  onSubmit,
  onClose,
}: RegionsAddModalProps) {
  const reduce = useReducedMotion();
  if (!open) return null;
  return (
    <div
      className="modal-backdrop"
      onClick={() => !creating && onClose()}
      role="dialog"
      aria-modal="true"
    >
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-semibold">Add region</h3>
        <p className="text-sm opacity-70">
          Register a new region. The probe path will hit the
          endpoint_url within ~1s of save.
        </p>
        <label className="block mt-3 text-sm">
          <span className="opacity-70">Code (lowercase URL-safe)</span>
          <input
            type="text"
            value={addCode}
            onChange={(e) => onChangeCode(e.target.value)}
            placeholder="us-east-1"
            className="input mt-1 w-full"
            disabled={creating}
          />
        </label>
        <label className="block mt-3 text-sm">
          <span className="opacity-70">Display name</span>
          <input
            type="text"
            value={addName}
            onChange={(e) => onChangeName(e.target.value)}
            placeholder="US East (primary)"
            className="input mt-1 w-full"
            disabled={creating}
          />
        </label>
        <label className="block mt-3 text-sm">
          <span className="opacity-70">Kind</span>
          <select
            value={addKind}
            onChange={(e) => onChangeKind(e.target.value)}
            className="input mt-1 w-full"
            disabled={creating}
          >
            <option value="primary">primary</option>
            <option value="replica">replica</option>
            <option value="standby">standby</option>
          </select>
        </label>
        <label className="block mt-3 text-sm">
          <span className="opacity-70">Endpoint URL (http/https)</span>
          <input
            type="text"
            value={addUrl}
            onChange={(e) => onChangeUrl(e.target.value)}
            placeholder="https://us-east-1.example.com/health"
            className="input mt-1 w-full"
            disabled={creating}
          />
        </label>
        {addErr ? (
          <div className="callout-error mt-3" role="alert">
            {addErr}
          </div>
        ) : null}
        <div className="flex gap-2 mt-4 justify-end">
          <button
            type="button"
            className="btn-secondary"
            onClick={onClose}
            disabled={creating}
          >
            Cancel
          </button>
          <motion.button
            type="button"
            className="btn-primary"
            onClick={onSubmit}
            disabled={creating}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
          >
            {creating ? 'Adding…' : 'Add region'}
          </motion.button>
        </div>
      </div>
    </div>
  );
}