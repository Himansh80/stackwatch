// Package homelab — Task Scheduler (H10) cron helpers.
//
// NextCronRun is the worker-side mirror of
// handler.ParseCron + handler.Next. We duplicate the parser
// here (rather than importing handler) to avoid a circular
// dependency: homelab.RunSchedulerJobAndInsertRun is called
// BY handlers_homelab_scheduler_jobs.go (via POST /:id/run),
// and homelab can't import handler without a cycle (handler
// imports homelab for the worker construction in main.go).
//
// The two parsers are functionally equivalent — same 5-field
// subset, same bounds. If you add a feature to one, mirror
// it in the other (or refactor to a shared package).
package homelab

import (
	"fmt"
	"strings"
	"time"
)

// NextCronRun parses a 5-field cron expression and returns
// the next time strictly AFTER `from` that matches. Returns
// the zero time + a descriptive error on parse failure.
//
// Same semantics as handler.ParseCron → handler.cronExpr.Next
// — kept as a separate implementation to avoid a circular
// import. The shared subset is what the UI hints at in
// SchedulerWidget.tsx.
func NextCronRun(schedule string, from time.Time) (time.Time, error) {
	expr, err := parseHomelabCron(schedule)
	if err != nil {
		return time.Time{}, err
	}
	return nextHomelabCron(expr, from)
}

// homelabCronField is the parsed representation of one field.
type homelabCronField struct {
	allowed map[int]struct{}
	star    bool
	lo, hi  int
}

// homelabCronExpr is the parsed 5-field cron schedule.
type homelabCronExpr struct {
	minute, hour, dom, month, dow homelabCronField
}

// homelabCronBounds mirrors the field bounds documented in
// the handler-package cron.go.
var homelabCronBounds = []struct{ lo, hi int }{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 31}, // day-of-month
	{1, 12}, // month
	{0, 7},  // day-of-week
}

func parseHomelabCron(raw string) (*homelabCronExpr, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("schedule is empty")
	}
	fields := strings.Fields(raw)
	if len(fields) != 5 {
		return nil, fmt.Errorf("schedule must have 5 fields (minute hour day-of-month month day-of-week); got %d", len(fields))
	}
	out := &homelabCronExpr{}
	parsed := []*homelabCronField{&out.minute, &out.hour, &out.dom, &out.month, &out.dow}
	for i, f := range fields {
		bounds := homelabCronBounds[i]
		cf, err := parseHomelabCronField(f, bounds.lo, bounds.hi)
		if err != nil {
			return nil, fmt.Errorf("field %d (%q): %w", i+1, f, err)
		}
		*parsed[i] = cf
	}
	if _, ok := out.dow.allowed[7]; ok {
		delete(out.dow.allowed, 7)
		if _, has0 := out.dow.allowed[0]; !has0 {
			out.dow.allowed[0] = struct{}{}
		}
	}
	return out, nil
}

func parseHomelabCronField(raw string, lo, hi int) (homelabCronField, error) {
	cf := homelabCronField{
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
		rLo, rHi, isStar, err := parseHomelabCronBase(base, lo, hi)
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
	return cf, nil
}

func parseHomelabCronBase(raw string, lo, hi int) (int, int, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "*" {
		return lo, hi, true, nil
	}
	if idx := strings.Index(raw, "-"); idx >= 0 {
		lStr := raw[:idx]
		hStr := raw[idx+1:]
		lv, lerr := parseHomelabCronInt(lStr, lo, hi)
		if lerr != nil {
			return 0, 0, false, lerr
		}
		hv, herr := parseHomelabCronInt(hStr, lo, hi)
		if herr != nil {
			return 0, 0, false, herr
		}
		if lv > hv {
			return 0, 0, false, fmt.Errorf("range %d-%d is descending", lv, hv)
		}
		return lv, hv, false, nil
	}
	v, err := parseHomelabCronInt(raw, lo, hi)
	if err != nil {
		return 0, 0, false, err
	}
	return v, v, false, nil
}

func parseHomelabCronInt(raw string, lo, hi int) (int, error) {
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

func (f homelabCronField) matchField(v int) bool {
	if f.star {
		return true
	}
	_, ok := f.allowed[v]
	return ok
}

func nextHomelabCron(c *homelabCronExpr, from time.Time) (time.Time, error) {
	t := from.UTC().Truncate(time.Minute).Add(time.Minute)
	deadline := from.UTC().Add(4 * 365 * 24 * time.Hour)
	for {
		if t.After(deadline) {
			return time.Time{}, fmt.Errorf("no matching time within 4 years")
		}
		if !c.month.matchField(int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			continue
		}
		if !c.dom.matchField(t.Day()) || !c.dow.matchField(int(t.Weekday())) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, time.UTC)
			continue
		}
		if !c.hour.matchField(t.Hour()) {
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
