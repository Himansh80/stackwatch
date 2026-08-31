import { useCallback, useMemo, useState } from 'react';
import CorrelationCard, {
  CorrelationGroup,
  CorrelationMember,
} from './shared/CorrelationCard';
import EmptyState from './shared/EmptyState';
import KpiCard from './shared/KpiCard';
import Button from './shared/Button';
import Input from './shared/Input';
import Modal from './shared/Modal';
import Textarea from './shared/Textarea';
import { ApiError, api } from '../lib/api';
import { motion, kpiStagger } from '../lib/motion';

/**
 * CorrelationsSection — Tier 8.3 (D9 Phase 3) UI surface for the
 * Alert Correlation + RCA page section. Mirrors the Phase 2
 * PredictiveAlertsSection pattern: section owns its modal +
 * createManualCorrelation logic so the IntelligencePage stays
 * under the 400-LOC cap.
 *
 * Layout:
 *   - 3 KpiCards: groups today / auto-detected ratio / feedback received
 *   - Top 5 correlation groups rendered as CorrelationCards
 *     (sorted by similarity_score DESC; ties broken by created_at DESC)
 *   - "Manual correlate" button + form modal (POST /correlations/manual)
 *   - Feedback buttons live inline on each card (handled by CorrelationCard)
 *
 * Tier 20 Phase E: the custom slow-query-explain-modal frame +
 * raw <input>/<textarea> + empty-state-cta/dash-icon-button markup
 * is replaced with the dashboard's shared <Modal>/<Input>/
 * <Textarea>/<Button> primitives.
 */

interface CorrelationsSectionProps {
  groups: CorrelationGroup[];
  busy: boolean;
  onError: (msg: string) => void;
  onCreated?: (groupId: string) => void;
}

