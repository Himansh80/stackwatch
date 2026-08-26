/**
 * Tier 14 Phase 14.10 — Tasks panel (live cluster task log).
 * Auto-refreshes every 5s. Filters by node, status, type.
 */
import { useCallback, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { api } from '../../lib/api';
import ProxmoxShell from './ProxmoxShell';

interface Task {
  upid: string;
  node: string;
  type: string;
  status: 'running' | 'OK' | 'error' | 'stopped' | string;
  starttime: number;
  endtime?: number;
  pid?: number;
  user?: string;
}

interface Node {
  node: string;
  status: string;
}

const REFRESH_MS = 5000;

export default function ProxmoxTasksPanel() {
  const [hostId, setHostId] = useState('');
  const [nodes, setNodes] = useState<Node[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [filterNode, setFilterNode] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [filterType, setFilterType] = useState('');
  const [stopping, setStopping] = useState<string | null>(null);

  // 1st: pick first host
  useEffect(() => {
    let alive = true;
    api<{ hosts?: Array<{ id: string; name: string }> }>(
      'GET',
      '/api/v1/proxmox/hosts',
    )
      .then((data) => {
        if (!alive) return;
        const hosts = data.hosts ?? [];
        if (hosts.length > 0 && hosts[0]) setHostId(hosts[0].id);
      })
      .catch(() => {/* ignore */});
    return () => { alive = false; };
  }, []);

  // 2nd: load nodes for selected host
  useEffect(() => {
    if (!hostId) return;
    let alive = true;
    api<{ nodes?: Node[] } | Node[]>(
      'GET',
      `/api/v1/proxmox/hosts/${hostId}/nodes`,
    )
      .then((d) => {
        if (!alive) return;
        setNodes(Array.isArray(d) ? d : d.nodes ?? []);
      })
      .catch(() => {/* ignore */});
    return () => { alive = false; };
  }, [hostId]);

  const load = useCallback(async () => {
    if (!hostId) return;
    setLoading(true);
    setError('');
    try {
      const data = await api<Task[] | { tasks?: Task[] }>('GET', `/api/v1/proxmox/hosts/${hostId}/cluster/tasks`);
      setTasks(Array.isArray(data) ? data : data.tasks ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Unable to load tasks.');
    } finally {
      setLoading(false);
    }
  }, [hostId]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    if (!hostId) return;
    const tick = () => { if (document.visibilityState === 'visible') void load(); };
    const handle = window.setInterval(tick, REFRESH_MS);
    document.addEventListener('visibilitychange', tick);
    return () => {
      window.clearInterval(handle);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [load, hostId]);

  async function stop(t: Task) {
    setStopping(t.upid);
    try {
      await api('DELETE', `/api/v1/proxmox/hosts/${hostId}/nodes/${encodeURIComponent(t.node)}/tasks/${encodeURIComponent(t.upid)}`);
      await load();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Stop failed.');
    } finally {
      setStopping(null);
    }
  }

  const types = Array.from(new Set(tasks.map((t) => t.type))).sort();
  const filtered = tasks.filter((t) => {
    if (filterNode && t.node !== filterNode) return false;
    if (filterStatus && t.status !== filterStatus) return false;
    if (filterType && t.type !== filterType) return false;
    return true;
  });

  return (
    <ProxmoxShell title="Tasks" subtitle="Cluster task log">
      <motion.div className="px-vm-page" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
      <h1 className="px-detail-name">Cluster tasks</h1>
      <div className="px-detail-meta">
        <span><strong>Total</strong> {tasks.length}</span>
        <span><strong>Showing</strong> {filtered.length}</span>
      </div>

      {error && <div className="px-error">⚠ {error}</div>}

      <div className="px-filterbar" style={{ marginBottom: 12 }}>
        <select value={filterNode} onChange={(e) => setFilterNode(e.target.value)} className="px-form-select">
          <option value="">All nodes</option>
          {nodes.map((n) => <option key={n.node} value={n.node}>{n.node}</option>)}
        </select>
        <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className="px-form-select">
          <option value="">All statuses</option>
          {['running', 'OK', 'error', 'stopped'].map((s) => <option key={s} value={s}>{s}</option>)}
        </select>
        <select value={filterType} onChange={(e) => setFilterType(e.target.value)} className="px-form-select">
          <option value="">All types</option>
          {types.map((t) => <option key={t} value={t}>{t}</option>)}
        </select>
        <button type="button" className="px-btn" onClick={() => void load()} disabled={loading}>
          ⟳ Refresh
        </button>
      </div>

      {loading && tasks.length === 0 ? (
        <p className="px-muted">Loading tasks…</p>
      ) : filtered.length === 0 ? (
        <div className="px-info-card">
          <h3>No tasks</h3>
          <p>No tasks match the current filters.</p>
        </div>
      ) : (
        <table className="px-network-table">
          <thead>
            <tr>
              <th>UPID</th>
              <th>Node</th>
              <th>Type</th>
              <th>Status</th>
              <th>User</th>
              <th>Started</th>
              <th>Duration</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((t) => {
              const duration = t.endtime ? `${t.endtime - t.starttime}s` : t.starttime ? 'running' : '—';
              return (
                <tr key={t.upid}>
                  <td className="px-mono" style={{ fontSize: 11 }}>{t.upid.split(':').pop() ?? t.upid}</td>
                  <td className="px-mono">{t.node}</td>
                  <td>{t.type}</td>
                  <td>
                    <span className={`px-action-badge px-action-${(t.status || 'accept').toLowerCase()}`}>
                      {t.status}
                    </span>
                  </td>
                  <td>{t.user || '—'}</td>
                  <td className="px-mono">{t.starttime ? new Date(t.starttime * 1000).toLocaleString() : '—'}</td>
                  <td className="px-mono">{duration}</td>
                  <td>
                    {t.status === 'running' && (
                      <button
                        type="button"
                        className="px-btn px-btn-quiet px-btn-danger-text"
                        onClick={() => stop(t)}
                        disabled={stopping === t.upid}
                      >
                        ⏹ Stop
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </motion.div>
    </ProxmoxShell>
  );
}