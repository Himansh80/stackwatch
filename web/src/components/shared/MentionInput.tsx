/**
 * MentionInput — Tier 7.11 (D12) @-mention autocomplete textarea.
 *
 * Props:
 *   users:      Array<{id, full_name, email}>
 *   onChange:   (text: string, mentionIds: string[]) => void
 *   placeholder?: string
 *
 * Layout:
 *   - Textarea (rows=4)
 *   - Below it: a horizontal chip strip of captured mentions so the
 *     caller can see who's been @-mentioned while they're typing.
 *   - An autocomplete dropdown that appears when the user types '@'
 *     followed by characters, with arrow-key / Enter / click
 *     selection. Selecting inserts '@FullName ' at the cursor and
 *     captures the user's id into the mentions set.
 *
 * Behavior:
 *   - Local state: text + Set<id> of mentions
 *   - Filters users by case-insensitive prefix match against the
 *     typed query after '@'
 *   - Caps results at 8 (avoids dropdown overload)
 *   - Backspace at position 0 of the @-query (or just after the
 *     trailing space of an inserted mention) removes the chip too —
 *     for Phase 5 we keep it simple: chip removal is via click only.
 *   - Re-fires onChange every time text OR mentions change so the
 *     parent always has both pieces in sync.
 *
 * The parent owns persistence — POST to /api/v1/timeline/comments
 * with body {body, incident_id} AND walks the mentionIds to insert
 * into the `mentions` table. This component is purely a UI helper;
 * it never calls the API directly.
 *
 * Reuses existing tokens (.notebook-editor-textarea + .threat-card-
 * meta-pill chips) so no new CSS classes are introduced.
 */

import { KeyboardEvent, useMemo, useRef, useState } from 'react';

export interface MentionUser {
  id: string;
  full_name: string;
  email: string;
}

export interface MentionInputProps {
  users: MentionUser[];
  onChange: (text: string, mentionIds: string[]) => void;
  placeholder?: string;
  rows?: number;
  defaultValue?: string;
}

const MAX_RESULTS = 8;

