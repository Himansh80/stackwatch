/**
 * Sidebar.tsx — main orchestrator.
 * Renders the brand block at top, all 4 nav sections, and the footer.
 *
 * Props:
 *   - onItemClick: optional click handler forwarded to every nav item
 *                  (used by mobile drawer to close after navigation)
 *   - inDrawer:    when true, the sidebar renders without its own
 *                  container (used inside the mobile drawer)
 */
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { NAV_SECTIONS } from './nav-config';
import SidebarSection from './SidebarSection';
import SidebarFooter from './SidebarFooter';
import BrandLogo from '../shared/BrandLogo';

interface SidebarProps {
  onItemClick?: () => void;
  inDrawer?: boolean;
}

export default function Sidebar({ onItemClick, inDrawer = false }: SidebarProps) {
  return (
    <aside className={'sb' + (inDrawer ? ' sb-drawer' : '')}>
      <Link to="/dashboard" className="sb-brand" onClick={onItemClick}>
        <BrandLogo variant="mark" size={28} />
        <span className="sb-brand-text">
          <strong>StackWatch</strong>
          <small>Infrastructure Control Plane</small>
        </span>
      </Link>

      <motion.nav
        className="sb-nav"
        initial={inDrawer ? { opacity: 0 } : false}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.2 }}
        aria-label="Primary navigation"
      >
        {NAV_SECTIONS.map((section) => (
          <SidebarSection
            key={section.label}
            section={section}
            onItemClick={onItemClick}
          />
        ))}
      </motion.nav>

      <SidebarFooter onItemClick={onItemClick} />
    </aside>
  );
}