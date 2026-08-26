/**
 * Tier 14 Phase 14.2 — Hardware tab content.
 * Reads from the full /qemu/:vmid config endpoint.
 */
import { motion } from 'framer-motion';

export interface VMConfig {
  name?: string;
  cores?: number;
  sockets?: number;
  memory?: number;
  scsihw?: string;
  bios?: string;
  machine?: string;
  cpu?: string;
  net0?: string;
  net1?: string;
  boot?: string;
  ostype?: string;
  smbios1?: string;
  [key: string]: unknown;
}

interface Props {
  config: VMConfig;
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="px-summary-row">
      <span className="px-summary-label">{label}</span>
      <span className="px-summary-value px-mono">{value}</span>
    </div>
  );
}

function parseNet(net?: string): { model: string; bridge: string; mac?: string } {
  if (!net) return { model: '—', bridge: '—' };
  // net0: virtio=BC:24:11:8D:5B:9C,bridge=vmbr0
  const parts = net.split(',');
  const head = parts[0] ?? '';
  const [model, mac] = head.split('=');
  const bridgePart = parts.find((p) => p.startsWith('bridge='));
  return {
    model: model || '—',
    bridge: bridgePart ? bridgePart.split('=')[1] ?? '—' : '—',
    mac,
  };
}

export default function ProxmoxDetailHardware({ config }: Props) {
  const totalCores = (config.sockets ?? 1) * (config.cores ?? 1);
  const memoryMB = config.memory ?? 0;
  const net0 = parseNet(config.net0);
  const net1 = parseNet(config.net1);

  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className="px-summary-grid">
        <section className="px-summary-card">
          <h3>CPU</h3>
          <Row label="Sockets" value={config.sockets ?? '—'} />
          <Row label="Cores per socket" value={config.cores ?? '—'} />
          <Row label="Total cores" value={totalCores} />
          <Row label="CPU type" value={config.cpu || 'host'} />
        </section>

        <section className="px-summary-card">
          <h3>Memory</h3>
          <Row label="RAM" value={`${memoryMB} MB (${(memoryMB / 1024).toFixed(1)} GB)`} />
        </section>

        <section className="px-summary-card">
          <h3>Firmware</h3>
          <Row label="BIOS" value={config.bios || 'seabios'} />
          <Row label="Machine" value={config.machine || '—'} />
          <Row label="SCSI controller" value={config.scsihw || 'virtio-scsi-pci'} />
          <Row label="OS type" value={config.ostype || '—'} />
        </section>

        <section className="px-summary-card">
          <h3>Boot order</h3>
          <Row label="Boot" value={config.boot || '—'} />
        </section>

        <section className="px-summary-card">
          <h3>Network</h3>
          <Row label="net0 model" value={net0.model} />
          <Row label="net0 bridge" value={net0.bridge} />
          {net0.mac && <Row label="net0 MAC" value={net0.mac} />}
          {config.net1 && (
            <>
              <Row label="net1 model" value={net1.model} />
              <Row label="net1 bridge" value={net1.bridge} />
              {net1.mac && <Row label="net1 MAC" value={net1.mac} />}
            </>
          )}
        </section>
      </div>
    </motion.div>
  );
}