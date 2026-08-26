/**
 * Tier 14 Phase 14.7 — Storage content browser.
 * Lists files (ISOs, templates, backups) for a Proxmox storage.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import { formatBytes } from '../../lib/proxmox';

interface StorageContent {
  volid: string;
  format?: string;
  size?: number;
  parent?: string;
  ctime?: string;
  notes?: string;
  used?: number;
  encryption?: string;
  verification?: { state: string };
}

interface Storage {
  storage: string;
  content: string;
}

export default function ProxmoxStorageContentPage() {
  const { hostId = '', node = '', storage = '' } = useParams();
  const [items, setItems] = useState<StorageContent[]>([]);
  const [contentTypes, setContentTypes] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [deleting, setDeleting] = useState<string | null>(null);
  const [uploadUrl, setUploadUrl] = useState('');
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    if (!hostId || !node || !storage) return;
    setLoading(true);
    setError('');
    try {
      // Get storage info
      const storagesResp = await api<{ storage?: Storage[] } | Storage[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/storage`,
      );
      const list = Array.isArray(storagesResp) ? storagesResp : storagesResp.storage ?? [];
      const found = list.find((s) => s.storage === storage);
      setContentTypes(found?.content ?? '');

      // Get content
      const data = await api<{ content?: StorageContent[] } | StorageContent[]>(
        'GET',
        `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage/${encodeURIComponent(storage)}/content`,
      );
      setItems(Array.isArray(data) ? data : data.content ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load storage content.');
    } finally {
      setLoading(false);
    }
  }, [hostId, node, storage]);

  useEffect(() => {
    void load();
  }, [load]);

  function doDelete() {
    if (!deleting) return;
    api('DELETE', `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage/${encodeURIComponent(storage)}/content?volume=${encodeURIComponent(deleting)}`)
      .then(() => {
        setDeleting(null);
        void load();
      })
      .catch((cause) => {
        setError(cause instanceof Error ? cause.message : 'Delete failed.');
      });
  }

  function doUpload() {
    if (!uploadUrl) return;
    setUploading(true);
    api('POST', `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(node)}/storage/${encodeURIComponent(storage)}/content?url=${encodeURIComponent(uploadUrl)}&filename=${encodeURIComponent(uploadUrl.split('/').pop() || 'upload')}`)
      .then(() => {
        setUploadUrl('');
        setUploading(false);
        void load();
      })
      .catch((cause) => {
        setError(cause instanceof Error ? cause.message : 'Upload failed.');
        setUploading(false);
      });
  }

  function formatType(volid: string): { icon: string; label: string; color: string } {
    if (volid.endsWith('.iso')) return { icon: '💿', label: 'ISO', color: 'green' };
    if (volid.includes('vztmpl')) return { icon: '📦', label: 'CT Template', color: 'blue' };
    if (volid.endsWith('.tar.gz') || volid.endsWith('.tar.xz')) return { icon: '🗜', label: 'Archive', color: 'amber' };
    if (volid.includes('backup') || volid.endsWith('.vma') || volid.endsWith('.vzdump')) return { icon: '💾', label: 'Backup', color: 'slate' };
    if (volid.endsWith('.img')) return { icon: '🖼', label: 'Disk image', color: 'green' };
    return { icon: '📄', label: 'File', color: 'slate' };
  }

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <Link to={`/proxmox/nodes/${hostId}/${encodeURIComponent(node)}`}>{node}</Link>
        <span className="px-detail-sep">›</span>
        <span>Storage: {storage}</span>
      </div>
      <h1 className="px-detail-name">Storage: {storage}</h1>
      <div className="px-detail-meta">
        <span><strong>Content types</strong> {contentTypes || '—'}</span>
        <span><strong>Files</strong> {items.length}</span>
      </div>

      {error && <div className="px-error">⚠ {error}</div>}

      {contentTypes.includes('iso') || contentTypes.includes('vztmpl') || contentTypes.includes('backup') ? (
        <section className="px-summary-card" style={{ marginTop: 12 }}>
          <h3>Upload from URL</h3>
          <div style={{ display: 'flex', gap: 8 }}>
            <input
              type="text"
              value={uploadUrl}
              onChange={(e) => setUploadUrl(e.target.value)}
              placeholder="https://example.com/ubuntu-22.04.iso"
              className="px-confirm-input"
              style={{ flex: 1 }}
            />
            <button
              type="button"
              className="px-btn"
              onClick={doUpload}
              disabled={!uploadUrl || uploading}
            >
              {uploading ? 'Uploading…' : 'Upload'}
            </button>
          </div>
        </section>
      ) : null}

      <div style={{ marginTop: 12 }}>
        {loading ? (
          <p className="px-muted">Loading content…</p>
        ) : items.length === 0 ? (
          <div className="px-info-card">
            <h3>No content</h3>
            <p>This storage has no files. Upload from URL above (if supported) or use the Proxmox web UI.</p>
          </div>
        ) : (
          <table className="px-network-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Volume ID</th>
                <th>Format</th>
                <th>Size</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((it) => {
                const t = formatType(it.volid);
                return (
                  <tr key={it.volid}>
                    <td>
                      <span className="px-mr-4">{t.icon}</span>
                      <span className={`px-action-badge px-action-${t.color}`}>{t.label}</span>
                    </td>
                    <td className="px-mono">{it.volid}</td>
                    <td>{it.format || '—'}</td>
                    <td className="px-mono">{formatBytes(it.size ?? 0)}</td>
                    <td className="px-mono">{it.ctime || '—'}</td>
                    <td>
                      <button
                        type="button"
                        className="px-btn px-btn-quiet px-btn-danger-text"
                        onClick={() => setDeleting(it.volid)}
                      >
                        ✕ Delete
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {deleting && (
        <motion.div
          className="px-confirm-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          onClick={() => setDeleting(null)}
        >
          <motion.div
            className="px-confirm-dialog"
            initial={{ scale: 0.95 }}
            animate={{ scale: 1 }}
            onClick={(e) => e.stopPropagation()}
          >
            <h3>Delete this file?</h3>
            <p>
              This will permanently delete <code>{deleting}</code>. This cannot be undone.
            </p>
            <div className="px-confirm-actions">
              <button type="button" className="px-btn px-btn-quiet" onClick={() => setDeleting(null)}>
                Cancel
              </button>
              <button type="button" className="px-btn px-btn-danger" onClick={doDelete}>
                Yes, delete
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </motion.div>
  );
}