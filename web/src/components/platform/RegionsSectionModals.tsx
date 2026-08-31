import type { RegionRow } from './RegionsSection';
import Button from '../shared/Button';
import Input from '../shared/Input';
import Modal from '../shared/Modal';
import Select, { type SelectOption } from '../shared/Select';

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

const REGION_KIND_OPTIONS: SelectOption[] = [
  { value: 'primary', label: 'primary' },
  { value: 'replica', label: 'replica' },
  { value: 'standby', label: 'standby' },
];

interface RegionsDetailModalProps {
  region: RegionRow | null;
  onClose: () => void;
}

export function RegionsDetailModal({ region, onClose }: RegionsDetailModalProps) {
  return (
    <Modal
      open={!!region}
      onClose={onClose}
      title={region?.display_name ?? ''}
    >
      {region ? (
        <>
          <p
            style={{
              fontFamily: 'var(--font-mono, monospace)',
              fontSize: 13,
              color: 'var(--text-muted)',
              margin: '0 0 var(--space-4)',
            }}
          >
            {region.code}
          </p>
          <dl
            style={{
              display: 'grid',
              gridTemplateColumns: 'auto 1fr',
              gap: 'var(--space-2) var(--space-4)',
              fontSize: 14,
              margin: 0,
            }}
          >
            <dt style={{ color: 'var(--text-muted)' }}>Kind</dt>
            <dd style={{ margin: 0 }}>{region.region_kind}</dd>
            <dt style={{ color: 'var(--text-muted)' }}>Endpoint</dt>
            <dd style={{ margin: 0, fontFamily: 'var(--font-mono, monospace)', wordBreak: 'break-all' }}>
              {region.endpoint_url}
            </dd>
            <dt style={{ color: 'var(--text-muted)' }}>Last status</dt>
            <dd style={{ margin: 0 }}>{region.last_health_status || 'unknown'}</dd>
            <dt style={{ color: 'var(--text-muted)' }}>Latency</dt>
            <dd style={{ margin: 0 }}>
              {region.last_health_latency_ms != null
                ? `${region.last_health_latency_ms} ms`
                : '—'}
            </dd>
            <dt style={{ color: 'var(--text-muted)' }}>Last error</dt>
            <dd style={{ margin: 0 }}>{region.last_health_error || '—'}</dd>
            <dt style={{ color: 'var(--text-muted)' }}>Last probe</dt>
            <dd style={{ margin: 0 }}>
              {region.last_health_at
                ? new Date(region.last_health_at).toLocaleString()
                : 'never'}
            </dd>
          </dl>
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginTop: 'var(--space-4)',
            }}
          >
            <Button type="button" variant="secondary" onClick={onClose}>
              Close
            </Button>
          </div>
        </>
      ) : null}
    </Modal>
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
  return (
    <Modal open={open} onClose={onClose} title="Add region">
      <p
        style={{
          fontSize: 13,
          color: 'var(--text-muted)',
          margin: '0 0 var(--space-4)',
        }}
      >
        Register a new region. The probe path will hit the
        endpoint_url within ~1s of save.
      </p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
        <Input
          label="Code (lowercase URL-safe)"
          value={addCode}
          onChange={(e) => onChangeCode(e.target.value)}
          placeholder="us-east-1"
          disabled={creating}
          fullWidth
        />
        <Input
          label="Display name"
          value={addName}
          onChange={(e) => onChangeName(e.target.value)}
          placeholder="US East (primary)"
          disabled={creating}
          fullWidth
        />
        <Select
          label="Kind"
          value={addKind}
          onChange={(e) => onChangeKind(e.target.value)}
          options={REGION_KIND_OPTIONS}
          disabled={creating}
          fullWidth
        />
        <Input
          label="Endpoint URL (http/https)"
          value={addUrl}
          onChange={(e) => onChangeUrl(e.target.value)}
          placeholder="https://us-east-1.example.com/health"
          disabled={creating}
          fullWidth
        />
        {addErr ? (
          <div className="callout-error" role="alert">
            {addErr}
          </div>
        ) : null}
        <div
          style={{
            display: 'flex',
            gap: 'var(--space-2)',
            justifyContent: 'flex-end',
            marginTop: 'var(--space-2)',
          }}
        >
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={creating}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            onClick={onSubmit}
            disabled={creating}
            loading={creating}
          >
            {creating ? 'Adding…' : 'Add region'}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
