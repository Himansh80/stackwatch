/**
 * Skeleton loader card — used while data is loading on initial mount.
 * Mirrors the MetricCard layout so the page doesn't jump when data
 * arrives.
 */
export default function SkeletonCard() {
  return (
    <article className="dash-metric dash-metric-skeleton">
      <div className="dash-metric-top">
        <span className="dash-metric-icon dash-skel-line" style={{ width: 18, height: 18, display: 'inline-block' }} />
        <span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} />
      </div>
      <strong className="dash-skel-line" style={{ width: 50, height: 28 }} />
      <span className="dash-metric-hint dash-skel-line" style={{ width: 120, height: 10 }} />
    </article>
  );
}
