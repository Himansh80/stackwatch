import { useCallback, useEffect, useMemo, useRef, useState, KeyboardEvent } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, useReducedMotion } from '../../lib/motion';

/**
 * GlobalSearch — Tier 10 Phase 7 (H7 — Search).
 *
 * Global search bar that lives in the HomelabPage topbar (NOT a
 * tab). Calls:
 *   GET /api/v1/homelab/search/suggestions?q=...
 *   GET /api/v1/homelab/search/kinds
 *
 * On input, debounces 300ms then fetches up to 8 typeahead
 * suggestions. The dropdown renders below the input with one row
 * per suggestion. Click a row → onNavigate(kind, id) callback
 * (parent switches tab + sets focus). Escape closes the dropdown.
 * Enter with a non-empty q would normally open the full search
 * page — Phase 7 keeps it inline (no separate /homelab/search
 * route), so Enter simply runs the same suggestion query and
 * picks the top result.
 *
 * Props:
 *   className   — optional wrapper class for the topbar layout
 *   onNavigate  — (kind: string, id: string) => void. Required so
 *                 the parent can switch to the right tab when the
 *                 user clicks a suggestion. Services kind carries
 *                 the URL itself (no tab switch — open new tab).
 *
 * Motion: pageEnter on the dropdown (reuses the existing token
 * exported from motion.tsx — no new variants). Honors
 * useReducedMotion.
 */

interface GlobalSearchProps {
  className?: string;
  onNavigate: (kind: string, id: string, url: string) => void;
}

interface Suggestion {
  kind: string;
  id: string;
  title: string;
  url: string;
}

interface KindMeta {
  kind: string;
  label: string;
  icon: string;
}

interface SuggestionsResponse {
  suggestions: string | Suggestion[];
  count: number;
  q: string;
  kinds?: string[];
}

interface KindsResponse {
  kinds: KindMeta[];
  count: number;
}

const DEBOUNCE_MS = 300;

const KIND_ICON_FALLBACK: Record<string, string> = {
  notes: '📝',
  todos: '✅',
  services: '📌',
  calendars: '📅',
  downloads: '⬇️',
  media: '🎬',
};

const KIND_LABEL_FALLBACK: Record<string, string> = {
  notes: 'Note',
  todos: 'Todo',
  services: 'Service',
  calendars: 'Calendar',
  downloads: 'Download',
  media: 'Media',
};

function asSuggestions(raw: unknown): Suggestion[] {
  // Defensive: api<T> returns `any` cast as T. The server returns
  // `{suggestions: [...]}` but a future refactor could rename the
  // key. Walk the shape so a minor backend tweak doesn't break
  // the dropdown silently.
  if (!raw) return [];
  if (Array.isArray(raw)) return raw as Suggestion[];
  if (typeof raw === 'object') {
    const obj = raw as Record<string, unknown>;
    if (Array.isArray(obj.suggestions)) return obj.suggestions as Suggestion[];
    if (Array.isArray(obj.results)) return obj.results as Suggestion[];
  }
  return [];
}

function asKinds(raw: unknown): KindMeta[] {
  if (!raw || typeof raw !== 'object') return [];
  const obj = raw as Record<string, unknown>;
  if (Array.isArray(obj.kinds)) return obj.kinds as KindMeta[];
  return [];
}

