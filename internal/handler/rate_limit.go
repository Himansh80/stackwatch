// Package handler — In-process login rate limiter (Tier 0).
//
// Sliding window: 5 attempts per IP/email per 5 minutes, blocks for 60s
// after the threshold is hit. In-memory only — for multi-node deployments
// swap with Redis; for single-node (current stackwatch deploy) this is
// sufficient.
package handler

import (
	"sync"
	"time"

	"github.com/stackwatch/platform/internal/kernel"
)

// loginRateLimiter implements a per-key sliding window.
type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
	blockFor time.Duration
	blocked  map[string]time.Time
}

// newLoginRateLimiter returns a limiter with the given thresholds.
func newLoginRateLimiter(max int, window, blockFor time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
		blockFor: blockFor,
		blocked:  make(map[string]time.Time),
	}
}

// recordAndCheck adds an attempt and returns (allowed, retryAfter).
// If allowed is false, retryAfter is the duration the caller should wait
// before retrying (zero if no specific wait is needed).
//
// IMPORTANT: when a key is currently blocked, the attempt is NOT
// recorded against the sliding window and the block is NOT extended.
// Counting blocked retries against the window would push the
// block-window deadline further out on every retry — that's the
// classic "stuck on 'please wait 60 seconds' forever" bug. Instead,
// blocked keys get the actual remaining-time-until-unblock as
// their retryAfter and no new timestamp is recorded.
func (r *loginRateLimiter) recordAndCheck(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// 1. Is this key currently blocked?
	if until, ok := r.blocked[key]; ok {
		if now.Before(until) {
			// Stay blocked. DO NOT record this attempt against the
			// sliding window and DO NOT extend the block. Just tell
			// the caller how long is left until the block lifts.
			return false, until.Sub(now)
		}
		delete(r.blocked, key)
	}

	// 2. Trim old attempts outside the window.
	if arr, ok := r.attempts[key]; ok {
		cutoff := now.Add(-r.window)
		i := 0
		for ; i < len(arr); i++ {
			if arr[i].After(cutoff) {
				break
			}
		}
		r.attempts[key] = arr[i:]
	}

	// 3. Record this attempt (only if NOT currently blocked — see step 1).
	r.attempts[key] = append(r.attempts[key], now)

	// 4. If we're at or past the max, block.
	if len(r.attempts[key]) >= r.max {
		blockUntil := now.Add(r.blockFor)
		r.blocked[key] = blockUntil
		// Return the actual remaining time until the block lifts, not
		// the full blockFor duration. Lets the caller show a live
		// countdown that actually counts down to zero.
		return false, blockUntil.Sub(now)
	}
	return true, 0
}

// defaultLoginLimiter: 5 attempts / 5 minutes / 60s block.
var defaultLoginLimiter = newLoginRateLimiter(5, 5*time.Minute, 60*time.Second)

// loginKey returns the rate-limit key for a login attempt: "ip|email".
// Hashing IP + email gives a low-cardinality bucket that is hard for an
// attacker to bypass by rotating emails or IPs alone.
func loginKey(ip, email string) string {
	return ip + "|" + email
}

// checkLoginRateLimit records an attempt and returns ErrTooManyRequests
// if the key is rate-limited. Otherwise returns nil.
func checkLoginRateLimit(ip, email string) (retryAfter time.Duration, err error) {
	if ip == "" {
		ip = "unknown"
	}
	if email == "" {
		email = "unknown"
	}
	allowed, wait := defaultLoginLimiter.recordAndCheck(loginKey(ip, email))
	if !allowed {
		return wait, kernel.ErrTooManyRequests
	}
	return 0, nil
}
