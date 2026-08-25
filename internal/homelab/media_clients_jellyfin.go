// Jellyfin poller for the MediaWorker + the
// handler.PollMediaServerAndInsertState wrapper.
//
// Endpoints polled:
//   GET /Library/Media/Recent?limit=10
//     → recent additions (Jellyfin returns the 10 most-recently-
//       added items across all libraries)
//   GET /Sessions
//     → active playback sessions (Jellyfin returns a flat array
//       of NowPlayingItem-embedded session records)
//
// Auth: Jellyfin inherited Emby's auth scheme — both use the
// X-Emby-Token request header. The token is case-sensitive in
// older Jellyfin versions; we set the documented spelling.
//
// Time units: Jellyfin uses 100ns ticks for PositionTicks +
// RunTimeTicks. We convert to ms at the boundary so the rest of
// the handler doesn't need to know the unit.
//
// Split out from media_clients.go so this file stays under the
// 400-LOC cap. Emby uses dedicated types in
// media_clients_emby.go — the two APIs share the same endpoint
// shape but we keep the type names separate so a future Emby-
// specific quirk (e.g. Emby's stricter PremiereDate parsing) can
// be added without breaking Jellyfin.
package homelab

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// jellyfinItemsResponse is the wrapper Jellyfin returns for /Items
// endpoints. Items holds the per-item records.
type jellyfinItemsResponse struct {
	Items []jellyfinItem `json:"Items"`
}

type jellyfinItem struct {
	ID             string `json:"Id"`
	Name           string `json:"Name"`
	Type           string `json:"Type"`
	ProductionYear int    `json:"ProductionYear"`
	PremiereDate   string `json:"PremiereDate"`
	DateCreated    string `json:"DateCreated"`
	ImageTags      struct {
		Primary string `json:"Primary"`
	} `json:"ImageTags"`
}

// jellyfinSessionsResponse wraps /Sessions. Jellyfin returns an
// array (not an object) — the type alias documents that.
type jellyfinSessionsResponse []jellyfinSession

type jellyfinSession struct {
	ID        string `json:"Id"`
	PlayState *struct {
		PositionTicks int64 `json:"PositionTicks"`
	} `json:"PlayState"`
	NowPlayingItem *struct {
		Name         string `json:"Name"`
		RunTimeTicks int64 `json:"RunTimeTicks"`
		Type         string `json:"Type"`
	} `json:"NowPlayingItem"`
	Client      string `json:"Client"`
	DeviceName  string `json:"DeviceName"`
	UserName    string `json:"UserName"`
	Transcoding bool   `json:"IsTranscoding"`
}

// jellyfinTicksToMs converts Jellyfin's 100ns ticks to
// milliseconds. Integer division is fine — sub-millisecond
// accuracy doesn't matter for a progress bar.
func jellyfinTicksToMs(ticks int64) int64 {
	return ticks / 10000
}