export default function MentionInput({
  users,
  onChange,
  placeholder,
  rows = 4,
  defaultValue = '',
}: MentionInputProps) {
  const [text, setText] = useState<string>(defaultValue);
  const [mentionIds, setMentionIds] = useState<string[]>([]);
  const [query, setQuery] = useState<string>('');
  const [showMenu, setShowMenu] = useState<boolean>(false);
  const [highlight, setHighlight] = useState<number>(0);
  const cursorRef = useRef<number>(0);

  // After every text/mentions mutation we need to bubble to the
  // parent. We bundle that in handleText() to keep state derivation
  // in one place.
  const flush = (nextText: string, nextIds: string[]) => {
    setText(nextText);
    setMentionIds(nextIds);
    onChange(nextText, nextIds);
  };

  // Filter users by typed prefix. Case-insensitive substring match
  // against full_name OR email — substring on email is intentional
  // because some users have unreadable names.
  const filtered: MentionUser[] = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return users.slice(0, MAX_RESULTS);
    return users
      .filter((u) => {
        const name = (u.full_name || '').toLowerCase();
        const mail = (u.email || '').toLowerCase();
        return name.includes(q) || mail.includes(q);
      })
      .slice(0, MAX_RESULTS);
  }, [query, users]);

  // Resolve the mention set when text changes. We rebuild it from
  // scratch — the canonical set is "all user-ids that have an
  // '@<TheirFullName>' substring in the text AND appear in the
  // users list". Simpler than tracking offsets on every keystroke.
  const reconcileMentions = (nextText: string): string[] => {
    const found = new Set<string>();
    for (const u of users) {
      if (!u.full_name) continue;
      if (nextText.includes(`@${u.full_name}`)) {
        found.add(u.id);
      }
    }
    return Array.from(found);
  };

  const handleText = (raw: string, caret: number) => {
    const ids = reconcileMentions(raw);
    flush(raw, ids);

    // Detect an @-query at the caret. Walk backwards from caret
    // until we hit whitespace / start-of-string / another '@'. The
    // query is the substring between the last '@' and the caret.
    let i = caret - 1;
    while (i >= 0) {
      const ch = raw[i];
      if (ch === '@') break;
      if (ch === ' ' || ch === '\n' || ch === '\t') {
        // We're past the active mention token — close the menu.
        setQuery('');
        setShowMenu(false);
        return;
      }
      i--;
    }
    if (i < 0 || raw[i] !== '@') {
      setQuery('');
      setShowMenu(false);
      return;
    }
    const q = raw.slice(i + 1, caret);
    setQuery(q);
    setShowMenu(true);
    setHighlight(0);
    cursorRef.current = caret;
  };

  const pickUser = (user: MentionUser) => {
    // We rebuild the text with the new mention inserted at the
    // caret. Strategy: find the last '@' before the caret, take
    // text-before + '@<full_name>' + text-after-caret, then drop
    // any partial typed characters between the '@' and the caret.
    const caret = cursorRef.current;
    const before = text.slice(0, caret);
    const after = text.slice(caret);
    let i = before.length - 1;
    while (i >= 0 && before[i] !== '@') i--;
    if (i < 0 || before[i] !== '@') {
      // No active @-token — insert one at the caret.
      const insertion = `@${user.full_name} `;
      const next = before + insertion + after;
      handleText(next, before.length + insertion.length);
      return;
    }
    const next = before.slice(0, i) + `@${user.full_name} ` + after;
    const newCaret = i + `@${user.full_name} `.length;
    handleText(next, newCaret);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!showMenu || filtered.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlight((h) => (h + 1) % filtered.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlight((h) => (h - 1 + filtered.length) % filtered.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const u = filtered[highlight];
      if (u) pickUser(u);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      setShowMenu(false);
    }
  };

  const removeMention = (id: string) => {
    const user = users.find((u) => u.id === id);
    if (!user) return;
    const next = text.replace(`@${user.full_name}`, '').replace(/\s+/g, ' ').trim();
    const ids = reconcileMentions(next.length ? next + ' ' : next);
    flush(next, ids);
  };

  const chips = mentionIds
    .map((id) => users.find((u) => u.id === id))
    .filter((u): u is MentionUser => Boolean(u));

  return (
    <div className="mention-input">
      <textarea
        className="notebook-editor-textarea"
        rows={rows}
        value={text}
        maxLength={8192}
        onChange={(e) => {
          const v = e.target.value;
          const caret = e.target.selectionStart ?? v.length;
          handleText(v, caret);
        }}
        onKeyDown={onKeyDown}
        placeholder={placeholder || 'Type a comment. Use @name to mention teammates.'}
        spellCheck={false}
      />

      {showMenu && filtered.length > 0 ? (
        <div className="mention-input-menu" role="listbox">
          {filtered.map((u, idx) => (
            <button
              key={u.id}
              type="button"
              role="option"
              aria-selected={idx === highlight}
              className={`mention-input-menu-item${idx === highlight ? ' mention-input-menu-item-active' : ''}`}
              onClick={() => pickUser(u)}
            >
              <strong>@{u.full_name || u.email}</strong>
              <small>{u.email}</small>
            </button>
          ))}
        </div>
      ) : null}

      {chips.length > 0 ? (
        <div className="mention-input-chips">
          <span className="threat-card-meta-label">Mentions</span>
          {chips.map((u) => (
            <span key={u.id} className="threat-card-meta-pill">
              <code>@{u.full_name || u.email}</code>
              <button
                type="button"
                className="mention-input-chip-remove"
                aria-label={`Remove @${u.full_name || u.email}`}
                onClick={() => removeMention(u.id)}
              >
                ✕
              </button>
            </span>
          ))}
        </div>
      ) : null}
    </div>
  );
}
