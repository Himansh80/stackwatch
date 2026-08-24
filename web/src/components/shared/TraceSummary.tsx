import type { KeyboardEvent } from 'react';
import { motion, kpiEnter, useReducedMotion } from '../../lib/motion';

/**
 * TraceSummary — service node header used on the APM page grid.
 *
 * Shows the service name + language pill, plus three mini KPI tiles
 * (req rate / error rate / p95 latency). Visual contract (from
 * 003-tier7-datadog-parity §"TraceSummary.tsx"):
 *
 *   - radius `var(--radius)`, surface `var(--surface)`, border `--border`
 *   - language pill uses `--accent-soft` background + `--accent` text
 *   - 3 mini-tile grid: req rate, error rate, p95 (each ~equal width)
 *   - error_rate ≥ 5% paints its tile with `--red-soft` + `--red`
 *   - p95 ≥ 500ms paints its tile with `--amber-soft` + `--amber`
 *
 * Motion: parent should pass `kpiStagger` and this card uses `kpiEnter`
 * so it animates in on the strip. Honors `prefers-reduced-motion` via
 * `useReducedMotion()` so the entry snaps when the user prefers it.
 */

export interface TraceSummaryService {
  id: string;
  name: string;
  language: string;
  framework: string;
  request_rate: number;
  error_rate: number; // 0..1
  p95_latency: number; // ms
}

interface TraceSummaryProps {
  service: TraceSummaryService;
  onClick?: (service: TraceSummaryService) => void;
}

function formatRate(rate: number): string {
  if (!Number.isFinite(rate)) return '—';
  if (rate >= 100) return `${Math.round(rate)}/s`;
  if (rate >= 10) return `${rate.toFixed(1)}/s`;
  if (rate >= 1) return `${rate.toFixed(2)}/s`;
  return `${(rate * 60).toFixed(1)}/m`;
}

function formatP95(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—';
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
  return `${ms.toFixed(0)}ms`;
}

export default function TraceSummary({ service, onClick }: TraceSummaryProps) {
  const reduce = useReducedMotion();
  const errPct = (service.error_rate || 0) * 100;
  const errHot = errPct >= 5;
  const p95Hot = service.p95_latency >= 500;
  const handleClick = () => {
    if (onClick) onClick(service);
  };
  const handleKey = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!onClick) return;
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onClick(service);
    }
  };
  return (
    <motion.article
      className="dash-card apm-summary"
      variants={kpiEnter}
      whileHover={reduce ? undefined : { y: -2, transition: { duration: 0.15 } }}
      onClick={onClick ? handleClick : undefined}
      onKeyDown={handleKey}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      style={{ cursor: onClick ? 'pointer' : 'default' }}
    >
      <header className="apm-summary-header">
        <strong className="apm-summary-name">{service.name}</strong>
        {service.language ? (
          <span className="apm-pill apm-pill-language">{service.language}</span>
        ) : null}
      </header>
      <div className="apm-summary-tiles">
        <div className="apm-mini-tile" title="Request rate (last 5m)">
          <span className="apm-mini-label">REQ/S</span>
          <span className="apm-mini-value">{formatRate(service.request_rate)}</span>
        </div>
        <div
          className={`apm-mini-tile ${errHot ? 'apm-mini-tile-warn' : ''}`}
          title="Error rate (last 5m)"
        >
          <span className="apm-mini-label">ERR %</span>
          <span className="apm-mini-value">{errPct.toFixed(2)}%</span>
        </div>
        <div
          className={`apm-mini-tile ${p95Hot ? 'apm-mini-tile-warn' : ''}`}
          title="p95 latency (last 5m)"
        >
          <span className="apm-mini-label">P95</span>
          <span className="apm-mini-value">{formatP95(service.p95_latency)}</span>
        </div>
      </div>
      {service.framework ? (
        <span className="apm-summary-footer">{service.framework}</span>
      ) : null}
    </motion.article>
  );
}
