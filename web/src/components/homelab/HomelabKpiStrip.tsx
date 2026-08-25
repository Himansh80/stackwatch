import { motion, kpiEnter, kpiStagger, useReducedMotion } from '../../lib/motion';
import KpiCard from '../shared/KpiCard';

/**
 * HomelabKpiStrip — Tier 10 Phase 1 (H1 — Widget Framework).
 *
 * The top 4-card KPI strip on the HomelabPage header. Each card is
 * clickable and navigates to the relevant tab (Phase 2+ will wire
 * real data; for now Phase 1 renders placeholder values so the
 * visual is locked in and the wiring is mechanical later).
 *
 * Cards (left → right):
 *   1. Services   — pinned services count  (→ /homelab#services, Phase 2)
 *   2. Notes      — notes count            (→ /homelab#notes,    Phase 3)
 *   3. Todos      — todos due-soon count   (→ /homelab#todos,    Phase 3)
 *   4. RSS unread — RSS unread item count  (→ /homelab#rss,      Phase 8)
 *
 * Motion: kpiStagger on the wrapper; kpiEnter on each card. The
 * click-to-navigate handler receives the target tab id so the
 * parent HomelabPage can switch its active tab without a full
 * route change.
 *
 * Why this lives in a separate file (Phase 1 file budget ~150 LOC):
 *   The KpiCard component is reusable (already used by Intelligence,
 *   Dashboard, etc.). This file only owns the 4 cards + their
 *   navigation handlers — keeps the parent page focused on tab
 *   routing + layout management.
 */
interface HomelabKpiStripProps {
  /** Pinned-services count (Phase 2 wire-up). For Phase 1 this is 0. */
  pinnedServicesCount: number;
  /** Notes count (Phase 3 wire-up). For Phase 1 this is 0. */
  notesCount: number;
  /** Todos-due-soon count (Phase 3 wire-up). For Phase 1 this is 0. */
  todosDueSoonCount: number;
  /** RSS unread count (Phase 8 wire-up). For Phase 1 this is 0. */
  rssUnreadCount: number;
  /** Handler called when a card is clicked — receives the target tab id. */
  onNavigate: (tab: 'overview' | 'services' | 'notes' | 'todos' | 'rss') => void;
}

export default function HomelabKpiStrip({
  pinnedServicesCount,
  notesCount,
  todosDueSoonCount,
  rssUnreadCount,
  onNavigate,
}: HomelabKpiStripProps) {
  const reduce = useReducedMotion();

  // Each card: number is the headline value; accent reflects status
  // (everything is green=up at rest, amber when count > 0 indicates
  // attention-worthy, etc.). Phase 1 ships all zeros so all cards
  // render the "neutral" green baseline; future phases replace
  // these with real counts.
  const cards: Array<{
    label: string;
    value: number;
    tab: 'services' | 'notes' | 'todos' | 'rss';
    accent: 'cyan' | 'indigo' | 'amber' | 'violet';
    status: 'up' | 'crit' | 'neutral';
  }> = [
    {
      label: 'Services pinned',
      value: pinnedServicesCount,
      tab: 'services',
      accent: 'cyan',
      status: pinnedServicesCount > 0 ? 'up' : 'neutral',
    },
    {
      label: 'Notes',
      value: notesCount,
      tab: 'notes',
      accent: 'indigo',
      status: 'neutral',
    },
    {
      label: 'Todos due soon',
      value: todosDueSoonCount,
      tab: 'todos',
      accent: 'amber',
      status: todosDueSoonCount > 0 ? 'crit' : 'neutral',
    },
    {
      label: 'RSS unread',
      value: rssUnreadCount,
      tab: 'rss',
      accent: 'violet',
      status: rssUnreadCount > 0 ? 'up' : 'neutral',
    },
  ];

  return (
    <motion.div
      className="dash-metric-strip"
      initial="hidden"
      animate="show"
      variants={kpiStagger}
    >
      {cards.map((c) => (
        <motion.button
          key={c.label}
          type="button"
          className="homelab-kpi-button"
          onClick={() => onNavigate(c.tab)}
          variants={kpiEnter}
          whileHover={reduce ? undefined : { y: -2 }}
          whileTap={reduce ? undefined : { scale: 0.98 }}
          title={`Open ${c.label}`}
          aria-label={`Open ${c.label}`}
          style={{ background: 'transparent', border: 'none', padding: 0, cursor: 'pointer' }}
        >
          <KpiCard label={c.label} value={c.value} status={c.status} accent={c.accent} />
        </motion.button>
      ))}
    </motion.div>
  );
}