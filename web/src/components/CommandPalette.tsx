import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';

/**
 * Cmd+K / Ctrl+K command palette.
 *
 * Global navigation + quick-action launcher, the kind of UI that
 * Linear / Raycast / Grafana all have. The user types a query,
 * sees a filtered list of routes + actions, and presses Enter
 * to invoke the highlighted one.
 *
 * Stateless from the outside: parent passes `open` + `onClose`,
 * the palette manages its own query/highlight state and dispatches
 * the chosen action.
 */

interface PaletteCommand {
  id: string;
  title: string;
  hint?: string;
  group: 'Navigate' | 'Actions';
  keywords: string[];
  run: () => void;
}

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
}

export default function CommandPalette({ open, onClose }: CommandPaletteProps) {
  const nav = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState('');
  const [highlight, setHighlight] = useState(0);

  // Build the command list. Runs every render but the list is small
  // (≈ 12 items) so it's effectively free.
  const commands: PaletteCommand[] = useMemo(() => [
    { id: 'go-overview', title: 'Go to Overview', hint: 'Dashboard / /dashboard', group: 'Navigate', keywords: ['home', 'dashboard', 'overview'], run: () => { nav('/dashboard'); onClose(); } },
    { id: 'go-profile', title: 'Go to Profile', group: 'Navigate', keywords: ['profile', 'user', 'account'], run: () => { nav('/profile'); onClose(); } },
    { id: 'go-billing', title: 'Go to Billing', group: 'Navigate', keywords: ['billing', 'plan', 'subscription'], run: () => { nav('/billing'); onClose(); } },
    { id: 'go-settings', title: 'Go to Settings', group: 'Navigate', keywords: ['settings', 'preferences', 'config'], run: () => { nav('/settings'); onClose(); } },
    { id: 'go-proxmox', title: 'Go to Proxmox workspace', group: 'Navigate', keywords: ['proxmox', 'infrastructure', 'servers'], run: () => { nav('/proxmox'); onClose(); } },
    { id: 'go-truenas', title: 'Go to TrueNAS workspace', group: 'Navigate', keywords: ['truenas', 'storage', 'nas'], run: () => { nav('/truenas'); onClose(); } },
    { id: 'action-refresh', title: 'Refresh page', hint: 'Reload current route', group: 'Actions', keywords: ['refresh', 'reload'], run: () => { window.location.reload(); } },
    { id: 'action-toggle-theme', title: 'Toggle theme', hint: 'Switch dark/light (coming soon)', group: 'Actions', keywords: ['theme', 'dark', 'light', 'mode'], run: () => { onClose(); } },
    { id: 'action-show-shortcuts', title: 'Show keyboard shortcuts', hint: '? to reopen', group: 'Actions', keywords: ['shortcuts', 'help', 'keyboard'], run: () => { onClose(); } },
  ], [nav, onClose]);

  // Reset state every time the palette opens so we never show stale
  // query results from the last open.
  useEffect(() => {
    if (open) {
      setQuery('');
      setHighlight(0);
      // Focus the input on the next tick so the modal is in the DOM
      // before we try to .focus() the input.
      const t = window.setTimeout(() => inputRef.current?.focus(), 16);
      return () => window.clearTimeout(t);
    }
    return undefined;
  }, [open]);

  // Filter by query. Empty query shows everything. We match against
  // the title (case-insensitive) and any keyword tokens. The order
  // is preserved so the list doesn't shuffle every keystroke.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return commands;
    return commands.filter((c) => {
      const haystack = [c.title, c.hint || '', ...c.keywords].join(' ').toLowerCase();
      return haystack.includes(q);
    });
  }, [commands, query]);

  // Keep highlight in range when the filtered list changes length.
  useEffect(() => {
    if (highlight >= filtered.length) setHighlight(0);
  }, [filtered.length, highlight]);

  function handleKey(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setHighlight((h) => Math.min(filtered.length - 1, h + 1));
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      setHighlight((h) => Math.max(0, h - 1));
    } else if (event.key === 'Enter') {
      event.preventDefault();
      const cmd = filtered[highlight];
      if (cmd) cmd.run();
    }
  }

  if (!open) return null;

  return <div className="cmd-palette-backdrop" onClick={onClose} role="dialog" aria-modal="true" aria-label="Command palette">
    <div className="cmd-palette" onClick={(e) => e.stopPropagation()}>
      <div className="cmd-palette-input-wrap">
        <span className="cmd-palette-icon" aria-hidden="true">⌕</span>
        <input
          ref={inputRef}
          type="search"
          className="cmd-palette-input"
          placeholder="Type a command, route, or action..."
          value={query}
          onChange={(e) => { setQuery(e.target.value); setHighlight(0); }}
          onKeyDown={handleKey}
          autoComplete="off"
          spellCheck={false}
          aria-label="Command palette search"
        />
        <kbd className="cmd-palette-kbd">esc</kbd>
      </div>
      <div className="cmd-palette-list" role="listbox">
        {filtered.length === 0 ? <div className="cmd-palette-empty">
          <span>⌕</span>
          <strong>No results for "{query}"</strong>
          <small>Try "overview", "settings", or "refresh".</small>
        </div> : filtered.map((cmd, index) => <button
          key={cmd.id}
          className={`cmd-palette-item ${index === highlight ? 'cmd-palette-item-active' : ''}`}
          onMouseEnter={() => setHighlight(index)}
          onClick={() => cmd.run()}
          role="option"
          aria-selected={index === highlight}
        >
          <span className="cmd-palette-item-group">{cmd.group}</span>
          <span className="cmd-palette-item-body">
            <strong>{cmd.title}</strong>
            {cmd.hint && <small>{cmd.hint}</small>}
          </span>
          <kbd className="cmd-palette-kbd">↵</kbd>
        </button>)}
      </div>
    </div>
  </div>;
}
