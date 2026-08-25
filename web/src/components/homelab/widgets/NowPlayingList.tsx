import { fmtProgressMs, kindLabel, progressPercent } from './mediaFormatters';
import type { MediaNowPlayingRow } from './types';

/**
 * NowPlayingList — the list of active playback sessions rendered
 * inside MediaWidget. Split out so the widget stays under the
 * 400-LOC cap.
 *
 * One row per active session with title + user + progress bar +
 * transcoding "T" badge. The list is hidden when nothing is
 * playing (the common case for a household server during the
 * day).
 */
interface NowPlayingListProps {
  sessions: MediaNowPlayingRow[];
}

export default function NowPlayingList({ sessions }: NowPlayingListProps) {
  if (sessions.length === 0) return null;
  return (
    <div style={{ marginTop: 24 }}>
      <h4 style={{ margin: '0 0 8px 0', fontSize: 14, opacity: 0.8 }}>
        Now playing
      </h4>
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 6,
        }}
      >
        {sessions.map((s) => {
          const pct = progressPercent(s.progress_ms, s.duration_ms);
          return (
            <div
              key={s.id}
              className="homelab-service-card"
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 6,
                padding: '10px 12px',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  flexWrap: 'wrap',
                }}
              >
                <strong style={{ fontSize: 13 }}>{s.title}</strong>
                {s.transcoding ? (
                  <span
                    className="homelab-service-pill"
                    data-status="unknown"
                    style={{ fontSize: 9, padding: '1px 5px' }}
                    title="Server is transcoding this stream"
                  >
                    T
                  </span>
                ) : null}
                <span
                  style={{
                    fontSize: 11,
                    opacity: 0.65,
                    marginLeft: 'auto',
                  }}
                >
                  {s.user_name || 'unknown user'}
                  {s.player ? ` · ${s.player}` : ''}
                  {' · '}
                  {kindLabel(s.server_kind)}
                </span>
              </div>
              <div
                style={{
                  width: '100%',
                  height: 4,
                  background: 'var(--border, rgba(255,255,255,0.1))',
                  borderRadius: 2,
                  overflow: 'hidden',
                }}
                role="progressbar"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={pct}
              >
                <div
                  style={{
                    width: `${pct}%`,
                    height: '100%',
                    background: 'var(--accent, #60a5fa)',
                    transition: 'width 0.4s ease',
                  }}
                />
              </div>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  fontSize: 10,
                  opacity: 0.6,
                }}
              >
                <span>{fmtProgressMs(s.progress_ms)}</span>
                <span>
                  {fmtProgressMs(s.duration_ms)} · {pct}%
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
