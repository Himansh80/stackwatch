import { motion, shimmer, useReducedMotion } from '../../lib/motion';

interface SkeletonRowProps {
  /** Number of skeleton blocks per row. */
  columns: number;
  /** Number of rows to render. Defaults to 1. */
  rows?: number;
}

/**
 * SkeletonRow — placeholder rendered while a list (hosts, tokens,
 * settings) is fetching. One row = `columns` skeleton blocks. Use
 * multiple rows when the table has more than one entry to suggest
 * the eventual layout.
 *
 * Visual contract (002-polish spec §"Component additions"):
 *   - `--surface-2` background, `--surface-3` highlight via gradient
 *   - 16px-tall blocks, radius `--radius-sm`
 *   - shimmer motion variant (1.4s loop) when motion is allowed
 *
 * Honors `prefers-reduced-motion`: the shimmer keyframes never run
 * when the user has reduced motion enabled (the framer-motion variant
 * respects reduced motion automatically + the CSS `prefers-reduced-motion`
 * media query zeros any animation in dashboard.css).
 */
export default function SkeletonRow({ columns, rows = 1 }: SkeletonRowProps) {
  const reduce = useReducedMotion();
  const blocks = Array.from({ length: columns }, (_, i) => i);
  return (
    <>
      {Array.from({ length: rows }, (_, rowIdx) => (
        <div className="skeleton-row" key={`row-${rowIdx}`} aria-hidden="true">
          {blocks.map((colIdx) => (
            <motion.div
              key={`block-${rowIdx}-${colIdx}`}
              className="skeleton-row-block"
              variants={reduce ? undefined : shimmer}
              animate={reduce ? undefined : 'animate'}
            />
          ))}
        </div>
      ))}
    </>
  );
}