// pollJellyfin fetches the recent additions + active sessions
// from a Jellyfin server. Both endpoints require the X-Emby-Token
// header (Jellyfin inherited Emby's auth scheme).
func pollJellyfin(ctx context.Context, baseURL, apiKey string) (*MediaState, error) {
	joined, err := joinMediaBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeMediaClient()

	state := &MediaState{}

	// Recent additions. Jellyfin's /Library/Media/Recent?limit=10
	// returns the 10 most-recently-added items across all
	// libraries. Fields= asks the server to also return
	// PrimaryImageAspectRatio + DateCreated + ProductionYear on
	// each row — the recent-additions endpoint doesn't return
	// them by default.
	itemsReq, ierr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sLibrary/Media/Recent?limit=10&Fields=PrimaryImageAspectRatio,DateCreated,ProductionYear", joined), nil)
	if ierr != nil {
		return nil, fmt.Errorf("new Recent request: %w", ierr)
	}
	itemsReq.Header.Set("X-Emby-Token", apiKey)
	var itemsResp jellyfinItemsResponse
	if _, ierr := readMediaJSONBody(ctx, c, itemsReq, &itemsResp); ierr == nil {
		for _, m := range itemsResp.Items {
			if m.ID == "" || m.Name == "" {
				continue
			}
			addedAt := jellyfinAddedAt(m)
			state.RecentAdditions = append(state.RecentAdditions, MediaItem{
				ItemID:    m.ID,
				Title:     m.Name,
				Kind:      jellyfinKindNormalize(m.Type),
				Year:      jellyfinYearPtr(m.ProductionYear),
				PosterURL: jellyfinPosterURL(joined, m),
				AddedAt:   addedAt,
			})
		}
	}

	// Active sessions.
	sessReq, serr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sSessions", joined), nil)
	if serr != nil {
		return nil, fmt.Errorf("new Sessions request: %w", serr)
	}
	sessReq.Header.Set("X-Emby-Token", apiKey)
	var sessResp jellyfinSessionsResponse
	if _, serr := readMediaJSONBody(ctx, c, sessReq, &sessResp); serr != nil {
		// Partial-state return: keep recent additions if we got
		// them.
		if len(state.RecentAdditions) > 0 {
			return state, fmt.Errorf("Sessions fetch: %w", serr)
		}
		return state, fmt.Errorf("Sessions fetch: %w", serr)
	}
	for _, s := range sessResp {
		if s.ID == "" || s.NowPlayingItem == nil {
			continue
		}
		player := s.Client
		if player == "" {
			player = s.DeviceName
		}
		state.NowPlaying = append(state.NowPlaying, MediaSession{
			SessionID:   s.ID,
			Title:       s.NowPlayingItem.Name,
			UserName:    s.UserName,
			Player:      player,
			Transcoding: s.Transcoding,
			ProgressMs:  jellyfinTicksToMs(safePositionTicks(s.PlayState)),
			DurationMs:  jellyfinTicksToMs(s.NowPlayingItem.RunTimeTicks),
		})
	}

	return state, nil
}

// jellyfinAddedAt returns the upstream library's addedAt
// timestamp for an item. Jellyfin uses DateCreated as the
// canonical "added to library" field; PremiereDate is the
// release date. We prefer DateCreated when set, falling back to
// the parse of PremiereDate. Returns time.Time{} (zero value)
// when neither is available so the UPSERT still writes a row.
func jellyfinAddedAt(m jellyfinItem) time.Time {
	if m.DateCreated != "" {
		if t, err := time.Parse(time.RFC3339, m.DateCreated); err == nil {
			return t.UTC()
		}
	}
	if m.PremiereDate != "" {
		if t, err := time.Parse(time.RFC3339, m.PremiereDate); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// jellyfinKindNormalize maps Jellyfin's Type field to the
// vocabulary used by the widget. Jellyfin returns 'Movie',
// 'Series', 'Episode', 'Audio', 'MusicAlbum', etc.
func jellyfinKindNormalize(t string) string {
	switch strings.ToLower(t) {
	case "movie":
		return "movie"
	case "series":
		return "show"
	case "episode":
		return "episode"
	default:
		// Audio / photo items don't render in the carousel.
		return "movie"
	}
}

// jellyfinYearPtr returns a *int (nullable) only when the year
// is positive.
func jellyfinYearPtr(y int) *int {
	if y <= 0 {
		return nil
	}
	return &y
}

// jellyfinPosterURL returns the canonical thumbnail URL for a
// Jellyfin item. Jellyfin serves posters at
// /Items/<id>/Images/Primary?maxHeight=300&tag=<ImageTags.Primary>
// — when the item has no Primary image tag, we still return the
// /Items/<id>/Images/Primary path (Jellyfin serves a generic
// image in that case).
func jellyfinPosterURL(base string, m jellyfinItem) string {
	if m.ID == "" {
		return ""
	}
	if m.ImageTags.Primary == "" {
		return fmt.Sprintf("%sItems/%s/Images/Primary?maxHeight=300", base, m.ID)
	}
	return fmt.Sprintf("%sItems/%s/Images/Primary?maxHeight=300&tag=%s", base, m.ID, m.ImageTags.Primary)
}

// safePositionTicks returns the PlayState.PositionTicks when
// PlayState is non-nil, else 0. Guards the optional PlayState
// field so a nil deref can't crash the poller.
func safePositionTicks(ps *struct {
	PositionTicks int64 `json:"PositionTicks"`
}) int64 {
	if ps == nil {
		return 0
	}
	return ps.PositionTicks
}
