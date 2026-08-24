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
  motion as _motionBase,
  useReducedMotion as _useReducedMotionBase,
  type Variants,
} from 'framer-motion';
import { useEffect, useRef, useState } from 'react';

// Re-export common framer-motion pieces so pages do not need
// to import framer-motion directly. Keeps the noise to one import. ----
export const motion = _motionBase;
export const useReducedMotion = _useReducedMotionBase;
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
