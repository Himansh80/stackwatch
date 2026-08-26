/**
 * Tier 14 Phase 14.4 — LXC wizard step 2: Resources (cores, memory, disk).
 */
import { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import { readString } from '../../lib/proxmox';
import type { LxcSpec } from './lib/lxc-types';

interface Storage {
  storage: string;
  content: string;
}

interface Props {
  hostId: string;
  node: string;
  spec: LxcSpec;
  onChange: (patch: Partial<LxcSpec>) => void;
}

export default function ProxmoxLxcCreateStep2Resources({ hostId, node, spec, onChange }: Props) {
  const [storages, setStorages] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!hostId || !node) return;
    let alive = true;
    (async () => {
      try {
        const data = await api<{ storage?: Storage[] } | Storage[]>(
          'GET',
          `/api/v1/proxmox/hosts/${hostId}/storage`,
        );
        const list = Array.isArray(data) ? data : data.storage ?? [];
        const rootStorages = list
          .filter((s) => readString(s.content).includes('rootdir'))
          .map((s) => s.storage);
        if (alive) {
          setStorages(rootStorages);
          if (rootStorages.length > 0 && !spec.storage) {
            onChange({ storage: rootStorages[0] ?? '' });
          }
        }
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostId, node]);

  return (
    <div>
      <h3 className="px-wizard-step-title">Resources</h3>
      <p className="px-wizard-step-desc">
        Set CPU cores, memory, swap, and disk size for the container.
      </p>
      <div className="px-form-grid">
        <div className="px-form-field">
          <label>VMID</label>
          <input
            type="number"
            min="100"
            value={spec.vmid}
            onChange={(e) => onChange({ vmid: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field">
          <label>Hostname *</label>
          <input
            type="text"
            value={spec.hostname}
            onChange={(e) => onChange({ hostname: e.target.value })}
            placeholder="my-container"
            required
          />
        </div>
        <div className="px-form-field">
          <label>Cores</label>
          <input
            type="number"
            min="1"
            max="32"
            value={spec.cores}
            onChange={(e) => onChange({ cores: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field">
          <label>Memory (MB)</label>
          <input
            type="number"
            min="128"
            step="64"
            value={spec.memory}
            onChange={(e) => onChange({ memory: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field">
          <label>Swap (MB, 0 = none)</label>
          <input
            type="number"
            min="0"
            step="64"
            value={spec.swap}
            onChange={(e) => onChange({ swap: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field">
          <label>Disk (GB)</label>
          <input
            type="number"
            min="1"
            step="1"
            value={spec.disk}
            onChange={(e) => onChange({ disk: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field px-form-full">
          <label>Root storage *</label>
          {loading ? (
            <p className="px-muted">Loading storages…</p>
          ) : storages.length === 0 ? (
            <p className="px-confirm-warn">No rootdir-capable storage on this node.</p>
          ) : (
            <select
              value={spec.storage}
              onChange={(e) => onChange({ storage: e.target.value })}
              className="px-form-select"
            >
              <option value="">— select —</option>
              {storages.map((s) => <option key={s} value={s}>{s}</option>)}
            </select>
          )}
        </div>
      </div>
    </div>
  );
}