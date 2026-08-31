import { ReactNode } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

interface EmptyStateAction {
  label: string;
  onClick: () => void;
}

export interface EmptyStateProps {
  /** Inline SVG / Lucide-style icon node. Rendered inside a 96×96
   *  accent-tinted bubble so users see it as an illustration, not a glyph. */
  illustration: ReactNode;
  /** 24px / 600-weight headline. */
  headline: string;
  /** Optional 14px / muted subhead. */
  subhead?: string;
  /** Optional single primary CTA. One CTA is the spec — more than
   *  one turns into a decision-tree, which is the empty-state
   *  anti-pattern. */
  cta?: EmptyStateAction;
}

/**
 * EmptyState — shared zero-data placeholder used by every page that
 * renders a list (Dashboard hosts/workloads, Profile, Settings, Billing,
 * TrueNASWorkspace). One illustration, one headline, one subhead, one CTA.
 *
 * Visual contract (002-polish spec §"Component additions"):
 *   - Centered, max-width 480px
 *   - 96×96 illustration bubble with `var(--accent-soft)` background
 *   - 24px / 600 headline, 14px / muted subhead
 *   - Primary CTA button (accent fill, tokens only)
 *
 * Honors `prefers-reduced-motion` via `useReducedMotion()` so the
 * CTA spring snaps instead of animating when the user has reduced
 * motion enabled.
 */
export default function EmptyState({ illustration, headline, subhead, cta }: EmptyStateProps) {
  const reduce = useReducedMotion();
  return (
    <div className="empty-state" role="status" aria-live="polite">
      <div className="empty-state-illustration" aria-hidden="true">
        {illustration}
      </div>
      <h3 className="empty-state-headline">{headline}</h3>
      {subhead ? <p className="empty-state-subhead">{subhead}</p> : null}
      {cta ? (
        <motion.button
          type="button"
          className="empty-state-cta"
          onClick={cta.onClick}
          whileHover={reduce ? undefined : buttonSpring.whileHover}
          whileTap={reduce ? undefined : buttonSpring.whileTap}
          transition={buttonSpring.transition}
        >
          {cta.label}
        </motion.button>
      ) : null}
    </div>
  );
}
