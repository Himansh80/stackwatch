// Tier 10 Phase 9 — Task Scheduler (H10) cron parser.
//
// Minimal 5-field cron parser/validator used by the scheduler
// handlers + the SchedulerWorker to compute next_run_at. Split
// out of handlers_homelab_scheduler_types.go so the types
// file stays under the 400-LOC cap.
//
// We deliberately avoid github.com/robfig/cron as a new
// transitive dep (not in go.sum per Phase 0 review). The cron
// format is small enough that a hand-rolled parser stays
// readable AND lets us validate schedule strings without a
// runtime dep.
//
// Supports the common subset the UI hints at:
//   - "*"            → any value
//   - "5"            → literal value
//   - "1,3,5"        → list of values
//   - "0-30"         → range (inclusive)
//   - "*/15"         → step (every N units)
//   - "1-30/5"       → range with step
//   - "0-23/2"       → every 2 hours, hours 0..23
//
// Field bounds:
//   minute       0-59
//   hour         0-23
//   day-of-month 1-31
//   month        1-12
//   day-of-week  0-7  (0 and 7 = Sunday, both accepted)
//
// All bounds are checked — "60 0 * * *" is rejected, not
// silently rounded.
package handler

import (
	"fmt"
	"strings"
	"time"
)

// cronField is the parsed representation of one cron field.
// We store the explicit values in a small set + a star flag
// because the step/range expansion may include hundreds of
// values for "*/1" — the set is bounded by the field's hi.
type cronField struct {
	allowed map[int]struct{} // explicit values
	star    bool             // true when "*" → any value matches
	lo, hi  int              // valid range (inclusive)
}

// cronExpr is the parsed representation of a 5-field cron
// schedule. ParseCron returns one on success; Next fires it
// forward from a given time.
type cronExpr struct {
	minute, hour, dom, month, dow cronField
}

// allowedCronBounds mirrors the field bounds documented in
// the package doc. Used by parseCronField to reject out-of-
// range values BEFORE constructing the set.
var allowedCronBounds = []struct{ lo, hi int }{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 31}, // day-of-month
	{1, 12}, // month
	{0, 7},  // day-of-week (0 and 7 both = Sunday)
}

// ParseCron parses a 5-field cron expression. Returns the
// parsed expression on success, or a descriptive error.
//
// The schedule string is trimmed and field-whitespace is
// collapsed; "0  0  *  *  0" and "0 0 * * 0" both parse to
// the same thing.
func ParseCron(raw string) (*cronExpr, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("schedule is empty")
	}
	// We do NOT support seconds-precision cron (would require
	// 6 fields). Multiple spaces collapse; tab/CR/LF are
	// treated as separators.
	fields := strings.Fields(raw)
	if len(fields) != 5 {
		return nil, fmt.Errorf("schedule must have 5 fields (minute hour day-of-month month day-of-week); got %d", len(fields))
	}
	out := &cronExpr{}
	parsed := []*cronField{&out.minute, &out.hour, &out.dom, &out.month, &out.dow}
	for i, f := range fields {
		bounds := allowedCronBounds[i]
		cf, err := parseCronField(f, bounds.lo, bounds.hi)
		if err != nil {
			return nil, fmt.Errorf("field %d (%q): %w", i+1, f, err)
		}
		*parsed[i] = cf
	}
	// Day-of-week: collapse 7 → 0 (Sunday) so the user can
	// write either form. The scheduler engine treats them
	// identically.
	if _, ok := out.dow.allowed[7]; ok {
		delete(out.dow.allowed, 7)
		if _, has0 := out.dow.allowed[0]; !has0 {
			out.dow.allowed[0] = struct{}{}
		}
	}
	return out, nil
}

