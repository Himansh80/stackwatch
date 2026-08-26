import EmptyState from '../../components/shared/EmptyState';

export interface LogPattern {
  service: string;
  pattern: string;
  sample_count: number;
  last_seen_at: string;
}

interface LogPatternsTabProps {
  patterns: LogPattern[];
}

export default function LogPatternsTab({ patterns }: LogPatternsTabProps) {
  return (
    <section className="dash-section">
      <span className="dash-eyebrow">Detected</span>
      <h2 className="dash-section-title">Log patterns (last 24h)</h2>
      {patterns.length === 0 ? (
        <EmptyState
          illustration={<span style={{ fontSize: 36 }}>◷</span>}
          headline="No patterns detected"
          subhead="Once log events flow through StackWatch, repeated message prefixes will aggregate here."
        />
      ) : (
        <table className="dash-table">
          <thead>
            <tr><th>Service</th><th>Pattern</th><th>Count</th><th>Last seen</th></tr>
          </thead>
          <tbody>
            {patterns.map((p, idx) => (
              <tr key={`${p.service}-${idx}`}>
                <td><strong>{p.service}</strong></td>
                <td><code>{p.pattern}</code></td>
                <td>{p.sample_count}</td>
                <td>{p.last_seen_at ? new Date(p.last_seen_at).toLocaleString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
