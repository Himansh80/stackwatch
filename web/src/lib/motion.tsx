// Shared motion tokens for StackWatch public/auth pages. One source of
// truth so Landing, Login, Signup, ForgotPassword, and ResetPassword
// stay in sync without copy-paste drift.
//
// Usage:
//   import { motion, fadeUp, EASE_OUT, cardSpring, CountUp, LiveDot }
//     from '../lib/motion';
//   <motion.div variants={fadeUp}>...</motion.div>
//   <motion.button whileHover={cardSpring.whileHover}>...</motion.button>
//
// Honors prefers-reduced-motion via useReducedMotion() at every call site.
// Pair with the CSS block in styles.css that zeroes out hover transitions
// when reduced-motion is requested.
import {
  AnimatePresence,
  motion as _motionBase,
  useReducedMotion as _useReducedMotionBase,
  type Variants,
} from 'framer-motion';
import { useEffect, useRef, useState } from 'react';

// Re-export common framer-motion pieces so pages do not need
// to import framer-motion directly. Keeps the noise to one import. ----
export const motion = _motionBase;
export const useReducedMotion = _useReducedMotionBase;
export { AnimatePresence };
export type { Variants };

// ---------- Tokens -----------------------------------------------------
// Cubic ease-out: gentle in, decisive finish. Same on all three pages.
export const EASE_OUT: [number, number, number, number] = [0.16, 1, 0.3, 1];

// Card entrance: 8-12 px lift + opacity, 320-420 ms. Single shared feel.
export const cardEntrance: Variants = {
  hidden: { opacity: 0, y: 10 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: EASE_OUT } },
};

// Generic section item — eyebrow/h1/lede/cta on a hero, etc.
export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.42, ease: EASE_OUT } },
};

// Stagger factory. delayChildren = wait before the first child,
// staggerChildren = gap between siblings.
export const stagger = (delayChildren = 0, staggerChildren = 0.07): Variants => ({
  hidden: {},
  show: { transition: { delayChildren, staggerChildren } },
});

// ---------- Spring primitives ----------------------------------------
// Use as <motion.div whileHover={cardLift}> or <motion.button
//   whileHover={buttonSpring.whileHover} whileTap={buttonSpring.whileTap}>.
// The spring math is identical everywhere so buttons feel the same across
// pages.
export const cardLift = {
  y: -3,
  transition: { type: 'spring' as const, stiffness: 380, damping: 26 },
};
export const buttonSpring = {
  whileHover: { y: -1 },
  whileTap: { y: 0, scale: 0.98 },
  transition: { type: 'spring' as const, stiffness: 400, damping: 28 },
};

// ---------- KPI counter (animates from 0 -> n over durationMs) -------
// rAF + cubic ease-out. When reduced-motion is on, snaps to final value
// in one frame (no count-up).
export function CountUp({ value, durationMs = 700 }: { value: number; durationMs?: number }) {
  const reduce = useReducedMotion();
  const [display, setDisplay] = useState(reduce ? value : 0);
  const raf = useRef<number | null>(null);
  useEffect(() => {
    if (reduce) {
      setDisplay(value);
      return;
    }
    const start = performance.now();
    const ease = (t: number) => 1 - Math.pow(1 - t, 3);
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / durationMs);
      setDisplay(Math.round(value * ease(t)));
      if (t < 1) raf.current = requestAnimationFrame(tick);
    };
    raf.current = requestAnimationFrame(tick);
    return () => {
      if (raf.current != null) cancelAnimationFrame(raf.current);
    };
  }, [value, durationMs, reduce]);
  return <>{display}</>;
}

// ---------- Live dot ---------------------------------------------------
// Re-renders once per second so a CSS pulse animation looks alive.
// The pulse itself is a CSS keyframe defined in styles.css.
export function LiveDot() {
  const [, force] = useState(0);
  useEffect(() => {
    const id = window.setInterval(() => force((n) => n + 1), 1000);
    return () => window.clearInterval(id);
  }, []);
  return <span className="live-dot" aria-hidden="true" />;
}

// ---------- Children helper for form sections -------------------------
// Some pages want each form row to stagger in after the card. Use:
//   <motion.div variants={staggerFormRows}>...</motion.div>
//   <motion.label variants={formRow}>...</motion.label>
export const formRow: Variants = {
  hidden: { opacity: 0, y: 6 },
  show: { opacity: 1, y: 0, transition: { duration: 0.28, ease: EASE_OUT } },
};
export const staggerFormRows: Variants = stagger(0.1, 0.05);

// ---------- Datadog-grade motion library (Phase 1 of 002-polish) -------
// kpiEnter / kpiStagger: KPI strip entrance with a 50ms inter-card gap.
// Pair as <motion.div variants={kpiStagger}> wrapping N cards that each
// use <motion.div variants={kpiEnter}>. The 50ms stagger keeps the eye
// moving without making the strip feel slow.
export const kpiEnter: Variants = {
  hidden: { opacity: 0, y: 8 },
  show: { opacity: 1, y: 0, transition: { duration: 0.32, ease: EASE_OUT } },
};
export const kpiStagger: Variants = stagger(0.05, 0.06);

// statusPulse: a plain animate prop (not Variants) used directly on the
// StatusPill dot when the host is DOWN / CRIT. Opacity dips to 0.55 and
// returns, looping forever — Datadog's "the room is on fire" signal.
export const statusPulse = {
  animate: { opacity: [1, 0.55, 1] },
  transition: { duration: 1.5, repeat: Infinity, ease: 'easeInOut' as const },
};

// sparklineDraw: stroke-dashoffset draw-in for SVG <path>. Apply via
// <motion.path variants={sparklineDraw} /> on the chart line; the
// initial pathLength:0 means "fully hidden", show:"1" paints it in.
export const sparklineDraw: Variants = {
  hidden: { pathLength: 0, opacity: 0 },
  show: { pathLength: 1, opacity: 1, transition: { duration: 0.6, ease: EASE_OUT } },
};

// pageEnter: every route wraps its top-level container with
// <motion.div initial="hidden" animate="show" variants={pageEnter}>.
// 240ms matches the --motion-page CSS token so JS + CSS feel identical.
export const pageEnter: Variants = {
  hidden: { opacity: 0, y: 4 },
  show: { opacity: 1, y: 0, transition: { duration: 0.24, ease: EASE_OUT } },
};

// paletteEnter: CommandPalette entrance — fade + a subtle 0.98 → 1 scale
// pop. Exit reverses the scale for a Linear ⌘K feel.
export const paletteEnter: Variants = {
  hidden: { opacity: 0, scale: 0.98 },
  show: { opacity: 1, scale: 1, transition: { duration: 0.18, ease: EASE_OUT } },
  exit: { opacity: 0, scale: 0.98, transition: { duration: 0.12 } },
};

// shimmer: skeleton-row shimmer. backgroundPosition keyframes drive the
// traveling highlight; uses CSS variables for the actual gradient stops.
export const shimmer: Variants = {
  animate: {
    backgroundPosition: ['0% 50%', '100% 50%', '0% 50%'],
    transition: { duration: 1.4, repeat: Infinity, ease: 'linear' as const },
  },
};
