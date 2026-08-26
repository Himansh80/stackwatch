/**
 * Tier 14 Phase 14.7 — Node KPI strip (CPU / RAM / Disk / Load / Uptime).
 */
import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { formatBytes, formatPercent, formatUptime, readString } from '../../lib/proxmox';

interface NodeStatus {
  cpu?: number;
  maxcpu?: number;
  mem?: number;
  maxmem?: number;
  disk?: number;
  maxdisk?: number;
  uptime?: number;
  loadavg?: [string, string, string];
  status?: string;
}

interface Props {
  status: NodeStatus;
  loading?: boolean;
}

function useCountUp(target: number, duration = 600): number {
  const [value, setValue] = useState(0);
  useEffect(() => {
    const start = performance.now();
    let raf = 0;
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / duration);
      const eased = 1 - Math.pow(1 - t, 3);
      setValue(target * eased);
      if (t < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);
  return Math.round(value * 10) / 10;
}

function KPI({ label, value, hint, accent }: { label: string; value: React.ReactNode; hint?: string; accent: 'blue' | 'green' | 'amber' | 'red' | 'slate' }) {
  return (
    <motion.div
      className={`px-kpi px-kpi-${accent}`}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25 }}
    >
      <div className="px-kpi-strip" />
      <span className="px-kpi-label">{label}</span>
      <strong className="px-kpi-value">{value}</strong>
      {hint && <small className="px-kpi-hint">{hint}</small>}
    </motion.div>
  );
}

export default function ProxmoxNodeKpiStrip({ status, loading }: Props) {
  const cpuPct = status.maxcpu ? (status.cpu ?? 0) / status.maxcpu * 100 : 0;
  const memPct = status.maxmem ? (status.mem ?? 0) / status.maxmem * 100 : 0;
  const diskPct = status.maxdisk ? (status.disk ?? 0) / status.maxdisk * 100 : 0;
  const load1 = readString(status.loadavg?.[0]) || '—';

  const load = useCountUp(Number(load1) || 0);

  const items = [
    { label: 'CPU', value: formatPercent(cpuPct), hint: `${status.cpu ?? 0}/${status.maxcpu ?? 0} cores`, accent: 'blue' as const },
    { label: 'Memory', value: formatBytes(status.mem ?? 0), hint: `of ${formatBytes(status.maxmem ?? 0)} (${Math.round(memPct)}%)`, accent: 'green' as const },
    { label: 'Disk', value: formatBytes(status.disk ?? 0), hint: `of ${formatBytes(status.maxdisk ?? 0)} (${Math.round(diskPct)}%)`, accent: 'amber' as const },
    { label: 'Load (1m)', value: load, hint: 'system load average', accent: 'slate' as const },
    { label: 'Uptime', value: formatUptime(status.uptime), accent: 'green' as const },
  ];

  if (loading) {
    return (
      <div className="px-kpi-strip-row" aria-busy="true">
        {items.map((item) => (
          <div key={item.label} className={`px-kpi px-kpi-${item.accent} px-kpi-loading`}>
            <div className="px-kpi-strip" />
            <span className="px-kpi-label">{item.label}</span>
            <strong className="px-kpi-value">—</strong>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="px-kpi-strip-row">
      {items.map((item) => <KPI key={item.label} {...item} />)}
    </div>
  );
}