import { kindLabel } from './mediaFormatters';
import type { MediaRecentAdditionRow } from './types';

/**
 * RecentAdditionsCarousel — horizontal poster carousel rendered
 * inside MediaWidget. Split out so the widget stays under the
 * 400-LOC cap.
 *
 * One card per recent item with:
 *   - poster (with onError fallback to placeholder when the
 *     upstream image URL 404s — Plex auth can return 401 for
 *     /library/parts/* when the api_key isn't accepted)
 *   - title
 *   - year + item_kind + server_kind footer
 *
 * Hidden when no items have been added (the common case for a
 * brand-new install before the first poll lands).
 */
interface RecentAdditionsCarouselProps {
  items: MediaRecentAdditionRow[];
}

export default function RecentAdditionsCarousel({ items }: RecentAdditionsCarouselProps) {
  if (items.length === 0) return null;
  return (
    <div style={{ marginTop: 24 }}>
      <h4 style={{ margin: '0 0 8px 0', fontSize: 14, opacity: 0.8 }}>
        Recent additions
      </h4>
      <div
        style={{
          display: 'flex',
          gap: 12,
          overflowX: 'auto',
          paddingBottom: 4,
        }}
      >
        {items.map((r) => (
          <div
            key={r.id}
            className="homelab-service-card"
            style={{
              flex: '0 0 auto',
              width: 140,
              padding: 8,
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
            }}
          >
            <div
              style={{
                width: '100%',
                aspectRatio: '2 / 3',
                background: 'var(--bg-elev, rgba(255,255,255,0.05))',
                borderRadius: 4,
                overflow: 'hidden',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--fg-muted, #888)',
                fontSize: 11,
              }}
            >
              {r.poster_url ? (
                <img
                  src={r.poster_url}
                  alt={`${r.title} poster`}
                  style={{
                    width: '100%',
                    height: '100%',
                    objectFit: 'cover',
                  }}
                  loading="lazy"
                  onError={(e) => {
                    // Fall back to the placeholder if the
                    // upstream image URL 404s (Plex auth can
                    // return 401 for /library/parts/*).
                    (e.currentTarget as HTMLImageElement).style.display = 'none';
                  }}
                />
              ) : (
                <span>no poster</span>
              )}
            </div>
            <strong style={{ fontSize: 12, lineHeight: 1.2 }}>{r.title}</strong>
            <span style={{ fontSize: 10, opacity: 0.6 }}>
              {r.year ?? '—'} · {r.item_kind} · {kindLabel(r.server_kind)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
