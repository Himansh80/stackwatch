import { useEffect, useState } from 'react';
import { ApiError, api, getToken } from '../../lib/api';
import { motion, pageEnter, kpiStagger } from '../../lib/motion';
import Button from '../shared/Button';
import EmptyState from '../shared/EmptyState';
import KpiCard from '../shared/KpiCard';
import StatusPill from '../shared/StatusPill';
import { RegionsAddModal, RegionsDetailModal } from './RegionsSectionModals';

// Tier 11 Phase 6 — Multi-Region / HA (PL6).
//
// RegionsSection renders the platform-wide region catalog +
// health surface inside the unified PlatformPage. The catalog
// is global (no tenant scoping) so every super_admin sees the
// same set of regions; per-tenant scoping enters via the
// (already-existing) tenants.region_code column.
//
// Layout (top → bottom):
//   - Header       : "Regions" + "Add region" button (super_admin only)
//                    + "Probe health" button
//   - KPI strip    : 4 tiles — total regions / primary regions /
//                    up regions / degraded regions
//   - Region list  : one row per region with status pill + latency
//                    + last probe timestamp
//   - Empty state  : when no regions exist, friendly explanation
//                    + "Add your first region" CTA
//
// Data sources:
//   GET  /api/v1/platform/regions         — list catalog
//   POST /api/v1/platform/regions         — add region (super_admin)
//   GET  /api/v1/platform/regions/health  — probe every region
//
// Modal subcomponents live in RegionsSectionModals.tsx (kept
// out of this file so it stays under the 400-LOC cap).
//
// Auth: every load bails early when no JWT is present so we
// don't spam a 401 from the platform-admin login flow.
// Mutations (Add) require Role == super_admin.
//
// Motion : pageEnter on the section wrapper; kpiStagger on
// the KPI strip; buttonSpring on the action buttons. No new
// variants — reuses the existing motion primitives from
// src/lib/motion.tsx.

export interface RegionRow {
  id: string;
  code: string;
  display_name: string;
  region_kind: 'primary' | 'replica' | 'standby';
  endpoint_url: string;
  is_active: boolean;
  last_health_at?: string;
  last_health_status?: string;
  last_health_latency_ms?: number;
  last_health_error?: string;
  created_at: string;
  updated_at: string;
}

interface RegionList {
  regions: RegionRow[];
}

interface AddRegionBody {
  code: string;
  display_name: string;
  region_kind: string;
  endpoint_url: string;
}

interface RegionsSectionProps {
  /** When true, show the "Add region" button. Restricted to
   *  super_admin (a regular admin can't add infrastructure). */
  canMutate: boolean;
}

