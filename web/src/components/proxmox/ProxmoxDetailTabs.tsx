/**
 * Tier 14 Phase 14.2 — Tab nav for VM detail page.
 * Hash-based (location.hash) so URLs are deep-linkable.
 * Mobile: collapses to <select> dropdown below 768px.
 */
import { useEffect, useState } from 'react';

export type DetailTab = 'summary' | 'hardware' | 'network' | 'console' | 'snapshots' | 'firewall';

interface Tab {
  id: DetailTab;
  label: string;
  icon: string;
}

const TABS: Tab[] = [
  { id: 'summary', label: 'Summary', icon: '◉' },
  { id: 'hardware', label: 'Hardware', icon: '▣' },
  { id: 'network', label: 'Network', icon: '⌁' },
  { id: 'console', label: 'Console', icon: '▶' },
  { id: 'snapshots', label: 'Snapshots', icon: '◰' },
  { id: 'firewall', label: 'Firewall', icon: '◍' },
];

interface Props {
  active: DetailTab;
  onChange: (tab: DetailTab) => void;
}

export default function ProxmoxDetailTabs({ active, onChange }: Props) {
  const [showMobile, setShowMobile] = useState(false);

  // Open mobile dropdown when active tab changes (so user can pick a different one)
  useEffect(() => {
    setShowMobile(false);
  }, [active]);

  return (
    <>
      {/* Desktop: pill-style tabs */}
      <div className="px-tabs px-hide-mobile" role="tablist">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={active === tab.id}
            className={`px-tab ${active === tab.id ? 'px-tab-active' : ''}`}
            onClick={() => onChange(tab.id)}
          >
            <span className="px-tab-icon">{tab.icon}</span>
            <span>{tab.label}</span>
          </button>
        ))}
      </div>

      {/* Mobile: collapsible dropdown */}
      <div className="px-tabs-mobile px-show-mobile">
        <button
          type="button"
          className="px-tabs-mobile-trigger"
          onClick={() => setShowMobile((s) => !s)}
          aria-expanded={showMobile}
        >
          <span className="px-tab-icon">{TABS.find((t) => t.id === active)?.icon}</span>
          <span>{TABS.find((t) => t.id === active)?.label}</span>
          <span className={`px-tabs-mobile-caret ${showMobile ? 'px-tabs-mobile-open' : ''}`}>▾</span>
        </button>
        {showMobile && (
          <div className="px-tabs-mobile-menu" role="menu">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                role="menuitem"
                className={`px-tabs-mobile-item ${active === tab.id ? 'px-tabs-mobile-active' : ''}`}
                onClick={() => onChange(tab.id)}
              >
                <span className="px-tab-icon">{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </>
  );
}