// Emby poller for the MediaWorker + the
// handler.PollMediaServerAndInsertState wrapper.
//
// Endpoints polled (same as Jellyfin):
//
//	GET /Library/Media/Recent?limit=10
//	  → recent additions
//	GET /Sessions
//	  → active playback sessions
//
// Auth: Emby uses the X-Emby-Token request header.
//
// Time units: Emby uses 100ns ticks for PositionTicks +
// RunTimeTicks (same as Jellyfin) — we reuse the
// jellyfinTicksToMs helper.
//
// Emby's API is structurally identical to Jellyfin's — both
// projects share the same MediaBrowser codebase lineage. We use
// dedicated Emby types (rather than aliasing the Jellyfin types)
// so future Emby-specific quirks (e.g. Emby's stricter
// PremiereDate parsing or proprietary SortName field) can be
// added without breaking Jellyfin.
//
// Split out from media_clients.go so this file stays under the
// 400-LOC cap.
package homelab

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// embyItemsResponse mirrors jellyfinItemsResponse but is named
// separately so a future Emby-specific field (e.g. Emby's
// proprietary SortName) can be added without aliasing.
type embyItemsResponse struct {
	Items []embyItem `json:"Items"`
}

type embyItem struct {
	ID             string `json:"Id"`
	Name           string `json:"Name"`
	Type           string `json:"Type"`
	ProductionYear int    `json:"ProductionYear"`
	DateCreated    string `json:"DateCreated"`
	PremiereDate   string `json:"PremiereDate"`
	ImageTags      struct {
		Primary string `json:"Primary"`
	} `json:"ImageTags"`
}

type embySessionsResponse []embySession

type embySession struct {
	ID        string `json:"Id"`
	PlayState *struct {
		PositionTicks int64 `json:"PositionTicks"`
	} `json:"PlayState"`
	NowPlayingItem *struct {
		Name         string `json:"Name"`
		RunTimeTicks int64  `json:"RunTimeTicks"`
		Type         string `json:"Type"`
	} `json:"NowPlayingItem"`
	Client      string `json:"Client"`
	DeviceName  string `json:"DeviceName"`
	UserName    string `json:"UserName"`
	Transcoding bool   `json:"IsTranscoding"`
}

// pollEmby fetches the recent additions + active sessions from
// an Emby server. The endpoint shape matches Jellyfin (Emby's
// API is a superset of Jellyfin's) but we use dedicated types so
// a future Emby-specific quirk can be added without breaking
// Jellyfin.
func pollEmby(ctx context.Context, baseURL, apiKey string) (*MediaState, error) {
	joined, err := joinMediaBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeMediaClient()

	state := &MediaState{}

	// Recent additions.
	itemsReq, ierr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sLibrary/Media/Recent?limit=10&Fields=PrimaryImageAspectRatio,DateCreated,ProductionYear", joined), nil)
	if ierr != nil {
		return nil, fmt.Errorf("new Recent request: %w", ierr)
	}
	itemsReq.Header.Set("X-Emby-Token", apiKey)
	var itemsResp embyItemsResponse
	if _, ierr := readMediaJSONBody(ctx, c, itemsReq, &itemsResp); ierr == nil {
		for _, m := range itemsResp.Items {
			if m.ID == "" || m.Name == "" {
				continue
			}
			addedAt := embyAddedAt(m)
			state.RecentAdditions = append(state.RecentAdditions, MediaItem{
				ItemID:    m.ID,
				Title:     m.Name,
				Kind:      jellyfinKindNormalize(m.Type), // shared vocabulary
				Year:      jellyfinYearPtr(m.ProductionYear),
				PosterURL: embyPosterURL(joined, m),
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
	var sessResp embySessionsResponse
	if _, serr := readMediaJSONBody(ctx, c, sessReq, &sessResp); serr != nil {
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
			ProgressMs:  jellyfinTicksToMs(safePositionTicksEmby(s.PlayState)),
			DurationMs:  jellyfinTicksToMs(s.NowPlayingItem.RunTimeTicks),
		})
	}

	return state, nil
}

// embyAddedAt mirrors jellyfinAddedAt — Emby's DateCreated is
// the canonical addedAt.
func embyAddedAt(m embyItem) time.Time {
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

// embyPosterURL mirrors jellyfinPosterURL — same Emby/Jellyfin
// /Items/<id>/Images/Primary path.
func embyPosterURL(base string, m embyItem) string {
	if m.ID == "" {
		return ""
	}
	if m.ImageTags.Primary == "" {
		return fmt.Sprintf("%sItems/%s/Images/Primary?maxHeight=300", base, m.ID)
	}
	return fmt.Sprintf("%sItems/%s/Images/Primary?maxHeight=300&tag=%s", base, m.ID, m.ImageTags.Primary)
}

// safePositionTicksEmby is the Emby-side mirror of
// safePositionTicks (Go doesn't allow struct-pointer type aliases
// across named types — same shape, different named type).
func safePositionTicksEmby(ps *struct {
	PositionTicks int64 `json:"PositionTicks"`
}) int64 {
	if ps == nil {
		return 0
	}
	return ps.PositionTicks
}
