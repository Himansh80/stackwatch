/**
 * Tier 14 Phase 14.2 — Console tab placeholder.
 * Real noVNC + xterm.js console ships in Phase 14.3.
 */
import { motion } from 'framer-motion';

export default function ProxmoxDetailConsole() {
  return (
    <motion.div
      className="px-tab-pane"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className="px-info-card">
        <h3>Console — ships in v2.3 (Phase 14.3)</h3>
        <p>
          Browser-based noVNC + xterm.js terminal will land here. It will connect
          directly to the Proxmox VNC websocket for graphical consoles, or to the
          existing web-terminal WS bridge for serial / shell access.
        </p>
        <button type="button" className="px-btn px-btn-quiet" disabled title="Coming in Phase 14.3">
          Open noVNC (coming soon)
        </button>
      </div>
    </motion.div>
  );
}