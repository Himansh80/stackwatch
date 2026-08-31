/**
 * NotebookEditor — collaborative markdown notebook editor for Tier 7.10 (D11).
 *
 * Props:
 *   notebook: {id, title, content, author_id, last_edited_at, created_at?, collaborators?}
 *   onSave:  (nextContent: string) => void   // called on Save click; parent updates state + PUTs
 *   onClose: () => void                      // called when the X button is clicked
 *
 * Layout (top to bottom):
 *   - Title input — 16px bold
 *   - Collaborators chip row (small inline list, right-aligned)
 *   - Textarea — full width, monospace font (for code blocks), min-height 400px
 *   - Action row — "Last edited" timestamp on the left, Save + Close on the right
 *
 * Per the spec we deliberately use a PLAIN TEXTAREA — no ProseMirror,
 * no Slate. The backend stores the markdown verbatim and a separate
 * preview render can be added later by piping `content` through a
 * lightweight library like `marked` if/when needed.
 *
 * Design tokens only — no hex literals. Reuses the same .threat-card
 * frame + .threat-card-meta-pill chips from IncidentCard so the look
 * stays consistent across the operations surface.
 */

import { useState } from 'react';
import Button from './Button';
import Input from './Input';
import Textarea from './Textarea';

export interface NotebookCollaborator {
  user_id: string;
  role: 'viewer' | 'editor' | string;
  added_at?: string;
}

export interface Notebook {
  id: string;
  title: string;
  content: string;
  author_id?: string | null;
  last_edited_at: string;
  created_at?: string;
  collaborators?: NotebookCollaborator[];
}

export interface NotebookEditorProps {
  notebook: Notebook;
  onSave: (nextContent: string) => void;
  onClose: () => void;
}

function shortId(id: string | null | undefined): string {
  if (!id) return '—';
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

function collabClass(tone: 'viewer' | 'editor' | 'owner'): string {
  const base = 'threat-card-meta-pill';
  if (tone === 'editor') return `${base} notebook-collab-editor`;
  if (tone === 'owner') return `${base} notebook-collab-owner`;
  return `${base} notebook-collab-viewer`;
}

export default function NotebookEditor({ notebook, onSave, onClose }: NotebookEditorProps) {
  // Local controlled state — parent passes initial values, we buffer
  // edits and flush via onSave when the user clicks Save.
  const [title, setTitle] = useState<string>(notebook.title);
  const [content, setContent] = useState<string>(notebook.content);

  // Render collaborators as a chip row. The author (if known) is
  // always shown first as "owner"; subsequent rows are the
  // notebook_collaborators entries. Same shape either way so we can
  // .map() uniformly.
  const collabList: Array<{ user_id: string; role: 'viewer' | 'editor' | 'owner' }> = [];
  if (notebook.author_id) {
    collabList.push({ user_id: notebook.author_id, role: 'owner' });
  }
  for (const c of notebook.collaborators || []) {
    const role = (c.role || 'viewer') as 'viewer' | 'editor';
    // Skip a collab row that duplicates the author so the chip row
    // doesn't show the same id twice with different labels.
    if (c.user_id === notebook.author_id) continue;
    collabList.push({ user_id: c.user_id, role });
  }

  const lastEdited = notebook.last_edited_at
    ? new Date(notebook.last_edited_at).toLocaleString()
    : '—';

  const dirty = title !== notebook.title || content !== notebook.content;

  return (
    <div className="slow-query-explain-modal" role="dialog" aria-modal="true" aria-label={`Edit ${notebook.title}`}>
      <div className="slow-query-explain-modal-head">
        <strong>{title || 'Untitled notebook'}</strong>
        <button
          type="button"
          className="slow-query-explain-close"
          onClick={onClose}
          aria-label="Close"
        >
          ✕
        </button>
      </div>

      <div className="notebook-editor-body">
        <Input
          label="Title"
          type="text"
          value={title}
          maxLength={256}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Production runbook — Database failover"
          fullWidth
        />

        <div className="notebook-editor-collab-row">
          <span className="notebook-editor-collab-label">Collaborators</span>
          {collabList.length === 0 ? (
            <span className="threat-card-meta-pill">
              <span className="threat-card-meta-label">none</span>
            </span>
          ) : (
            collabList.map((c) => (
              <span key={c.user_id} className={collabClass(c.role)} title={`role: ${c.role}`}>
                <span className="threat-card-meta-label">{c.role}</span>
                <code>{shortId(c.user_id)}</code>
              </span>
            ))
          )}
        </div>

        <Textarea
          label="Markdown content"
          value={content}
          maxLength={131072}
          onChange={(e) => setContent(e.target.value)}
          placeholder={`# Runbook\n\n## Overview\nDescribe the system here.\n\n## Steps\n1. ...\n\n\`\`\`bash\necho run the command\n\`\`\``}
          rows={16}
          fullWidth
          spellCheck={false}
          style={{
            fontFamily: 'var(--mono, ui-monospace, SFMono-Regular, Menlo, monospace)',
            minHeight: 400,
            resize: 'vertical',
          }}
        />

        <div className="notebook-editor-actions">
          <span className="notebook-editor-meta">
            <span className="threat-card-meta-label">Last edited</span>
            <code>{lastEdited}</code>
          </span>
          <div className="notebook-editor-actions-right">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onClose}
            >
              Close
            </Button>
            <Button
              type="button"
              variant="primary"
              size="sm"
              disabled={!dirty}
              onClick={() => onSave(content)}
            >
              {dirty ? 'Save' : 'Saved'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
