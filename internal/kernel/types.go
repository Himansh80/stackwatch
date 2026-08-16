// Package kernel holds shared types used across all services.
// No business logic here. No imports from other internal packages.
package kernel

import (
	"errors"
	"time"
)

// Common errors used across all services.
var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrBadRequest      = errors.New("bad request")
	ErrInternal        = errors.New("internal error")
	ErrConflict        = errors.New("conflict")
	ErrTooManyRequests = errors.New("too many requests")
)

// Page is a generic pagination request.
type Page struct {
	Limit  int `json:"limit" validate:"min=1,max=500"`
	Offset int `json:"offset" validate:"min=0"`
}

// NewPage returns a Page with sensible defaults.
func NewPage() Page {
	return Page{Limit: 50, Offset: 0}
}

// TimeRange is a generic time range.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// NewTimeRange returns a TimeRange for the last `dur`.
func NewTimeRange(dur time.Duration) TimeRange {
	now := time.Now().UTC()
	return TimeRange{From: now.Add(-dur), To: now}
}
