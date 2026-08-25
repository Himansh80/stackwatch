/**
 * Tier 14 Phase 14.1 — Status filter chips + search bar.
 * Click a chip to filter the VM list to that status. "All" clears.
 */
import { motion } from 'framer-motion';

export type StatusFilter = 'all' | 'running' | 'stopped' | 'paused';

interface Props {
  search: string;
  onSearchChange: (value: string) => void;
  status: StatusFilter;
  onStatusChange: (value: StatusFilter) => void;
  counts?: Partial<Record<StatusFilter, number>>;
}

const chips: Array<{ id: StatusFilter; label: string; accent: string }> = [
  { id: 'all', label: 'All', accent: 'slate' },
  { id: 'running', label: 'Running', accent: 'green' },
  { id: 'stopped', label: 'Stopped', accent: 'slate' },
  { id: 'paused', label: 'Paused', accent: 'amber' },
];

export default function ProxmoxFilterBar({ search, onSearchChange, status, onStatusChange, counts }: Props) {
  return (
    <div className="px-filterbar">
      <input
        type="search"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search by name, VMID, or IP…"
        className="px-filterbar-search"
        aria-label="Search VMs"
      />
      <div className="px-filterbar-chips" role="radiogroup" aria-label="Filter by status">
        {chips.map((chip) => {
          const active = status === chip.id;
          const count = counts?.[chip.id];
          return (
            <motion.button
              key={chip.id}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => onStatusChange(chip.id)}
              whileHover={{ scale: 1.03 }}
              whileTap={{ scale: 0.96 }}
              className={`px-chip px-chip-${chip.accent} ${active ? 'px-chip-active' : ''}`}
            >
              <span>{chip.label}</span>
              {typeof count === 'number' && <span className="px-chip-count">{count}</span>}
            </motion.button>
          );
        })}
      </div>
    </div>
  );
}