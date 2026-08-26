/**
 * TopBarSearch.tsx — command palette trigger button.
 * Clicking it opens the existing CommandPalette.tsx (Cmd+K).
 *
 * The button is just a visual affordance; the actual palette is mounted
 * once at the AppShell level (we already have CommandPalette).
 */
import { SearchIcon } from '../icons';

interface TopBarSearchProps {
  onOpenPalette: () => void;
}

export default function TopBarSearch({ onOpenPalette }: TopBarSearchProps) {
  return (
    <button
      type="button"
      className="tb-search"
      onClick={onOpenPalette}
      aria-label="Open command palette (Cmd+K)"
    >
      <SearchIcon size={14} />
      <span className="tb-search-placeholder">Search &amp; navigate…</span>
      <kbd className="tb-search-kbd">⌘K</kbd>
    </button>
  );
}