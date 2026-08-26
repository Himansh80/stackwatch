/**
 * SidebarItem.tsx — single nav row.
 * Receives an item from nav-config.ts + current path, renders a <Link>
 * with icon + label + active-state styling.
 *
 * Visual states:
 *   - default:    muted text, transparent bg
 *   - hover:      text becomes bright, subtle bg
 *   - active:     accent text, elevated bg, 3px accent stripe on the left
 *
 * Active state is computed via NavLink (react-router) so it stays in
 * sync with the URL — no manual className management.
 */
import { Link, useLocation } from 'react-router-dom';
import { motion } from 'framer-motion';
import { getIcon } from '../icons';
import type { NavItem } from './nav-config';

interface SidebarItemProps {
  item: NavItem;
  onClick?: () => void;
}

function isItemActive(item: NavItem, currentPath: string): boolean {
  if (item.path === '/dashboard') return currentPath === '/dashboard';
  return currentPath === item.path || currentPath.startsWith(item.path + '/');
}

export default function SidebarItem({ item, onClick }: SidebarItemProps) {
  const Icon = getIcon(item.icon);
  const location = useLocation();
  const active = isItemActive(item, location.pathname);

  const className = 'sb-item' + (active ? ' sb-item-active' : '');

  const content = (
    <>
      <span className="sb-item-icon" aria-hidden="true">
        <Icon size={16} />
      </span>
      <span className="sb-item-label">{item.label}</span>
    </>
  );

  return (
    <motion.div
      whileHover={{ x: 2 }}
      transition={{ duration: 0.15, ease: [0.16, 1, 0.3, 1] }}
    >
      {item.external ? (
        <a
          href={item.path}
          target="_blank"
          rel="noopener noreferrer"
          className={className}
          onClick={onClick}
        >
          {content}
        </a>
      ) : (
        <Link to={item.path} className={className} onClick={onClick}>
          {content}
        </Link>
      )}
    </motion.div>
  );
}