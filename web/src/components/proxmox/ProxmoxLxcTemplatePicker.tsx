/**
 * Tier 14 Phase 14.4 — LXC template picker.
 * Fetches templates from each storage that supports vztmpl content.
 */
import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import { formatBytes, readString } from '../../lib/proxmox';

export interface Template {
  volid: string;
  format?: string;
  size?: number;
  notes?: string;
}

interface Storage {
  storage: string;
  content: string;
}

interface Props {
  hostId: string;
  node: string;
  selected: string;
  onChange: (template: Template | null) => void;
}

export default function ProxmoxLxcTemplatePicker({ hostId, node, selected, onChange }: Props) {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!hostId || !node) return;
    let alive = true;
    setLoading(true);
    (async () => {
      try {
        // Get all storages on this node that support vztmpl
        const storagesResp = await api<{ storage?: Storage[] } | Storage[]>(
          'GET',
          `/api/v1/proxmox/hosts/${hostId}/storage`,
        );
        const storages = Array.isArray(storagesResp) ? storagesResp : storagesResp.storage ?? [];
        const vztmplStorages = storages.filter((s) => readString(s.content).includes('vztmpl'));

        // Fetch templates from each
        const all: Template[] = [];
        for (const s of vztmplStorages) {
          try {
            const content = await api<{ content?: Template[] } | Template[]>(
              'GET',
              `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage/${encodeURIComponent(s.storage)}/content`,
            );
            const list = Array.isArray(content) ? content : content.content ?? [];
            all.push(...list.filter((t) => readString(t.format) === 'tgz' || t.volid.includes('.tar.')));
          } catch {
            /* ignore — storage might not allow listing */
          }
        }
        if (alive) setTemplates(all);
      } catch (cause) {
        if (alive) setError(cause instanceof Error ? cause.message : 'Unable to load templates.');
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [hostId, node]);

  if (loading) return <p className="px-muted">Loading templates…</p>;
  if (error) return <div className="px-error">⚠ {error}</div>;
  if (templates.length === 0) {
    return (
      <div className="px-info-card">
        <h3>No LXC templates found</h3>
        <p>Upload an LXC template (vztmpl) to a storage that supports it, then come back.</p>
      </div>
    );
  }

  return (
    <div className="px-template-grid">
      {templates.map((t) => {
        const name = t.volid.split('/').pop() ?? t.volid;
        const isSelected = selected === t.volid;
        return (
          <motion.button
            key={t.volid}
            type="button"
            className={`px-template-card ${isSelected ? 'px-template-active' : ''}`}
            onClick={() => onChange(t)}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.97 }}
          >
            <div className="px-template-icon">📦</div>
            <div className="px-template-name">{name}</div>
            <div className="px-template-meta">
              <span>{formatBytes(t.size ?? 0)}</span>
              {t.notes && <span className="px-template-notes">{t.notes}</span>}
            </div>
            {isSelected && <div className="px-template-checkmark">✓</div>}
          </motion.button>
        );
      })}
    </div>
  );
}