// parseCronField parses a single cron field ("*", "5",
// "1,3,5", "0-30", "*/15", "1-30/5") against bounds
// [lo, hi]. Returns the populated cronField + nil on success.
//
// The math: every comma-separated term is a "range with
// optional step". A bare "*" is a star term with no step
// (any value matches). A bare "5" is a single literal.
func parseCronField(raw string, lo, hi int) (cronField, error) {
	cf := cronField{
		allowed: map[int]struct{}{},
		lo:      lo,
		hi:      hi,
	}
	terms := strings.Split(raw, ",")
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			return cf, fmt.Errorf("empty term")
		}
		// Optional step suffix "/N".
		step := 1
		var base string
		if idx := strings.Index(term, "/"); idx >= 0 {
			base = term[:idx]
			stepStr := term[idx+1:]
			if stepStr == "" {
				return cf, fmt.Errorf("empty step in %q", term)
			}
			n := 0
			for _, ch := range stepStr {
				if ch < '0' || ch > '9' {
					return cf, fmt.Errorf("non-numeric step in %q", term)
				}
				n = n*10 + int(ch-'0')
				if n > 10000 {
					return cf, fmt.Errorf("step too large in %q", term)
				}
			}
			if n < 1 {
				return cf, fmt.Errorf("step must be >= 1 in %q", term)
			}
			step = n
		} else {
			base = term
		}
		// base is "*" or a literal "N" or a range "L-H".
		rLo, rHi, isStar, err := parseCronBase(base, lo, hi)
		if err != nil {
			return cf, err
		}
		if isStar {
			cf.star = true
		}
		for v := rLo; v <= rHi; v += step {
			if v < lo || v > hi {
				return cf, fmt.Errorf("value %d out of bounds [%d, %d]", v, lo, hi)
			}
			cf.allowed[v] = struct{}{}
		}
	}
	// If star was set somewhere AND there are explicit values,
	// that's redundant but not an error — we keep both, the
	// match check is the union of star + explicit.
	return cf, nil
}

// parseCronBase handles the prefix before the optional step:
// "*", "N", "N-M". Returns (rLo, rHi, isStar, error).
func parseCronBase(raw string, lo, hi int) (int, int, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "*" {
		return lo, hi, true, nil
	}
	if idx := strings.Index(raw, "-"); idx >= 0 {
		lStr := raw[:idx]
		hStr := raw[idx+1:]
		lv, lerr := parseCronInt(lStr, lo, hi)
		if lerr != nil {
			return 0, 0, false, lerr
		}
		hv, herr := parseCronInt(hStr, lo, hi)
		if herr != nil {
			return 0, 0, false, herr
		}
		if lv > hv {
			return 0, 0, false, fmt.Errorf("range %d-%d is descending", lv, hv)
		}
		return lv, hv, false, nil
	}
	v, err := parseCronInt(raw, lo, hi)
	if err != nil {
		return 0, 0, false, err
	}
	return v, v, false, nil
}

// parseCronInt parses a non-negative integer in [lo, hi].
func parseCronInt(raw string, lo, hi int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty integer")
	}
	if len(raw) > 6 {
		return 0, fmt.Errorf("integer too long")
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("non-numeric integer %q", raw)
		}
		n = n*10 + int(ch-'0')
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("value %d out of bounds [%d, %d]", n, lo, hi)
	}
	return n, nil
}

// matchField returns true when v is in the field's allowed set
// (or the field is a pure star). Star + explicit is the union
// of both — cron semantics.
func (f cronField) matchField(v int) bool {
	if f.star {
		return true
	}
	_, ok := f.allowed[v]
	return ok
}

// Next returns the next time strictly AFTER `from` that
// matches this cron expression. Always returns a time
// strictly in the future (never == from). Bounded by 4 years
// of search — beyond that, the schedule is pathological and
// we surface an error.
//
// Algorithm: increment by 1 minute, check all 5 fields at each
// step. Worst case is O(4years × 525600) iterations, but the
// implementation skips whole months/days/hours when the field
// doesn't match, so realistic expressions (every 5min, daily,
// weekly) match within seconds.
func (c *cronExpr) Next(from time.Time) (time.Time, error) {
	// Cron operates in the user's local wall-clock semantically
	// but practically every server stores UTC. Use UTC
	// throughout to avoid DST surprises.
	t := from.UTC().Truncate(time.Minute).Add(time.Minute)
	deadline := from.UTC().Add(4 * 365 * 24 * time.Hour)
	for {
		if t.After(deadline) {
			return time.Time{}, fmt.Errorf("no matching time within 4 years")
		}
		if !c.month.matchField(int(t.Month())) {
			// Skip to the first of next month at 00:00.
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			continue
		}
		if !c.dom.matchField(t.Day()) || !c.dow.matchField(int(t.Weekday())) {
			// Skip to the next day at 00:00.
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, time.UTC)
			continue
		}
		if !c.hour.matchField(t.Hour()) {
			// Skip to the next hour at minute 0.
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, time.UTC)
			continue
		}
		if !c.minute.matchField(t.Minute()) {
			t = t.Add(time.Minute)
			continue
		}
		return t, nil
	}
}

// ValidateCron is a thin convenience wrapper used by the
// handler — returns nil if the string parses cleanly.
func ValidateCron(raw string) error {
	_, err := ParseCron(raw)
	return err
}
