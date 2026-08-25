// Parse helpers for the CalendarWorker — split out of calendar.go
// to keep that file under the 400-LOC cap.
//
// These MUST stay in lock-step with the matching fetch+parse
// path in calendar.go (fetchAndParse) — the helper file lives in
// the same package so all unexported types are visible. See
// calendar.go for the sync contract.
package homelab

import (
	"time"

	ics "github.com/arran4/golang-ical"
)

// parsedEvent is the worker's internal representation of one
// VEVENT. Kept separate from the handler's eventRow so a future
// schema change in homelab_events doesn't ripple into the parser.
type parsedEvent struct {
	UID         string
	Summary     string
	Description string
	Location    string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
}

// readStart reads DTSTART handling both datetime and all-day forms.
// The arran4 parser exposes two methods — GetStartAt for datetime,
// GetAllDayStartAt for date-only. We try datetime first (more
// common), then fall back. Returns ok=false when neither form is
// present (real-world feeds have been observed missing DTSTART
// entirely; we skip those rather than fail the whole sync).
func readStart(ve *ics.VEvent) (time.Time, bool, bool) {
	t, err := ve.GetStartAt()
	if err == nil && !t.IsZero() {
		return t, false, true
	}
	t, err = ve.GetAllDayStartAt()
	if err == nil && !t.IsZero() {
		return t, true, true
	}
	return time.Time{}, false, false
}

// propertyValue returns the .Value of the given property on a
// component, or "" if the property is absent. Saves a nil-check
// at every call site against the generic arran4 GetProperty API.
func propertyValue(cb *ics.VEvent, prop ics.ComponentProperty) string {
	p := cb.GetProperty(prop)
	if p == nil {
		return ""
	}
	return p.Value
}
