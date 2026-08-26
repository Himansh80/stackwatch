/**
 * Tier 14 Phase 14.5 — ISO picker.
 * Fetches ISOs from each storage that supports iso content.
 */
import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import { formatBytes, readString } from '../../lib/proxmox';

export interface Iso {
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
  onSelect: (iso: Iso | null) => void;
}

const NO_MEDIA: Iso = { volid: '', format: 'none', notes: 'Boot from existing disk' };

export default function ProxmoxIsoPicker({ hostId, node, selected, onSelect }: Props) {
  const [isos, setIsos] = useState<Iso[]>([NO_MEDIA]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!hostId || !node) return;
    let alive = true;
    setLoading(true);
    (async () => {
      try {
        const storagesResp = await api<{ storage?: Storage[] } | Storage[]>(
          'GET',
          `/api/v1/proxmox/hosts/${hostId}/storage`,
        );
        const storages = Array.isArray(storagesResp) ? storagesResp : storagesResp.storage ?? [];
        const isoStorages = storages.filter((s) => readString(s.content).includes('iso'));

        const all: Iso[] = [NO_MEDIA];
        for (const s of isoStorages) {
          try {
            const content = await api<{ content?: Iso[] } | Iso[]>(
              'GET',
              `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage/${encodeURIComponent(s.storage)}/content`,
            );
            const list = Array.isArray(content) ? content : content.content ?? [];
            all.push(...list.filter((i) => readString(i.format) === 'iso' || i.volid.endsWith('.iso')));
          } catch {
            /* ignore */
          }
        }
        if (alive) setIsos(all);
      } catch (cause) {
        if (alive) setError(cause instanceof Error ? cause.message : 'Unable to load ISOs.');
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [hostId, node]);

  if (loading) return <p className="px-muted">Loading ISOs…</p>;
  if (error) return <div className="px-error">⚠ {error}</div>;
  if (isos.length === 1) {
    return (
      <div className="px-info-card">
        <h3>No ISOs found</h3>
        <p>Upload an ISO to a storage that supports <code>iso</code> content, then come back.</p>
      </div>
    );
  }

  return (
    <div className="px-template-grid">
      {isos.map((iso, idx) => {
        const name = iso.volid === '' ? 'No media (boot from disk)' : iso.volid.split('/').pop() ?? iso.volid;
        const isSelected = selected === iso.volid;
        return (
          <motion.button
            key={iso.volid || `nomedia-${idx}`}
            type="button"
            className={`px-template-card ${isSelected ? 'px-template-active' : ''}`}
            onClick={() => onSelect(iso)}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.97 }}
          >
            <div className="px-template-icon">{iso.volid === '' ? '○' : '💿'}</div>
            <div className="px-template-name">{name}</div>
            <div className="px-template-meta">
              {iso.volid !== '' && <span>{formatBytes(iso.size ?? 0)}</span>}
              {iso.notes && <span className="px-template-notes">{iso.notes}</span>}
            </div>
            {isSelected && <div className="px-template-checkmark">✓</div>}
          </motion.button>
        );
      })}
    </div>
  );
}