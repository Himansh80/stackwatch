/**
 * Tier 14 Phase 14.6 — VM wizard step 2: Hardware (extended with Cloud-Init / EFI / TPM).
 */
import { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import { readString } from '../../lib/proxmox';
import type { VmSpec } from './lib/vm-types';

interface Storage {
  storage: string;
  content: string;
}

interface Props {
  hostId: string;
  node: string;
  spec: VmSpec;
  onChange: (patch: Partial<VmSpec>) => void;
}

export default function ProxmoxVmCreateStep2Hardware({ hostId, node, spec, onChange }: Props) {
  const [storages, setStorages] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!hostId || !node) return;
    let alive = true;
    (async () => {
      try {
        const data = await api<{ storage?: Storage[] } | Storage[]>(
          'GET',
          `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage`,
        );
        const list = Array.isArray(data) ? data : data.storage ?? [];
        const imgStorages = list
          .filter((s) => readString(s.content).includes('images') || readString(s.content).includes('rootdir'))
          .map((s) => s.storage);
        if (alive) {
          setStorages(imgStorages);
          if (imgStorages.length > 0 && !spec.storage) {
            onChange({ storage: imgStorages[0] ?? '' });
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
      <h3 className="px-wizard-step-title">Hardware</h3>
      <p className="px-wizard-step-desc">
        Configure VMID, name, CPU, memory, disk, BIOS, machine, and boot type.
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
          <label>Name *</label>
          <input
            type="text"
            value={spec.name}
            onChange={(e) => onChange({ name: e.target.value })}
            placeholder="my-vm"
            required
          />
        </div>
        <div className="px-form-field">
          <label>Cores</label>
          <input
            type="number"
            min="1"
            max="128"
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
          <label>Disk (GB)</label>
          <input
            type="number"
            min="1"
            step="1"
            value={spec.disk}
            onChange={(e) => onChange({ disk: Number(e.target.value) })}
          />
        </div>
        <div className="px-form-field">
          <label>Root storage *</label>
          {loading ? (
            <p className="px-muted">Loading storages…</p>
          ) : storages.length === 0 ? (
            <p className="px-confirm-warn">No image-capable storage on this node.</p>
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
        <div className="px-form-field">
          <label>BIOS</label>
          <select
            value={spec.bios}
            onChange={(e) => onChange({ bios: e.target.value as VmSpec['bios'] })}
            className="px-form-select"
          >
            <option value="seabios">SeaBIOS (legacy)</option>
            <option value="ovmf">OVMF (UEFI)</option>
          </select>
        </div>
        <div className="px-form-field">
          <label>Machine</label>
          <select
            value={spec.machine}
            onChange={(e) => onChange({ machine: e.target.value as VmSpec['machine'] })}
            className="px-form-select"
          >
            <option value="q35">q35 (modern)</option>
            <option value="i440fx">i440fx (legacy)</option>
          </select>
        </div>
        <div className="px-form-field">
          <label>CPU type</label>
          <select
            value={spec.cpuType}
            onChange={(e) => onChange({ cpuType: e.target.value })}
            className="px-form-select"
          >
            <option value="host">host (passthrough)</option>
            <option value="kvm64">kvm64 (default)</option>
            <option value="x86-64-v2-AES">x86-64-v2-AES</option>
            <option value="x86-64-v3">x86-64-v3</option>
          </select>
        </div>
        <div className="px-form-field">
          <label>OS type</label>
          <select
            value={spec.osType}
            onChange={(e) => onChange({ osType: e.target.value as VmSpec['osType'] })}
            className="px-form-select"
          >
            <option value="l26">Linux 2.6+</option>
            <option value="win11">Windows 11 / 2022</option>
            <option value="other">Other</option>
          </select>
        </div>
      </div>

      <div className="px-form-toggles">
        <div className="px-form-field px-form-toggle">
          <label htmlFor="px-vm-cloudinit">
            <input
              id="px-vm-cloudinit"
              type="checkbox"
              checked={spec.cloudInit}
              onChange={(e) => onChange({ cloudInit: e.target.checked })}
            />
            Cloud-Init drive (first-boot user/network/ssh-key config)
          </label>
        </div>
        <div className="px-form-field px-form-toggle">
          <label htmlFor="px-vm-efi">
            <input
              id="px-vm-efi"
              type="checkbox"
              checked={spec.efiDisk}
              onChange={(e) => onChange({ efiDisk: e.target.checked, tpmState: e.target.checked ? spec.tpmState : false })}
            />
            EFI disk (OVMF + EFI storage, required for Windows / modern Linux)
          </label>
        </div>
        <div className="px-form-field px-form-toggle">
          <label htmlFor="px-vm-tpm" className={spec.efiDisk ? '' : 'px-form-disabled'}>
            <input
              id="px-vm-tpm"
              type="checkbox"
              checked={spec.tpmState}
              onChange={(e) => onChange({ tpmState: e.target.checked })}
              disabled={!spec.efiDisk}
            />
            TPM state (required for Windows 11)
          </label>
        </div>
      </div>
    </div>
  );
}