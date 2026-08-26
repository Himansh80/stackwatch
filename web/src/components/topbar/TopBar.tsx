/**
 * TopBar.tsx — orchestrator for the sticky top bar.
 * Renders the following zones from left to right:
 *   1. Mobile hamburger (visible only <768px)
 *   2. Greeting + LIVE indicator
 *   3. Breadcrumb
 *   4. Search / command palette trigger (flex-grow, takes remaining space)
 *   5. Time widget
 *   6. Weather widget
 *   7. Notifications
 *   8. Refresh
 *   9. Profile
 *
 * Sticky to top with backdrop-blur for premium feel.
 */
import TopBarGreeting from './TopBarGreeting';
import TopBarBreadcrumb from './TopBarBreadcrumb';
import TopBarSearch from './TopBarSearch';
import TopBarTime from './TopBarTime';
import TopBarWeather from './TopBarWeather';
import TopBarRefresh from './TopBarRefresh';
import TopBarNotifications from './TopBarNotifications';
import TopBarProfile from './TopBarProfile';
import { MenuIcon } from '../icons';

interface TopBarProps {
  onMobileMenu: () => void;
  onOpenPalette: () => void;
  onRefresh: () => Promise<void> | void;
}

export default function TopBar({ onMobileMenu, onOpenPalette, onRefresh }: TopBarProps) {
  return (
    <header className="tb">
      <button
        type="button"
        className="tb-hamburger"
        onClick={onMobileMenu}
        aria-label="Open navigation menu"
      >
        <MenuIcon size={18} />
      </button>

      <TopBarGreeting />
      <TopBarBreadcrumb />

      <div className="tb-search-wrap">
        <TopBarSearch onOpenPalette={onOpenPalette} />
      </div>

      <div className="tb-right">
        <TopBarTime />
        <TopBarWeather />
        <TopBarNotifications />
        <TopBarRefresh onRefresh={onRefresh} />
        <TopBarProfile />
      </div>
    </header>
  );
}