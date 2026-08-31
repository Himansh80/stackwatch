import { useCallback, useEffect, useMemo, useState } from 'react';
import { ApiError, api, getToken } from '../../../lib/api';
import { motion, pageEnter } from '../../../lib/motion';
import Button from '../../shared/Button';
import ServiceTile from './ServiceTile';
import PinModal from './PinModal';
import type {
  AggregateResponse,
  PinFormState,
  PinnedService,
  ServicesResponse,
  ServiceKind,
} from './types';

/**
 * ServiceStatusWidget — Tier 10 Phase 2 (H2 — Service Status).
 *
 * The Services tab content on the HomelabPage. Loads the caller's
 * pinned services via:
 *   GET /api/v1/homelab/services          — list of pins + latest health
 *   GET /api/v1/homelab/services/aggregate — up/degraded/down/unknown summary
 *
 * And writes via:
 *   POST   /api/v1/homelab/services          — pin a new service (modal)
 *   PATCH  /api/v1/homelab/services/:id      — update a pin (inline edit)
 *   DELETE /api/v1/homelab/services/:id      — unpin
 *   POST   /api/v1/homelab/services/:id/probe — one-shot probe
 *   POST   /api/v1/homelab/services/probe-all  — fire-and-forget batch
 *
 * Status pills use the same color tokens as the KPI strip:
 *   up=green, degraded=amber, down=red, unknown=gray (var(--surface-3)).
 *
 * Pin count cap (50/user) is enforced server-side; we surface a
 * friendly error from the API rather than enforcing client-side
 * (single source of truth).
 *
 * Motion: pageEnter on the wrapper. No new variants — reuses the
 * tokens defined in lib/motion.tsx.
 */

const emptyPin: PinFormState = { name: '', url: '', kind: 'http', icon: '' };

export default function ServiceStatusWidget({ config: _config }: { config?: { limit?: number } }) {
  // `config` is reserved for future options (limit, sort, filter);
  // destructured with the `_` prefix so the no-unused-vars lint
  // doesn't complain while keeping the prop wired through.
  void _config;
  const [services, setServices] = useState<PinnedService[]>([]);
  const [aggregate, setAggregate] = useState<AggregateResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [pinFormOpen, setPinFormOpen] = useState(false);
  const [pinForm, setPinForm] = useState<PinFormState>(emptyPin);
  const [pinFormBusy, setPinFormBusy] = useState(false);
  const [pinFormError, setPinFormError] = useState('');
  const [probeBusyId, setProbeBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!getToken()) {
      setError('Sign in to view your pinned services.');
      setLoading(false);
      return;
    }
    try {
      const [list, agg] = await Promise.all([
        api<ServicesResponse>('GET', '/api/v1/homelab/services'),
        api<AggregateResponse>('GET', '/api/v1/homelab/services/aggregate'),
      ]);
      setServices(list.services || []);
      setAggregate(agg);
      setError('');
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const openPinForm = useCallback(() => {
    setPinForm(emptyPin);
    setPinFormError('');
    setPinFormOpen(true);
  }, []);

  const closePinForm = useCallback(() => {
    setPinFormOpen(false);
  }, []);

  const submitPin = useCallback(async () => {
    setPinFormBusy(true);
    setPinFormError('');
    try {
      const icon = pinForm.icon.trim();
      const body: { name: string; url: string; kind: ServiceKind; icon?: string } = {
        name: pinForm.name.trim(),
        url: pinForm.url.trim(),
        kind: pinForm.kind,
      };
      if (icon) body.icon = icon;
      await api('POST', '/api/v1/homelab/services', body);
      setPinFormOpen(false);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setPinFormError(msg);
    } finally {
      setPinFormBusy(false);
    }
  }, [pinForm, load]);

  const deletePin = useCallback(async (id: string) => {
    if (!window.confirm('Unpin this service? Health history will be deleted too.')) return;
    try {
      await api('DELETE', `/api/v1/homelab/services/${id}`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load]);

  const toggleEnabled = useCallback(async (svc: PinnedService) => {
    try {
      await api('PATCH', `/api/v1/homelab/services/${svc.id}`, { enabled: !svc.enabled });
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load]);

  const probeOne = useCallback(async (id: string) => {
    setProbeBusyId(id);
    try {
      await api('POST', `/api/v1/homelab/services/${id}/probe`);
      await load();
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    } finally {
      setProbeBusyId(null);
    }
  }, [load]);

  const probeAll = useCallback(async () => {
    try {
      await api('POST', '/api/v1/homelab/services/probe-all');
      // Worker + probe results land within a few seconds. Defer
      // the refresh so the UI feels responsive.
      window.setTimeout(() => { void load(); }, 2500);
    } catch (cause) {
      const msg = cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message;
      setError(msg);
    }
  }, [load]);

  const tiles = useMemo(() => services.map((svc) => (
    <ServiceTile
      key={svc.id}
      svc={svc}
      probing={probeBusyId === svc.id}
      onClick={() => window.open(svc.url, '_blank', 'noopener,noreferrer')}
      onProbe={() => void probeOne(svc.id)}
      onToggle={() => void toggleEnabled(svc)}
      onDelete={() => void deletePin(svc.id)}
    />
  )), [services, probeBusyId, probeOne, toggleEnabled, deletePin]);

  if (loading) {
    return (
      <div className="homelab-grid-loading" role="status" aria-live="polite">
        Loading your pinned services…
      </div>
    );
  }

  return (
    <motion.div
      className="homelab-services-widget"
      initial="hidden"
      animate="show"
      variants={pageEnter}
    >
      {error ? <div className="dash-error" role="alert">{error}</div> : null}

      <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
        <Button variant="primary" size="sm" onClick={openPinForm}>
          + Pin service
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void probeAll()}
          disabled={services.length === 0}
        >
          Probe all
        </Button>
      </div>

      {services.length === 0 ? (
        <div className="homelab-services-empty">
          <p className="homelab-services-empty-title">No services pinned yet</p>
          <p className="homelab-services-empty-hint">
            Pin your Proxmox dashboard, Jellyfin, Pi-hole, NAS, or any
            self-hosted service to see its live status here. The
            background worker probes every pinned service every 60s.
          </p>
        </div>
      ) : (
        <div className="homelab-services-grid">{tiles}</div>
      )}

      {aggregate ? (
        <div className="homelab-services-footer">
          <div className="homelab-services-aggregate">
            <span className="homelab-services-aggregate-dot" data-status="up">{aggregate.up} up</span>
            <span className="homelab-services-aggregate-dot" data-status="degraded">{aggregate.degraded} degraded</span>
            <span className="homelab-services-aggregate-dot" data-status="down">{aggregate.down} down</span>
            {aggregate.unknown > 0 ? <span>{aggregate.unknown} unknown</span> : null}
          </div>
          <span>{aggregate.total} pinned</span>
        </div>
      ) : null}

      {pinFormOpen ? (
        <PinModal
          form={pinForm}
          setForm={setPinForm}
          busy={pinFormBusy}
          error={pinFormError}
          onClose={closePinForm}
          onSubmit={() => void submitPin()}
        />
      ) : null}
    </motion.div>
  );
}