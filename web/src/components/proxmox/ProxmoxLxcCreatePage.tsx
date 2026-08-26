/**
 * Tier 14 Phase 14.4 — LXC create wizard orchestrator.
 * 4-step flow with progress bar.
 */
import { useCallback, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';
import ProxmoxHostSelector from './ProxmoxHostSelector';
import ProxmoxLxcCreateStep1Template from './ProxmoxLxcCreateStep1Template';
import type { Template } from './ProxmoxLxcTemplatePicker';
import ProxmoxLxcCreateStep2Resources from './ProxmoxLxcCreateStep2Resources';
import ProxmoxLxcCreateStep3Network from './ProxmoxLxcCreateStep3Network';
import ProxmoxLxcCreateStep4Confirm from './ProxmoxLxcCreateStep4Confirm';
import { DEFAULT_LXC_SPEC, type LxcSpec } from './lib/lxc-types';

const STEPS = [
  { id: 1, title: 'Template' },
  { id: 2, title: 'Resources' },
  { id: 3, title: 'Network' },
  { id: 4, title: 'Confirm' },
] as const;

export default function ProxmoxLxcCreatePage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [hostId, setHostId] = useState(searchParams.get('hostId') ?? '');
  const [node, setNode] = useState(searchParams.get('node') ?? '');
  const [step, setStep] = useState(1);
  const [spec, setSpec] = useState<LxcSpec>(DEFAULT_LXC_SPEC);
  const [template, setTemplate] = useState<Template | null>(null);
  const [error, setError] = useState<string>('');
  const [busy, setBusy] = useState(false);

  const updateSpec = useCallback((patch: Partial<LxcSpec>) => {
    setSpec((s) => ({ ...s, ...patch }));
  }, []);

  function next() {
    setError('');
    if (step === 1 && !spec.ostemplate) {
      setError('Pick a template first.');
      return;
    }
    if (step === 2) {
      if (!spec.hostname) { setError('Hostname is required.'); return; }
      if (!spec.storage) { setError('Root storage is required.'); return; }
      if (spec.vmid < 100) { setError('VMID must be >= 100.'); return; }
    }
    if (step === 3 && spec.ipv4Mode === 'static' && spec.ipv4Addr && !/^\d+\.\d+\.\d+\.\d+\/\d+$/.test(spec.ipv4Addr)) {
      setError('IPv4 address must be in CIDR format (e.g. 192.168.0.50/24).');
      return;
    }
    setStep((s) => Math.min(4, s + 1));
  }
  function back() { setStep((s) => Math.max(1, s - 1)); }

  const submit = useCallback(async () => {
    if (!hostId || !node) return;
    setBusy(true);
    setError('');
    try {
      // Build Proxmox API body
      const body: Record<string, unknown> = {
        vmid: spec.vmid,
        hostname: spec.hostname,
        ostemplate: spec.ostemplate,
        cores: spec.cores,
        memory: spec.memory,
        swap: spec.swap,
        rootfs: `${spec.storage}:${spec.disk}`,
        net0: buildNet0(spec),
        start: spec.startOnBoot ? 1 : 0,
      };
      await api(
        'POST',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/lxc`,
        body,
      );
      navigate(`/proxmox-lxc/${hostId}/${encodeURIComponent(node)}/${spec.vmid}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to create container.');
      setBusy(false);
    }
  }, [hostId, node, spec, navigate]);

  if (!hostId || !node) {
    return (
      <div className="px-vm-page">
        <ProxmoxHostSelector
          selected={hostId}
          onChange={(id) => {
            setHostId(id);
            // Pick first node from existing API
            api<{ nodes?: Array<{ node: string }> }>(
              'GET',
              `/api/v1/proxmox/hosts/${id}/nodes`,
            )
              .then((d) => {
                const first = (Array.isArray(d) ? d : d.nodes ?? [])[0];
                if (first) setNode(first.node);
              })
              .catch(() => undefined);;
          }}
        />
        <div className="px-info-card">
          <h3>Select a host and node</h3>
          <p>Pick where the new container will live.</p>
        </div>
      </div>
    );
  }

  return (
    <ProxmoxShell title="Create LXC" subtitle="4-step wizard to provision a new container">
      <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-wizard-header">
        <h2>Create LXC container</h2>
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
          <ProxmoxLxcCreateStep1Template
            hostId={hostId}
            node={node}
            selectedVolid={spec.ostemplate}
            onSelect={(t) => {
              setTemplate(t);
              updateSpec({ ostemplate: t.volid });
            }}
          />
        )}
        {step === 2 && (
          <ProxmoxLxcCreateStep2Resources
            hostId={hostId}
            node={node}
            spec={spec}
            onChange={updateSpec}
          />
        )}
        {step === 3 && (
          <ProxmoxLxcCreateStep3Network spec={spec} onChange={updateSpec} />
        )}
        {step === 4 && (
          <ProxmoxLxcCreateStep4Confirm
            spec={spec}
            templateName={template?.volid.split('/').pop() ?? ''}
            onToggleStart={() => updateSpec({ startOnBoot: !spec.startOnBoot })}
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
            {busy ? 'Creating…' : 'Create container'}
          </button>
        )}
      </div>
    </motion.div>
    </ProxmoxShell>
  );
}

function buildNet0(spec: LxcSpec): string {
  let parts = `name=eth0,bridge=${spec.bridge}`;
  if (spec.vlanTag) parts += `,tag=${spec.vlanTag}`;
  if (spec.ipv4Mode === 'static' && spec.ipv4Addr) {
    parts += `,ip=${spec.ipv4Addr}${spec.ipv4Gw ? `,gw=${spec.ipv4Gw}` : ''}`;
  } else if (spec.ipv4Mode === 'dhcp') {
    parts += ',ip=dhcp';
  }
  if (spec.ipv6Mode === 'static' && spec.ipv6Addr) {
    parts += `,ip6=${spec.ipv6Addr}${spec.ipv6Gw ? `,gw6=${spec.ipv6Gw}` : ''}`;
  } else if (spec.ipv6Mode === 'dhcp') {
    parts += ',ip6=dhcp';
  } else if (spec.ipv6Mode === 'slaac') {
    parts += ',ip6=auto';
  }
  return parts;
}