export default function RegionsSection({ canMutate }: RegionsSectionProps) {
  const [list, setList] = useState<RegionList | null>(null);
  const [busy, setBusy] = useState(false);
  const [probing, setProbing] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');
  const [modalRegion, setModalRegion] = useState<RegionRow | null>(null);
  const [addOpen, setAddOpen] = useState(false);

  // Add-region form fields (controlled — keeps the body valid
  // before submit so we can show inline errors).
  const [addCode, setAddCode] = useState('');
  const [addName, setAddName] = useState('');
  const [addKind, setAddKind] = useState('primary');
  const [addUrl, setAddUrl] = useState('');
  const [addErr, setAddErr] = useState('');

  const requireAuth = (): boolean => {
    if (!getToken()) {
      setError('Sign in to view regions.');
      return false;
    }
    return true;
  };

  const load = async () => {
    if (!requireAuth()) return;
    setBusy(true);
    setError('');
    try {
      const data = await api<RegionList>(
        'GET',
        '/api/v1/platform/regions'
      );
      setList(data);
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    } finally {
      setBusy(false);
    }
  };

  const handleProbe = async () => {
    if (!requireAuth()) return;
    setProbing(true);
    setError('');
    try {
      await api('GET', '/api/v1/platform/regions/health');
      // Refresh the catalog so the freshly-probed
      // last_health_* columns are visible.
      await load();
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    } finally {
      setProbing(false);
    }
  };

  const handleAdd = async () => {
    if (!requireAuth()) return;
    setAddErr('');
    const code = addCode.trim().toLowerCase();
    if (!/^[a-z0-9-]{2,32}$/.test(code)) {
      setAddErr('Code must match ^[a-z0-9-]{2,32}$ (lowercase URL-safe).');
      return;
    }
    if (!addName.trim() || !addUrl.trim()) {
      setAddErr('Display name + endpoint URL are required.');
      return;
    }
    setCreating(true);
    try {
      const body: AddRegionBody = {
        code,
        display_name: addName.trim(),
        region_kind: addKind,
        endpoint_url: addUrl.trim(),
      };
      await api('POST', '/api/v1/platform/regions', body);
      setAddOpen(false);
      setAddCode('');
      setAddName('');
      setAddKind('primary');
      setAddUrl('');
      await load();
    } catch (cause) {
      setAddErr(
        cause instanceof ApiError
          ? cause.friendlyMessage
          : (cause as Error).message
      );
    } finally {
      setCreating(false);
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const regions: RegionRow[] = list?.regions ?? [];
  const primaryCount = regions.filter(
    (r) => r.region_kind === 'primary'
  ).length;
  const upCount = regions.filter(
    (r) => r.last_health_status === 'up'
  ).length;
  const degradedCount = regions.filter(
    (r) => r.last_health_status === 'degraded'
  ).length;
  const showEmpty = !busy && regions.length === 0;

  return (
    <motion.div
      initial="hidden"
      animate="show"
      variants={pageEnter}
      className="space-y-6 p-2"
    >
      {/* Header */}
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">Regions</h2>
          <p className="text-sm opacity-70">
            Multi-region / HA catalog — primary sites, replicas, and
            standby instances.
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void handleProbe()}
            disabled={probing || busy}
            loading={probing}
          >
            {probing ? 'Probing…' : 'Probe health'}
          </Button>
          {canMutate ? (
            <Button
              variant="primary"
              size="sm"
              onClick={() => setAddOpen(true)}
              disabled={busy}
            >
              + Add region
            </Button>
          ) : null}
        </div>
      </header>

      {error ? (
        <div className="callout-error" role="alert">
          {error}
        </div>
      ) : null}

      {/* KPI strip */}
      <motion.section
        variants={kpiStagger}
        initial="hidden"
        animate="show"
        className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3"
      >
        <KpiCard label="Total regions" value={regions.length} />
        <KpiCard
          label="Primary"
          value={primaryCount}
          accent="indigo"
        />
        <KpiCard
          label="Up"
          value={upCount}
          status="up"
          accent="green"
        />
        <KpiCard
          label="Degraded"
          value={degradedCount}
          status={degradedCount > 0 ? 'stale' : 'neutral'}
          accent={degradedCount > 0 ? 'amber' : 'cyan'}
        />
      </motion.section>

      {/* Region list OR empty state */}
      {showEmpty ? (
        <EmptyState
          illustration={<span aria-hidden="true">🌐</span>}
          headline="No regions configured yet"
          subhead="Add a primary region to start tracking health across sites. Replicas and standby instances can be added next."
          cta={
            canMutate
              ? {
                  label: 'Add your first region',
                  onClick: () => setAddOpen(true),
                }
              : undefined
          }
        />
      ) : (
        <section className="dash-table-wrap" aria-label="Regions">
          <table className="dash-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Display name</th>
                <th>Kind</th>
                <th>Endpoint</th>
                <th>Status</th>
                <th>Latency</th>
                <th>Last probe</th>
              </tr>
            </thead>
            <tbody>
              {regions.map((r) => (
                <tr
                  key={r.id}
                  onClick={() => setModalRegion(r)}
                  className="cursor-pointer hover:bg-[var(--surface-2)]"
                >
                  <td className="font-mono">{r.code}</td>
                  <td>{r.display_name}</td>
                  <td>
                    <span className="dash-status dash-status-neutral">
                      {r.region_kind}
                    </span>
                  </td>
                  <td className="font-mono text-xs opacity-70">
                    {r.endpoint_url}
                  </td>
                  <td>
                    <StatusPill status={r.last_health_status || 'unknown'} />
                  </td>
                  <td>
                    {r.last_health_latency_ms != null
                      ? `${r.last_health_latency_ms} ms`
                      : '—'}
                  </td>
                  <td>
                    {r.last_health_at
                      ? new Date(r.last_health_at).toLocaleString()
                      : 'never'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      <RegionsDetailModal
        region={modalRegion}
        onClose={() => setModalRegion(null)}
      />
      <RegionsAddModal
        open={addOpen}
        creating={creating}
        addCode={addCode}
        addName={addName}
        addKind={addKind}
        addUrl={addUrl}
        addErr={addErr}
        onChangeCode={setAddCode}
        onChangeName={setAddName}
        onChangeKind={setAddKind}
        onChangeUrl={setAddUrl}
        onSubmit={handleAdd}
        onClose={() => !creating && setAddOpen(false)}
      />
    </motion.div>
  );
}