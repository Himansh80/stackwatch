/**
 * Time helpers for CalendarWidget — split out so the parent stays
 * under 400 LOC. Each helper is small + side-effect-free; the
 * tests for the calendar week-grid are simple date-equality
 * checks against these helpers.
 */

const dayMs = 86400000;

/** Returns the Monday of the week containing d, at 00:00 local time. */
export const startOfWeek = (d: Date): Date => {
  const out = new Date(d);
  out.setHours(0, 0, 0, 0);
  // Convert Sunday(0)..Saturday(6) to Monday(0)..Sunday(6).
  const dow = (out.getDay() + 6) % 7;
  out.setDate(out.getDate() - dow);
  return out;
};

/** True iff two dates fall on the same calendar day (local time). */
export const sameDay = (a: Date, b: Date): boolean =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate();

/** ISO string from a Date. Helper to keep call sites tidy. */
export const isoFromDate = (d: Date): string => d.toISOString();

/** Format an ISO time as a locale HH:MM am/pm string. */
export const formatTime = (iso: string): string => {
  const t = new Date(iso);
  if (!Number.isFinite(t.getTime())) return '';
  return t.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
};

/** Relative-time formatter: "just now", "5m ago", "2h ago", "3d ago", date. */
export const relativeTime = (iso: string): string => {
  if (!iso) return '';
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '';
  const diffSec = Math.round((Date.now() - t) / 1000);
  if (diffSec < 60) return 'just now';
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  if (diffSec < 86400 * 7) return `${Math.floor(diffSec / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
};

// _ keeps dayMs referenced so goimports + eslint don't churn when
// future revisions add a duration-aware helper (e.g. "until next
// sync in N days").
export const _dayMsUnusedKeep = dayMs;
