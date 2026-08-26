/**
 * Tier 14 Phase 14.1 — KPI strip for the VM list page.
 * Shows: total / running / stopped / paused / cluster CPU / cluster RAM.
 * Animated count-up + color accent stripes (Datadog style).
 */
import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import type { ProxmoxResource } from '../../lib/proxmox';
import { formatPercent, readString } from '../../lib/proxmox';

interface Props {
  resources: ProxmoxResource[];
  loading?: boolean;
}

interface KPI {
  label: string;
  value: number | string;
  hint?: string;
  accent: 'blue' | 'green' | 'red' | 'amber' | 'slate';
}

function countByStatus(rows: ProxmoxResource[]): Record<string, number> {
  const counts = { running: 0, stopped: 0, paused: 0, other: 0 };
  for (const row of rows) {
    const status = readString(row.status).toLowerCase();
    if (status === 'running') counts.running += 1;
    else if (status === 'stopped') counts.stopped += 1;
    else if (status === 'paused') counts.paused += 1;
    else counts.other += 1;
  }
  return counts;
}

function clusterUsage(rows: ProxmoxResource[]): { cpu: number; ram: number; ramTotal: number } {
  // Proxmox /vms returns cpu_usage (fraction 0-1) and mem_used/mem_total
  // (bytes). Fall back to legacy /cluster/resources keys for old code paths.
  let cpu = 0;
  let ram = 0;
  let ramTotal = 0;
  let n = 0;
  for (const row of rows) {
    const c = typeof row.cpu_usage === 'number' ? row.cpu_usage
      : typeof row.cpu === 'number' ? row.cpu : 0;
    if (c > 0) { cpu += c; n += 1; }
    const m = typeof row.mem_used === 'number' ? row.mem_used
      : typeof row.mem === 'number' ? row.mem : 0;
    if (m > 0) ram += m;
    const mt = typeof row.mem_total === 'number' ? row.mem_total
      : typeof row.maxmem === 'number' ? row.maxmem : 0;
    if (mt > 0) ramTotal += mt;
  }
  return { cpu: n ? cpu / n : 0, ram, ramTotal };
}

/**
 * useCountUp — animates a numeric value from 0 to target over 600ms.
 */
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

function KPI({ label, value, hint, accent }: KPI) {
  return (
    <motion.div
      className={`px-kpi px-kpi-${accent}`}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: 'easeOut' }}
    >
      <div className="px-kpi-strip" />
      <span className="px-kpi-label">{label}</span>
      <strong className="px-kpi-value">
        {typeof value === 'number' ? value : value}
      </strong>
      {hint && <small className="px-kpi-hint">{hint}</small>}
    </motion.div>
  );
}

export default function ProxmoxKpiStrip({ resources, loading }: Props) {
  const counts = countByStatus(resources);
  const usage = clusterUsage(resources);

  const total = useCountUp(resources.length);
  const running = useCountUp(counts.running);
  const stopped = useCountUp(counts.stopped);
  const paused = useCountUp(counts.paused);
  const cpu = useCountUp(usage.cpu);
  const ramPct = usage.ramTotal ? Math.round((usage.ram / usage.ramTotal) * 100) : 0;
  const ram = useCountUp(ramPct);

  const items: KPI[] = [
    { label: 'Total', value: total, hint: 'VMs + LXC', accent: 'slate' },
    { label: 'Running', value: running, accent: 'green' },
    { label: 'Stopped', value: stopped, accent: 'slate' },
    { label: 'Paused', value: paused, accent: 'amber' },
    { label: 'Cluster CPU', value: formatPercent(cpu), hint: 'avg across guests', accent: 'blue' },
    { label: 'Cluster RAM', value: `${ram}%`, hint: 'used / total', accent: 'blue' },
  ];

  if (loading && resources.length === 0) {
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
      {items.map((item) => (
        <KPI key={item.label} {...item} />
      ))}
    </div>
  );
}