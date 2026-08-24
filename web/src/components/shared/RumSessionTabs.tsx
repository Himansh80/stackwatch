import ResourceWaterfall, { WaterfallResource } from './ResourceWaterfall';

/**
 * RumSessionTabs — the four tab bodies for the session detail page.
 *
 * Kept as a separate file from RumSessionPage so the page stays
 * under the 400-LOC modularity cap. The page imports these tab
 * renderers and picks one based on the active tab state.
 */

export interface RumVital {
  id: string;
  url: string;
  metric: string;
  value: number;
  rating: string;
  ts: string;
}

export interface RumInteraction {
  id: string;
  action: string;
  target: string;
  duration_ms: number;
  ts: string;
}

export interface RumLongTask {
  id: string;
  duration_ms: number;
  ts: string;
}

const RATING_TONE: Record<string, 'good' | 'neutral' | 'warn' | 'bad'> = {
  good: 'good',
  'needs-improvement': 'warn',
  poor: 'bad',
};

function formatVital(metric: string, value: number): string {
  if (!Number.isFinite(value)) return '—';
  if (metric === 'cls') return value.toFixed(3);
  if (metric === 'ttfb' || metric === 'fcp' || metric === 'lcp' || metric === 'fid') {
    return `${Math.round(value)}ms`;
  }
  return String(value);
}

export function VitalsTab({ vitals }: { vitals: RumVital[] }) {
  if (!vitals.length) {
    return <p className="dash-section-lede">No web vitals reported for this session.</p>;
  }
  return (
    <table className="sw-table">
      <thead>
        <tr>
          <th>Metric</th>
          <th>Value</th>
          <th>Rating</th>
          <th>URL</th>
          <th>Time</th>
        </tr>
      </thead>
      <tbody>
        {vitals.map((v) => {
          const tone = RATING_TONE[v.rating] || 'neutral';
          return (
            <tr key={v.id}>
              <td><code>{v.metric.toUpperCase()}</code></td>
              <td>{formatVital(v.metric, v.value)}</td>
              <td><span className={`sw-status-${tone}`}>{v.rating || '—'}</span></td>
              <td title={v.url}>{v.url.slice(0, 48)}</td>
              <td>{v.ts ? new Date(v.ts).toLocaleString() : '—'}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

export function InteractionsTab({ interactions }: { interactions: RumInteraction[] }) {
  if (!interactions.length) {
    return <p className="dash-section-lede">No interactions captured for this session.</p>;
  }
  return (
    <table className="sw-table">
      <thead>
        <tr>
          <th>Action</th>
          <th>Target</th>
          <th>Duration</th>
          <th>Time</th>
        </tr>
      </thead>
      <tbody>
        {interactions.map((i) => (
          <tr key={i.id}>
            <td><code>{i.action}</code></td>
            <td title={i.target}>{i.target ? i.target.slice(0, 48) : '—'}</td>
            <td>{i.duration_ms || 0}ms</td>
            <td>{i.ts ? new Date(i.ts).toLocaleString() : '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function ResourcesTab({ resources }: { resources: WaterfallResource[] }) {
  if (!resources.length) {
    return (
      <p className="dash-section-lede">
        No network requests were sent to /api/v1/rum/resource-timings for this session.
      </p>
    );
  }
  return <ResourceWaterfall resources={resources} />;
}

export function ErrorsTab({ errorCount }: { errorCount: number }) {
  return (
    <p className="dash-section-lede">
      Errors are deduped globally — see the RUM overview for the full list.
      This session had {errorCount} error(s).
    </p>
  );
}
