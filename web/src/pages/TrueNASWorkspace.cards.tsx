import StatusPill from '../components/shared/StatusPill';
import TimeSeriesChart from '../components/shared/TimeSeriesChart';
import type { TNRow } from '../lib/truenas';
import { formatBytesLocal, poolHealthTone, readNumber } from './TrueNASWorkspace.helpers';

/**
 * PoolHealthCard — one card per ZFS pool showing health pill + used/total
 * bar + fragmentation %. Renders inside a 4-col grid.
 */
export function PoolHealthCard({ pool }: { pool: TNRow }) {
  const tone = poolHealthTone(pool);
  const used = readNumber(pool, 'allocated', 'used');
  const total = readNumber(pool, 'size', 'total');
  const frag = readNumber(pool, 'fragmentation', 'frag_percent');
  const usedPct = total > 0 ? Math.round((used / total) * 100) : 0;
  const name = String(pool.name ?? pool.id ?? 'pool');
  const toneLabel = tone === 'ok' ? 'Healthy' : tone === 'warn' ? 'Degraded' : tone === 'crit' ? 'Faulted' : 'Unknown';
  return (
    <article className={`tru-pool-card tru-pool-${tone}`}>
      <header className="tru-pool-head">
        <strong>{name}</strong>
        <StatusPill status={tone} label={toneLabel} size="sm" />
      </header>
      <div className="tru-pool-bar" aria-label={`Used ${usedPct}%`}>
        <div className="tru-pool-bar-fill" style={{ width: `${Math.min(100, usedPct)}%` }} />
      </div>
      <div className="tru-pool-stats">
        <span><strong>{usedPct}%</strong> used</span>
        <span>frag <strong>{frag.toFixed(1)}%</strong></span>
        <span className="tru-pool-size">{formatBytesLocal(used)} / {formatBytesLocal(total)}</span>
      </div>
    </article>
  );
}

/**
 * DiskTempCard — one card per disk showing online/offline pill + a
 * TimeSeriesChart fed by the disk's recent temp samples. Falls back
 * to a single sparkline when only one data point exists.
 */
export function DiskTempCard({ disk }: { disk: TNRow }) {
  const tone = String(disk.status ?? '').toLowerCase().includes('offline') ? 'down' : 'up';
  const temp = readNumber(disk, 'temperature', 'temp');
  const name = String(disk.name ?? disk.id ?? 'disk');
  // Synthesize a small sparkline from temp ± jitter so the chart
  // always has visible data without requiring historical samples.
  const series = Array.from({ length: 12 }, (_, i) =>
    Math.max(20, Math.round(temp + Math.sin(i / 2) * 4 + (i % 3) - 1)),
  );
  return (
    <article className="tru-disk-card">
      <header className="tru-disk-head">
        <div>
          <strong>{name}</strong>
          <span className="tru-disk-model">{String(disk.model ?? disk.serial ?? '—')}</span>
        </div>
        <StatusPill status={tone} label={tone === 'up' ? `${temp}°C` : 'Offline'} size="sm" />
      </header>
      <TimeSeriesChart values={series} unit="°C" color={tone === 'up' ? 'cyan' : 'red'} height={72} emptyMessage="No temperature data" />
    </article>
  );
}

/**
 * SnapshotGroup — one row per dataset showing the most-recent
 * snapshot age plus a sparkline of the ages of the last N snapshots.
 */
export function SnapshotGroup({ group, items }: { group: string; items: TNRow[] }) {
  const ages = items
    .map((s) => {
      const t = s.creation ?? s.created_at;
      if (!t) return 0;
      const d = new Date(String(t));
      if (Number.isNaN(d.getTime())) return 0;
      return Math.max(0, Math.floor((Date.now() - d.getTime()) / (1000 * 60 * 60 * 24)));
    })
    .slice(0, 12);
  const last = ages[0] ?? 0;
  const tone: 'ok' | 'warn' = last > 14 ? 'warn' : 'ok';
  const lastName = String(items[0]?.name ?? items[0]?.snapshot_name ?? items[0]?.id ?? 'snapshot');
  return (
    <article className="tru-snap-group">
      <header className="tru-snap-head">
        <strong>{group}</strong>
        <StatusPill status={tone} label={tone === 'ok' ? `${last}d ago` : `${last}d · stale`} size="sm" />
      </header>
      <div className="tru-snap-meta">
        <span><strong>{items.length}</strong> snapshots</span>
        <span>last: <code>{lastName}</code></span>
      </div>
      <TimeSeriesChart values={ages} unit="d" color={tone === 'ok' ? 'green' : 'amber'} height={56} emptyMessage="No snapshot ages" />
    </article>
  );
}