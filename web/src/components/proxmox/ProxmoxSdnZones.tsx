/**
 * Tier 14 Phase 14.8 — SDN zones (read-only list for Phase 14.8).
 * Full CRUD ships in Phase 14.10.
 */
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';

interface SdnZone {
  zone: string;
  type?: string;
  bridge?: string;
  ipam?: string;
  dhcp?: string;
  state?: string;
}

export default function ProxmoxSdnZones() {
  const [zones, setZones] = useState<SdnZone[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api<{ zones?: SdnZone[] } | SdnZone[]>('GET', '/api/v1/proxmox/sdn/zones');
      const list = Array.isArray(data) ? data : data.zones ?? [];
      setZones(list);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load SDN zones.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  return (
    <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <div className="px-detail-breadcrumb">
        <Link to="/proxmox">Proxmox</Link>
        <span className="px-detail-sep">›</span>
        <span>SDN zones</span>
      </div>
      <h1 className="px-detail-name">SDN zones</h1>
      <div className="px-detail-meta">
        <span><strong>Total</strong> {zones.length}</span>
        <span className="px-confirm-warn" style={{ fontSize: 12 }}>
          Read-only in Phase 14.8 — full CRUD ships in Phase 14.10
        </span>
      </div>

      {error && <div className="px-error">⚠ {error}</div>}

      {loading ? (
        <p className="px-muted">Loading SDN zones…</p>
      ) : zones.length === 0 ? (
        <div className="px-info-card">
          <h3>No SDN zones configured</h3>
          <p>Configure SDN in Proxmox under Datacenter → SDN to see zones here.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>Zone</th>
              <th>Type</th>
              <th>Bridge</th>
              <th>IPAM</th>
              <th>DHCP</th>
              <th>State</th>
            </tr>
          </thead>
          <tbody>
            {zones.map((z) => (
              <tr key={z.zone}>
                <td className="px-mono">{z.zone}</td>
                <td>{z.type || '—'}</td>
                <td className="px-mono">{z.bridge || '—'}</td>
                <td>{z.ipam || '—'}</td>
                <td>{z.dhcp || '—'}</td>
                <td>
                  <span className={`px-action-badge px-action-${z.state === 'available' ? 'accept' : 'drop'}`}>
                    {z.state || 'unknown'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </motion.div>
  );
}