export default function CorrelationsSection({
  groups,
  busy,
  onError,
  onCreated,
}: CorrelationsSectionProps) {
  const [sectionBusy, setSectionBusy] = useState(false);
  const isBusy = busy || sectionBusy;

  // Manual correlate modal state.
  const [modal, setModal] = useState(false);
  const [alertIDsRaw, setAlertIDsRaw] = useState('');
  const [reason, setReason] = useState('');
  const [similarity, setSimilarity] = useState('1.0');

  // Per-group expanded details cache: groupId → members fetched via
  // GET /correlations/group/:id. Keeps the page lightweight; if the
  // user expands a card we fetch once and cache forever (within this
  // session).
  const [memberCache, setMemberCache] = useState<Record<string, CorrelationMember[]>>({});

  // KPIs computed locally from the groups list + cache.
  const counts = useMemo(() => {
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;
    let today = 0;
    let auto = 0;
    for (const g of groups) {
      const ts = new Date(g.created_at).getTime();
      if (ts >= oneDayAgo) today += 1;
      if (g.auto_detected) auto += 1;
    }
    return {
      today,
      auto,
      ratio: groups.length === 0 ? 0 : Math.round((auto / groups.length) * 100),
      feedback: Object.keys(memberCache).length,
    };
  }, [groups, memberCache]);

  // Sort top-5 by similarity_score DESC; ties by created_at DESC.
  const top = useMemo(
    () =>
      [...groups]
        .sort((a, b) => {
          if (b.similarity_score !== a.similarity_score) {
            return b.similarity_score - a.similarity_score;
          }
          return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
        })
        .slice(0, 5),
    [groups],
  );

  // Expand handler — fetch /group/:id and cache the members.
  const onExpand = useCallback(
    async (groupId: string) => {
      if (memberCache[groupId]) return;
      try {
        const resp = await api<{
          members?: CorrelationMember[];
          group?: { correlation_id: string };
        }>('GET', `/api/v1/correlations/group/${groupId}`);
        const members = resp.members || [];
        setMemberCache((prev) => ({ ...prev, [groupId]: members }));
      } catch (cause) {
        // Non-fatal — the card still renders without the expanded list.
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      }
    },
    [memberCache, onError],
  );

  // Feedback handler — POST /correlations/feedback.
  const onFeedback = useCallback(
    async (groupId: string, useful: boolean) => {
      setSectionBusy(true);
      try {
        await api('POST', '/api/v1/correlations/feedback', {
          correlation_id: groupId,
          useful,
          note: '',
        });
      } catch (cause) {
        onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
      } finally {
        setSectionBusy(false);
      }
    },
    [onError],
  );

  // Manual correlate submit handler.
  const submitManual = useCallback(async () => {
    const raw = alertIDsRaw.trim();
    if (!raw) {
      onError('At least two alert IDs are required.');
      return;
    }
    // Accept comma- or newline-separated UUIDs.
    const parsed = raw
      .split(/[\s,]+/)
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
    if (parsed.length < 2) {
      onError('At least two distinct alert IDs are required.');
      return;
    }
    const sim = parseFloat(similarity);
    const body: Record<string, unknown> = {
      alert_ids: parsed,
      reason: reason.trim(),
    };
    if (!Number.isNaN(sim) && sim >= 0 && sim <= 1) {
      body.similarity_score = sim;
    }
    setSectionBusy(true);
    try {
      const resp = await api<{ correlation_id?: string }>(
        'POST',
        '/api/v1/correlations/manual',
        body,
      );
      setModal(false);
      setAlertIDsRaw('');
      setReason('');
      setSimilarity('1.0');
      if (resp.correlation_id && onCreated) {
        onCreated(resp.correlation_id);
      }
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSectionBusy(false);
    }
  }, [alertIDsRaw, similarity, reason, onError, onCreated]);

  // Hydrate each top card's members from the cache (if any).
  const hydratedTop = useMemo(
    () =>
      top.map((g) => ({
        ...g,
        members: memberCache[g.correlation_id] || [],
      })),
    [top, memberCache],
  );

  return (
    <>
      <motion.div
        className="dash-metric-strip"
        initial="hidden"
        animate="show"
        variants={kpiStagger}
        style={{ marginTop: 16 }}
      >
        <KpiCard
          label="Correlations (24h)"
          value={counts.today}
          status={counts.today > 0 ? 'crit' : 'up'}
          accent={counts.today > 0 ? 'red' : 'cyan'}
        />
        <KpiCard
          label="Auto-detected ratio"
          value={`${counts.ratio}%`}
          status="neutral"
          accent="indigo"
        />
        <KpiCard
          label="Expanded views"
          value={counts.feedback}
          status="neutral"
          accent="cyan"
        />
      </motion.div>

      <section className="dash-section">
        <span className="dash-eyebrow">Alert Correlation + RCA</span>
        <h2 className="dash-section-title">Top correlation groups</h2>
        <p
          className="dash-section-sub"
          style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 0 }}
        >
          Each card is one correlation group — a cluster of alerts the correlator thinks
          share a root cause. Similarity score is the cluster&apos;s tightness (0..1).
          Click <em>Expand</em> to fetch the per-member severity + RCA hint, or{' '}
          <em>👍 / 👎</em> to record feedback for the model.
        </p>
        {hydratedTop.length === 0 ? (
          <EmptyState
            illustration={<span style={{ fontSize: 36 }}>⌬</span>}
            headline="No correlations yet"
            subhead="When 3+ alerts fire on the same server within 5 minutes the background correlator groups them. The cards will populate as groups are detected."
          />
        ) : (
          <div className="threat-card-list">
            {hydratedTop.map((g) => (
              <CorrelationCard
                key={g.correlation_id}
                group={g}
                busy={isBusy}
                onFeedback={onFeedback}
                onExpand={onExpand}
              />
            ))}
          </div>
        )}

        <div style={{ marginTop: 12 }}>
          <Button
            type="button"
            variant="primary"
            size="md"
            onClick={() => setModal(true)}
            disabled={isBusy}
          >
            + Manual correlate
          </Button>
        </div>
      </section>

      <Modal open={modal} onClose={() => setModal(false)} title="Create manual correlation" size="md">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void submitManual();
          }}
          style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
        >
          <Textarea
            label="Alert IDs (comma- or newline-separated, at least 2)"
            required
            rows={3}
            value={alertIDsRaw}
            onChange={(e) => setAlertIDsRaw(e.target.value)}
            placeholder="uuid-of-alert-a, uuid-of-alert-b, uuid-of-alert-c"
            fullWidth
          />
          <Input
            label="Reason"
            type="text"
            maxLength={2048}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="same root cause: deployment rollback at 14:32"
            fullWidth
          />
          <Input
            label="Similarity score (0..1, optional)"
            type="number"
            min={0}
            max={1}
            step={0.01}
            value={similarity}
            onChange={(e) => setSimilarity(e.target.value)}
            fullWidth
          />
          <p style={{ color: 'var(--text-muted)', fontSize: 11, margin: 0 }}>
            Manual correlations are tagged <code>auto_detected=false</code> so the UI
            distinguishes them from background-detected groups. The first alert_id is
            treated as the root alert.
          </p>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setModal(false)}
              disabled={isBusy}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              size="sm"
              loading={sectionBusy}
              disabled={isBusy || !alertIDsRaw.trim()}
            >
              {sectionBusy ? 'Creating…' : 'Correlate'}
            </Button>
          </div>
        </form>
      </Modal>
    </>
  );
}
