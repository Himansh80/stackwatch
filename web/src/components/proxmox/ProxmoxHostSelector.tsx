/**
 * Tier 14 Phase 14.1 — Host selector dropdown.
 * Lists all registered Proxmox hosts with online/offline indicator.
 */
import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';

interface ProxmoxHost {
  id: string;
  name: string;
  base_url: string;
  status?: string;
}

interface Props {
  selected: string;
  onChange: (hostId: string) => void;
}

export default function ProxmoxHostSelector({ selected, onChange }: Props) {
  const [hosts, setHosts] = useState<ProxmoxHost[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api<{ hosts?: ProxmoxHost[] } | ProxmoxHost[]>('GET', '/api/v1/proxmox/hosts')
      .then((data) => {
        if (!alive) return;
        const list = Array.isArray(data) ? data : data.hosts ?? [];
        setHosts(list);
        if (!selected && list.length > 0 && list[0]) onChange(list[0].id);
        setLoading(false);
      })
      .catch((cause) => {
        if (!alive) return;
        setError(cause instanceof Error ? cause.message : 'Unable to load hosts.');
        setLoading(false);
      });
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading) {
    return (
      <div className="px-host-selector">
        <span className="px-muted">Loading hosts…</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="px-host-selector px-host-error">
        <span>⚠ {error}</span>
      </div>
    );
  }

  if (hosts.length === 0) {
    return (
      <div className="px-host-selector px-host-empty">
        <span>No Proxmox hosts registered. Add one in Settings → Hosts.</span>
      </div>
    );
  }

  return (
    <div className="px-host-selector">
      <label htmlFor="px-host-select" className="px-host-label">
        Proxmox host:
      </label>
      <select
        id="px-host-select"
        value={selected}
        onChange={(event) => onChange(event.target.value)}
        className="px-host-select"
      >
        {hosts.map((host) => (
          <option key={host.id} value={host.id}>
            {host.name} ({host.base_url})
          </option>
        ))}
      </select>
      <motion.span
        key={selected}
        className="px-host-status"
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
      >
        {hosts.find((h) => h.id === selected)?.status === 'online' ? '● online' : '● offline'}
      </motion.span>
    </div>
  );
}