// Plex poller for the MediaWorker + the
// handler.PollMediaServerAndInsertState wrapper.
//
// Endpoints polled:
//
//	GET /library/recentlyAdded?X-Plex-Container-Size=10
//	  → recent additions (Plex returns ~10 items by default;
//	    X-Plex-Container-Size caps the result set per the
//	    documented Plex API)
//	GET /status/sessions
//	  → active playback sessions (Plex uses the same MediaContainer
//	    shape as /library/recentlyAdded; we re-decode each
//	    Metadata element into plexSession to pull the per-session
//	    fields like TranscodeSession)
//
// Auth: Plex uses the X-Plex-Token request header — set on
// every request below. Token-as-query-param (?X-Plex-Token=) is
// also supported by Plex but the header is the documented /
// preferred way.
//
// Split out from media_clients.go so this file stays under the
// 400-LOC cap.
package homelab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// plexMediaContainer is the wrapper Plex returns for every
// metadata endpoint. The MediaContainer field holds the per-
// endpoint payload. Different endpoints use different fields
// inside MediaContainer (Metadata for items + sessions;
// Directory for library lists).
type plexMediaContainer struct {
	MediaContainer plexMediaContainerBody `json:"MediaContainer"`
}

type plexMediaContainerBody struct {
	// Size is the number of items Plex found, regardless of
	// pagination. We ignore it and just iterate Metadata — Plex
	// returns one entry per item when X-Plex-Container-Size is
	// set.
	Size     int             `json:"size"`
	Metadata []plexMediaItem `json:"Metadata"`
}

// plexMediaItem is the subset of a Plex metadata row we read for
// recently-added items.
type plexMediaItem struct {
	RatingKey string `json:"ratingKey"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Year      int    `json:"year"`
	Thumb     string `json:"thumb"`
	AddedAt   int64  `json:"addedAt"`
}

// plexSession is the subset of a Plex /status/sessions row we
// read. TranscodeSession is present (and non-null) when the
// server is transcoding the stream — used for the "T" badge in
// the widget.
type plexSession struct {
	SessionID string `json:"Session"`
	Title     string `json:"title"`
	Player    struct {
		Title string `json:"title"`
	} `json:"Player"`
	User struct {
		Title string `json:"title"`
	} `json:"User"`
	TranscodeSession *string `json:"TranscodeSession"`
	ViewOffset       int64   `json:"viewOffset"`
	Duration         int64   `json:"duration"`
}

// pollPlex fetches the recent additions + active sessions from a
// Plex Media Server. Both endpoints require the X-Plex-Token
// header (set below) — no other auth is supported.
//
// Partial-state return: if the recent-additions call succeeds but
// the sessions call fails, we return the recent-additions rows +
// a wrapped error so the dashboard can render the recent list
// while showing the error pill on the server chip.
func pollPlex(ctx context.Context, baseURL, apiKey string) (*MediaState, error) {
	joined, err := joinMediaBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeMediaClient()

	state := &MediaState{}

	// Recent additions — Plex returns ~10 items by default.
	itemsReq, ierr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%slibrary/recentlyAdded?X-Plex-Container-Size=10", joined), nil)
	if ierr != nil {
		return nil, fmt.Errorf("new recentlyAdded request: %w", ierr)
	}
	itemsReq.Header.Set("X-Plex-Token", apiKey)
	var itemsResp plexMediaContainer
	if _, itemsErr := readMediaJSONBody(ctx, c, itemsReq, &itemsResp); itemsErr == nil {
		for _, m := range itemsResp.MediaContainer.Metadata {
			if m.RatingKey == "" || m.Title == "" {
				continue
			}
			addedAt := time.Unix(m.AddedAt, 0).UTC()
			state.RecentAdditions = append(state.RecentAdditions, MediaItem{
				ItemID:    m.RatingKey,
				Title:     m.Title,
				Kind:      plexKindNormalize(m.Type),
				Year:      plexYearPtr(m.Year),
				PosterURL: joined + plexThumbPath(m.Thumb, m.RatingKey),
				AddedAt:   addedAt,
			})
		}
	}

	// Active sessions.
	sessReq, serr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sstatus/sessions", joined), nil)
	if serr != nil {
		return nil, fmt.Errorf("new sessions request: %w", serr)
	}
	sessReq.Header.Set("X-Plex-Token", apiKey)
	sessReq.Header.Set("Accept", "application/json")
	var sessResp plexMediaContainer
	if _, sessErr := readMediaJSONBody(ctx, c, sessReq, &sessResp); sessErr != nil {
		// Sessions fetch failed — return what we have (recent
		// additions may be present from the earlier call).
		if len(state.RecentAdditions) > 0 {
			return state, fmt.Errorf("sessions fetch: %w", sessErr)
		}
		return state, fmt.Errorf("sessions fetch: %w", sessErr)
	}
	// plexSession is structurally similar to plexMediaItem at
	// the top level (Session, Title, etc.) but we decode into
	// a separate type for clarity. Re-decode each item
	// individually so we can pull the TranscodeSession field
	// without breaking the items response shape.
	for _, m := range sessResp.MediaContainer.Metadata {
		var s plexSession
		raw, _ := json.Marshal(m)
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		if s.SessionID == "" {
			continue
		}
		state.NowPlaying = append(state.NowPlaying, MediaSession{
			SessionID:   s.SessionID,
			Title:       s.Title,
			UserName:    s.User.Title,
			Player:      s.Player.Title,
			Transcoding: s.TranscodeSession != nil && *s.TranscodeSession != "",
			ProgressMs:  s.ViewOffset,
			DurationMs:  s.Duration,
		})
	}

	return state, nil
}

// plexKindNormalize maps Plex's type field to the lower-case
// vocabulary used by the widget ('movie' | 'show' | 'episode').
// Plex uses 'movie', 'show', 'season', 'episode', 'track',
// 'album', 'artist' — we keep just the three the widget cares
// about.
func plexKindNormalize(t string) string {
	switch strings.ToLower(t) {
	case "movie":
		return "movie"
	case "show", "season":
		return "show"
	case "episode":
		return "episode"
	default:
		// Music + photo items don't render in the recent-
		// additions carousel. Default to "movie" so a future
		// item type doesn't crash the widget.
		return "movie"
	}
}

// plexYearPtr returns a *int (nullable) only when the year is
// positive. Plex returns 0 for items without a year (e.g.
// episodes on a season listing); we treat that as "no year".
func plexYearPtr(y int) *int {
	if y <= 0 {
		return nil
	}
	return &y
}

// plexThumbPath returns the path component Plex uses for poster
// thumbnails. Plex serves posters at /library/metadata/<key>/thumb
// when given just the rating key. When the item already has a
// full thumb path (e.g. /library/metadata/123/thumb/1234567890)
// we preserve it as-is. The apiKey parameter is reserved for
// future hardening (e.g. embedding the token in the image URL for
// Plex servers that require auth on /library/parts/* requests).
func plexThumbPath(thumb, ratingKey string) string {
	if thumb != "" {
		return strings.TrimPrefix(thumb, "/")
	}
	if ratingKey != "" {
		return "library/metadata/" + ratingKey + "/thumb"
	}
	return ""
}
