import { useState } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';
import { ApiError, api } from '../../lib/api';

/**
 * TrainAnomalyModelModal — Tier 8.1 modal for "Train new model" on
 * the Intelligence page. Extracted from IntelligencePage.tsx so
 * that file stays under the 400-LOC cap after Phase 4 added the
 * NoiseReductionSection below.
 *
 * Form fields:
 *   - metric_name     (required)
 *   - server_id       (optional — blank = tenant-wide)
 *   - model_type      ('welford' | 'ewma', default welford)
 *
 * Submission calls POST /api/v1/anomaly/train, then onSaved() so
 * the parent can reload the model list + events.
 */

interface TrainAnomalyModelModalProps {
  busy: boolean;
  onClose: () => void;
  onSaved: () => void;
  onError: (msg: string) => void;
}

export default function TrainAnomalyModelModal({
  busy,
  onClose,
  onSaved,
  onError,
}: TrainAnomalyModelModalProps) {
  const reduce = useReducedMotion();
  const [metric, setMetric] = useState('');
  const [server, setServer] = useState('');
  const [modelType, setModelType] = useState<'welford' | 'ewma'>('welford');
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!metric.trim()) {
      onError('Metric name is required.');
      return;
    }
    setSubmitting(true);
    try {
      const body: Record<string, string> = {
        metric_name: metric.trim(),
        model_type: modelType,
      };
      if (server.trim()) body.server_id = server.trim();
      await api('POST', '/api/v1/anomaly/train', body);
      onSaved();
    } catch (cause) {
      onError(cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  const isBusy = busy || submitting;

  return (
    <div className="slow-query-explain-modal" role="dialog" aria-modal="true">
      <div className="slow-query-explain-modal-head">
        <strong>Train new model</strong>
        <button
          type="button"
          className="slow-query-explain-close"
          onClick={onClose}
          aria-label="Close"
        >
          ✕
        </button>
      </div>
      <form
        className="notebook-create-form"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <label>
          <span>Metric name</span>
          <input
            type="text"
            required
            maxLength={256}
            value={metric}
            onChange={(e) => setMetric(e.target.value)}
            placeholder="cpu.user_pct"
          />
        </label>
        <label>
          <span>Server ID (optional — leave blank for tenant-wide)</span>
          <input
            type="text"
            value={server}
            onChange={(e) => setServer(e.target.value)}
            placeholder="uuid of the server, or blank for tenant-wide"
          />
        </label>
        <label>
          <span>Model type</span>
          <select
            value={modelType}
            onChange={(e) => setModelType(e.target.value as 'welford' | 'ewma')}
          >
            <option value="welford">Welford + EWMA (recommended)</option>
            <option value="ewma">EWMA only</option>
          </select>
        </label>
        <div className="incident-create-actions">
          <motion.button
            type="button"
            className="dash-icon-button"
            onClick={onClose}
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={isBusy}
          >
            Cancel
          </motion.button>
          <motion.button
            type="submit"
            className="empty-state-cta"
            whileHover={reduce ? undefined : buttonSpring.whileHover}
            whileTap={reduce ? undefined : buttonSpring.whileTap}
            transition={buttonSpring.transition}
            disabled={isBusy || !metric.trim()}
          >
            {isBusy ? 'Training…' : 'Train'}
          </motion.button>
        </div>
      </form>
    </div>
  );
}