export default function GlobalSearch({ className, onNavigate }: GlobalSearchProps) {
  const reduce = useReducedMotion();
  const [query, setQuery] = useState('');
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [kinds, setKinds] = useState<KindMeta[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [open, setOpen] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<number | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Fetch the kinds catalog once on mount. Cheap (static) — no
  // need to refetch on every dropdown open. If the call fails
  // (e.g. user logged out), fall back to the hardcoded kind
  // dictionary so the dropdown still renders useful labels.
  useEffect(() => {
    let cancelled = false;
    if (!getToken()) return undefined;
    const load = async () => {
      try {
        const res = await api<KindsResponse>('GET', '/api/v1/homelab/search/kinds');
        if (!cancelled) {
          const list = asKinds(res);
          if (list.length > 0) setKinds(list);
        }
      } catch (cause) {
        // Swallow — the fallback dictionary in KIND_ICON_FALLBACK
        // / KIND_LABEL_FALLBACK keeps the UI usable if the
        // /kinds endpoint is unreachable. Don't surface this as
        // an error (the search itself still works).
        if (!cancelled) {
          // eslint-disable-next-line no-console
          console.warn('[GlobalSearch] failed to load kinds catalog:', cause);
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  // Stable lookup helpers from the kinds catalog. Fall back to
  // the hardcoded dictionary if the catalog never loaded.
  const iconFor = useCallback(
    (kind: string): string => {
      const hit = kinds.find(k => k.kind === kind);
      if (hit?.icon) return hit.icon;
      return KIND_ICON_FALLBACK[kind] || '•';
    },
    [kinds],
  );
  const labelFor = useCallback(
    (kind: string): string => {
      const hit = kinds.find(k => k.kind === kind);
      if (hit?.label) return hit.label;
      return KIND_LABEL_FALLBACK[kind] || kind;
    },
    [kinds],
  );

  // Debounced suggestions fetch. Cancels in-flight requests on
  // every new keystroke (so a slow earlier query can't overwrite
  // a faster later one — common race condition in typeahead).
  const fetchSuggestions = useCallback(async (q: string) => {
    if (!getToken()) {
      setSuggestions([]);
      setLoading(false);
      return;
    }
    if (abortRef.current) abortRef.current.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    setError('');
    try {
      const res = await api<SuggestionsResponse>(
        'GET',
        `/api/v1/homelab/search/suggestions?q=${encodeURIComponent(q)}`,
        undefined,
        true,
        { signal: ctrl.signal },
      );
      const list = asSuggestions(res);
      setSuggestions(list);
      setHighlight(0);
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === 'AbortError') return;
      if (cause instanceof ApiError && cause.status === 401) {
        // Token expired mid-session — silently close the
        // dropdown rather than flooding the user with errors.
        setOpen(false);
        setSuggestions([]);
      } else {
        setError(cause instanceof Error ? cause.message : 'search failed');
        setSuggestions([]);
      }
    } finally {
      if (abortRef.current === ctrl) setLoading(false);
    }
  }, []);

  // Debounce wrapper. setTimeout id is stored on a ref so we can
  // cancel on the next keystroke without a re-render.
  useEffect(() => {
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }
    const trimmed = query.trim();
    if (trimmed.length === 0) {
      setSuggestions([]);
      setLoading(false);
      return undefined;
    }
    if (trimmed.length < 2) {
      // Backend rejects queries <2 chars only informally — keep
      // an empty list rather than spamming the API on every
      // single character typed.
      setSuggestions([]);
      setLoading(false);
      return undefined;
    }
    debounceRef.current = window.setTimeout(() => {
      void fetchSuggestions(trimmed);
    }, DEBOUNCE_MS);
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
    };
  }, [query, fetchSuggestions]);

  // Click-outside closes the dropdown. We use a mousedown
  // listener (not click) so the dropdown closes before any
  // click-on-document logic in parent components fires.
  useEffect(() => {
    function onMouseDown(e: MouseEvent) {
      if (!wrapperRef.current) return;
      if (!wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener('mousedown', onMouseDown);
    return () => document.removeEventListener('mousedown', onMouseDown);
  }, []);

  // Cleanup on unmount — abort any pending fetch + clear the
  // debounce so a stray timer can't update unmounted state.
  useEffect(() => {
    return () => {
      if (abortRef.current) abortRef.current.abort();
      if (debounceRef.current !== null) window.clearTimeout(debounceRef.current);
    };
  }, []);

  function handleKey(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      e.preventDefault();
      setOpen(false);
      inputRef.current?.blur();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setOpen(true);
      setHighlight(h => Math.min(suggestions.length - 1, h + 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlight(h => Math.max(0, h - 1));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const target = suggestions[highlight];
      if (target) {
        onNavigate(target.kind, target.id, target.url);
        setOpen(false);
        setQuery('');
        setSuggestions([]);
      } else if (suggestions.length > 0) {
        // No highlight set — pick the top result.
        const top = suggestions[0];
        onNavigate(top.kind, top.id, top.url);
        setOpen(false);
        setQuery('');
        setSuggestions([]);
      }
    }
  }

  function handleSelect(s: Suggestion) {
    onNavigate(s.kind, s.id, s.url);
    setOpen(false);
    setQuery('');
    setSuggestions([]);
  }

  // Pre-compute the empty / loading / error / results render.
  const showEmpty = useMemo(
    () => open && query.trim().length >= 2 && !loading && suggestions.length === 0 && !error,
    [open, query, loading, suggestions.length, error],
  );

  return (
    <div ref={wrapperRef} className={`homelab-search ${className || ''}`.trim()}>
      <div className="homelab-search-input-wrap">
        <span className="homelab-search-icon" aria-hidden="true">⌕</span>
        <input
          ref={inputRef}
          type="search"
          className="homelab-search-input"
          placeholder="Search notes, todos, services, calendars, downloads, media..."
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
          onFocus={() => { if (suggestions.length > 0) setOpen(true); }}
          onKeyDown={handleKey}
          autoComplete="off"
          spellCheck={false}
          aria-label="Search your homelab"
          aria-autocomplete="list"
          aria-expanded={open}
          aria-controls="homelab-search-listbox"
        />
        {loading ? (
          <span className="homelab-search-spinner" aria-label="Searching">
            <motion.span
              animate={reduce ? undefined : { rotate: 360 }}
              transition={reduce ? undefined : { duration: 0.9, repeat: Infinity, ease: 'linear' }}
            >
              ◌
            </motion.span>
          </span>
        ) : null}
      </div>

      {open ? (
        <motion.div
          id="homelab-search-listbox"
          role="listbox"
          className="homelab-search-dropdown"
          initial={reduce ? false : { opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={reduce ? { duration: 0 } : { duration: 0.15 }}
        >
          {error ? (
            <div className="homelab-search-empty" role="alert">
              <strong>Search failed</strong>
              <small>{error}</small>
            </div>
          ) : showEmpty ? (
            <div className="homelab-search-empty">
              <span aria-hidden="true">⌕</span>
              <strong>No results for &ldquo;{query.trim()}&rdquo;</strong>
              <small>Try a different word, or check spelling.</small>
            </div>
          ) : suggestions.length === 0 ? (
            <div className="homelab-search-empty">
              <span aria-hidden="true">⌕</span>
              <strong>Type 2+ characters</strong>
              <small>Search across notes, todos, services, calendars, downloads, media.</small>
            </div>
          ) : (
            <>
              {suggestions.map((s, i) => (
                <button
                  key={`${s.kind}-${s.id}-${i}`}
                  type="button"
                  className={`homelab-search-item${i === highlight ? ' homelab-search-item-active' : ''}`}
                  role="option"
                  aria-selected={i === highlight}
                  onMouseEnter={() => setHighlight(i)}
                  onClick={() => handleSelect(s)}
                >
                  <span className="homelab-search-item-icon" aria-hidden="true">{iconFor(s.kind)}</span>
                  <span className="homelab-search-item-body">
                    <strong>{s.title}</strong>
                    <small>{labelFor(s.kind)}</small>
                  </span>
                  <kbd className="homelab-search-kbd">↵</kbd>
                </button>
              ))}
            </>
          )}
        </motion.div>
      ) : null}
    </div>
  );
}