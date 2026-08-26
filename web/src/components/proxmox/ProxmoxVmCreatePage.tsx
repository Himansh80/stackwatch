/**
 * Tier 14 Phase 14.6 — VM create wizard orchestrator (extended for multi-NIC + EFI + Cloud-Init).
 */
import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';
import ProxmoxHostSelector from './ProxmoxHostSelector';
import ProxmoxVmCreateStep1Iso, { type Iso } from './ProxmoxVmCreateStep1Iso';
import ProxmoxVmCreateStep2Hardware from './ProxmoxVmCreateStep2Hardware';
import ProxmoxVmCreateStep3Network from './ProxmoxVmCreateStep3Network';
import ProxmoxVmCreateStep4Confirm from './ProxmoxVmCreateStep4Confirm';
import { DEFAULT_VM_SPEC, type Nic, type VmSpec } from './lib/vm-types';

const STEPS = [
  { id: 1, title: 'ISO' },
  { id: 2, title: 'Hardware' },
  { id: 3, title: 'Network' },
  { id: 4, title: 'Confirm' },
] as const;

function buildNicStr(nic: Nic): string {
  let s = `${nic.netModel},bridge=${nic.bridge}`;
  if (nic.vlanTag) s += `,tag=${nic.vlanTag}`;
  return s;
}

export default function ProxmoxVmCreatePage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [hostId, setHostId] = useState(searchParams.get('hostId') ?? '');
  const [node, setNode] = useState(searchParams.get('node') ?? '');
  const [nodes, setNodes] = useState<Array<{ node: string }>>([]);
  const [step, setStep] = useState(1);
  const [spec, setSpec] = useState<VmSpec>(DEFAULT_VM_SPEC);
  const [iso, setIso] = useState<Iso | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!hostId) return;
    let alive = true;
    api<{ nodes?: Array<{ node: string }> } | Array<{ node: string }>>(
      'GET',
      `/api/v1/proxmox/hosts/${hostId}/nodes`,
    )
      .then((d) => {
        if (!alive) return;
        const list = Array.isArray(d) ? d : d.nodes ?? [];
        setNodes(list);
        if (list.length > 0 && !node) {
          const first = list[0];
          if (first) setNode(first.node);
        }
      })
      .catch(() => {/* ignore */});
    return () => {
      alive = false;
    };
  }, [hostId, node]);

  const updateSpec = useCallback((patch: Partial<VmSpec>) => {
    setSpec((s) => ({ ...s, ...patch }));
  }, []);

  function next() {
    setError('');
    if (step === 1 && !iso) {
      setError('Pick an ISO first (or "No media").');
      return;
    }
    if (step === 2) {
      if (!spec.name) { setError('Name is required.'); return; }
      if (!spec.storage) { setError('Root storage is required.'); return; }
      if (spec.vmid < 100) { setError('VMID must be >= 100.'); return; }
    }
    if (step === 3 && spec.nics.length === 0) {
      setError('Add at least one NIC.'); return;
    }
    setStep((s) => Math.min(4, s + 1));
  }
  function back() { setStep((s) => Math.max(1, s - 1)); }

  const submit = useCallback(async () => {
    if (!hostId || !node) return;
    setBusy(true);
    setError('');
    try {
      const body: Record<string, unknown> = {
        vmid: spec.vmid,
        name: spec.name,
        memory: spec.memory,
        cores: spec.cores,
        scsihw: 'virtio-scsi-pci',
        bios: spec.bios,
        machine: spec.machine,
        cpu: spec.cpuType,
        ostype: spec.osType,
        scsi0: `${spec.storage}:${spec.disk}`,
        ide2: spec.iso ? `${spec.iso},media=cdrom` : 'none,media=cdrom',
        start: spec.startAfterCreate ? 1 : 0,
      };
      // Multi-NIC: net0..netN
      spec.nics.forEach((nic, idx) => {
        body[`net${idx}`] = buildNicStr(nic);
      });
      // Cloud-Init drive
      if (spec.cloudInit) {
        body.ide2 = 'none,media=cdrom';
        body.cicustom = `user=${spec.storage}:snippets/${spec.vmid}-user.yaml`;
      }
      // EFI disk
      if (spec.efiDisk) {
        body.efidisk0 = `${spec.storage}:1,efitype=4m,pre-enrolled-keys=1`;
      }
      // TPM state
      if (spec.tpmState) {
        body.tpmstate0 = `${spec.storage}:1,version=v2.0`;
      }
      await api(
        'POST',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/qemu`,
        body,
      );
      navigate(`/proxmox-vms/${hostId}/${encodeURIComponent(node)}/${spec.vmid}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to create VM.');
      setBusy(false);
    }
  }, [hostId, node, spec, navigate]);

  if (!hostId || !node) {
    return (
      <div className="px-vm-page">
        <ProxmoxHostSelector
          selected={hostId}
          onChange={(id) => setHostId(id)}
        />
        {nodes.length > 0 && (
          <div className="px-form-field">
            <label>Node</label>
            <select
              value={node}
              onChange={(e) => setNode(e.target.value)}
              className="px-form-select"
            >
              <option value="">— select —</option>
              {nodes.map((n) => <option key={n.node} value={n.node}>{n.node}</option>)}
            </select>
          </div>
        )}
        <div className="px-info-card">
          <h3>Select a host and node</h3>
          <p>Pick where the new VM will live.</p>
        </div>
      </div>
    );
  }

  return (
    <ProxmoxShell title="Create VM" subtitle="4-step wizard to provision a new VM">
      <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-wizard-header">
        <h2>Create VM</h2>
        <p>on {hostId.slice(0, 8)} / {node}</p>
      </div>

      <ol className="px-wizard-steps">
        {STEPS.map((s) => (
          <li
            key={s.id}
            className={`px-wizard-step ${step === s.id ? 'px-wizard-active' : ''} ${step > s.id ? 'px-wizard-done' : ''}`}
          >
            <span className="px-wizard-step-num">{step > s.id ? '✓' : s.id}</span>
            <span>{s.title}</span>
          </li>
        ))}
      </ol>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-wizard-body">
        {step === 1 && (
          <ProxmoxVmCreateStep1Iso
            hostId={hostId}
            node={node}
            selectedVolid={spec.iso}
            onSelect={(i) => {
              setIso(i);
              updateSpec({ iso: i.volid });
            }}
          />
        )}
        {step === 2 && (
          <ProxmoxVmCreateStep2Hardware
            hostId={hostId}
            node={node}
            spec={spec}
            onChange={updateSpec}
          />
        )}
        {step === 3 && (
          <ProxmoxVmCreateStep3Network spec={spec} onChange={updateSpec} />
        )}
        {step === 4 && (
          <ProxmoxVmCreateStep4Confirm
            spec={spec}
            isoName={iso?.volid.split('/').pop() ?? ''}
            onToggleStart={() => updateSpec({ startAfterCreate: !spec.startAfterCreate })}
          />
        )}
      </div>

      <div className="px-wizard-actions">
        {step > 1 && (
          <button type="button" className="px-btn px-btn-quiet" onClick={back} disabled={busy}>
            ← Back
          </button>
        )}
        {step < 4 && (
          <button type="button" className="px-btn" onClick={next}>
            Next →
          </button>
        )}
        {step === 4 && (
          <button type="button" className="px-btn" onClick={submit} disabled={busy}>
            {busy ? 'Creating…' : 'Create VM'}
          </button>
        )}
      </div>
    </motion.div>
    </ProxmoxShell>
  );
}