import { motion, shimmer, useReducedMotion } from '../../lib/motion';

/**
 * Skeleton loader card — used while data is loading on initial mount.
 * Mirrors the KpiCard layout so the page doesn't jump when data arrives.
 *
 * Uses the shared `shimmer` motion variant for the block pulse, so the
 * loading state stays in sync with `SkeletonRow` everywhere else. The
 * shimmer is suppressed when the user prefers reduced motion.
 */
export default function SkeletonCard() {
  const reduce = useReducedMotion();
  return (
    <motion.article
      className="dash-metric dash-metric-skeleton"
      variants={reduce ? undefined : shimmer}
      animate={reduce ? undefined : 'animate'}
    >
      <div className="dash-metric-top">
        <span className="dash-metric-icon dash-skel-line" style={{ width: 18, height: 18, display: 'inline-block' }} />
        <span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} />
      </div>
      <strong className="dash-skel-line" style={{ width: 50, height: 28 }} />
      <span className="dash-metric-hint dash-skel-line" style={{ width: 120, height: 10 }} />
    </motion.article>
  );
}
