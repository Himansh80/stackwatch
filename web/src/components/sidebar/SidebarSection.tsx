/**
 * SidebarSection.tsx — section heading + items list.
 * Renders the uppercase label + a vertical list of SidebarItems.
 *
 * Section labels are uppercase + letter-spaced + muted color per the
 * design system (`var(--text-xs)` + `var(--tracking-widest)`).
 */
import type { NavSection } from './nav-config';
import SidebarItem from './SidebarItem';

interface SidebarSectionProps {
  section: NavSection;
  onItemClick?: () => void; // forwarded to each item (for mobile drawer)
}

export default function SidebarSection({ section, onItemClick }: SidebarSectionProps) {
  return (
    <div className="sb-section">
      <div className="sb-section-label">{section.label}</div>
      <div className="sb-section-items">
        {section.items.map((item) => (
          <SidebarItem key={item.path} item={item} onClick={onItemClick} />
        ))}
      </div>
    </div>
  );
}