// Tier 10 Phase 9 — Task Scheduler (H10). SECURITY-CRITICAL.
//
// Allowlists, constants, and the per-field validation
// primitives for the user-schedulable HTTP-probe cron. Lives
// in its own file so handlers_homelab_scheduler_jobs.go can
// stay under the 400-LOC cap.
//
// Every entry point that accepts user input MUST go through
// one of these validators. The defensive principle: the
// validation is a multi-layered perimeter, not a single
// check. Adding a new action kind or a new header name MUST
// be a deliberate, reviewed change here.
package handler

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ------------------------------------------------------------------
// Allowlists + constants — the security guardrails.
// ------------------------------------------------------------------

// allowedActionKinds is the ONLY set of action kinds the API
// will accept. Adding a new kind here is a deliberate security
// decision — the worker MUST support it (currently it supports
// http_get + http_post only).
var allowedActionKinds = map[string]struct{}{
	"http_get":  {},
	"http_post": {},
}

// blockedHeaderNames is the set of HTTP header names the user
// MUST NOT be able to forge. Comparison is case-insensitive
// (HTTP headers are case-insensitive per RFC 7230). Blocking
// Host prevents header-injection attacks against the target's
// virtual-host routing; Cookie/Authorization prevent the
// scheduler from being used to relay stolen credentials.
//
// Adding a new blocked name here is safe — existing rows that
// use it would have to have been inserted under a prior version
// of this code; in that case the worker/handler will skip them
// (see validateSchedulerHeaders).
var blockedHeaderNames = map[string]struct{}{
	"host":         {},
	"cookie":       {},
	"authorization": {},
	"set-cookie":   {},
	"proxy-authenticate": {},
	"proxy-authorization": {},
	"transfer-encoding": {},
	"content-length": {},
}

// blockedHeaderNamesForLog is the human-friendly list we
// surface to the frontend so the user knows what they can't
// send. Keep it in sync with blockedHeaderNames.
var blockedHeaderNamesForLog = []string{
	"Host", "Cookie", "Authorization", "Set-Cookie",
	"Proxy-Authenticate", "Proxy-Authorization",
	"Transfer-Encoding", "Content-Length",
}

// headerNameRe is the regex a header name must match to be
// accepted. Plain alphanumeric + dash + underscore, 1-64 chars
// (matches RFC 7230 token charset more loosely — we don't need
// to be strict because we're blocking the dangerous names
// separately).
var headerNameRe = regexp.MustCompile(`^[A-Za-z0-9-_]{1,64}$`)

// headerValueMaxLen caps the value of a single user-supplied
// header. 1024 covers any reasonable auth-token, webhook secret,
// or signed-payload header without enabling a single header
// entry to blow memory.
const headerValueMaxLen = 1024

// allowedSchedulerHeaderMax is the maximum number of user
// headers per job. 10 covers realistic webhook payloads (a few
// auth/sig/content-meta headers) without enabling a runaway
// key-value blob.
const allowedSchedulerHeaderMax = 10

// schedulerBodyMaxBytes is the max size of a POST body (the
// http_get path forces body=nil). 4KB matches the upstream
// proxy's body limit and is enough for a JSON webhook payload
// with sane fields.
const schedulerBodyMaxBytes = 4096

// allowedSchedulerBodyContentTypes is the only content-types
// we'll accept on a POST body. We do NOT sniff — the user
// declares the content-type in headers, and the handler rejects
// the body if the declared content-type doesn't match. The
// body itself is passed through verbatim (the target parses it).
var allowedSchedulerBodyContentTypes = map[string]struct{}{
	"text/plain":        {},
	"application/json":  {},
	"application/x-www-form-urlencoded": {},
}

// schedulerNameMaxLen caps the user-supplied job name. 100 chars
// covers any reasonable descriptive label.
const schedulerNameMaxLen = 100

// schedulerURLMaxLen caps the URL so a pathological input can't
// blow the row. 2048 matches the browser's practical limit.
const schedulerURLMaxLen = 2048

// schedulerJobCapPerUser is the maximum number of jobs a single
// user may define. Enforced in CreateHomelabSchedulerJob. Picked
// to be high enough that power users won't hit it (25 schedules
// is a lot of personal cron) but low enough that a malicious
// user can't fill the table with thousands of tiny jobs.
const schedulerJobCapPerUser = 25

// schedulerJobListLimit / schedulerJobListLimitMax are the
// default + ceiling for GET /scheduler/jobs. We don't paginate
// — the per-user cap of 25 is already a hard ceiling so a
// stretchy default is fine.
const schedulerJobListLimit = 100
const schedulerJobListLimitMax = 250

// schedulerRunListLimit / schedulerRunListLimitMax are the
// default + ceiling for GET /scheduler/runs.
const schedulerRunListLimit = 50
const schedulerRunListLimitMax = 200

// schedulerRunRateLimitMinInterval is the minimum interval
// between two runs of the same job. Enforced in
// homelab.RunSchedulerJobAndInsertRun so a freshly-inserted
// job with `* * * * *` can't fire every tick for a single
// user (the worker uses 1-minute ticks so this 5-minute floor
// matches the spec).
const schedulerRunRateLimitMinInterval = 5 * time.Minute

// schedulerExecTimeout caps a single HTTP probe. 30s matches
// the spec and matches the RSS worker ceiling.
const schedulerExecTimeout = 30 * time.Second

// schedulerErrorMessageMaxLen caps the error_message column on
// homelab_scheduler_runs so a verbose TLS/timeout error can't
// blow the row.
const schedulerErrorMessageMaxLen = 500

// ------------------------------------------------------------------
// Validation primitives — exported where the worker package
// also needs them (RunSchedulerJobAndInsertRun).
// ------------------------------------------------------------------

// validateSchedulerActionKind returns nil when raw is in
// allowedActionKinds, otherwise a descriptive error.
func validateSchedulerActionKind(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("action_kind is required")
	}
	if _, ok := allowedActionKinds[raw]; !ok {
		return fmt.Errorf("action_kind must be one of: http_get, http_post")
	}
	return nil
}

// validateSchedulerName rejects empty + too-long names.
func validateSchedulerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > schedulerNameMaxLen {
		return fmt.Errorf("name must be %d characters or fewer", schedulerNameMaxLen)
	}
	return nil
}

// validateSchedulerURL is the strict add-time check. It:
//   1. Parses the URL.
//   2. Requires the scheme to be http or https.
//   3. Requires a non-empty host.
//   4. Caps the length.
//   5. Resolves the host (defense against DNS-based SSRF —
//      a hostname like "localhost.localdomain" or one whose
//      A record is 127.0.0.1 must be blocked).
//   6. Verifies every resolved IP is outside the private/
//      loopback/link-local CIDR set (validateSchedulerHostNotPrivate).
func validateSchedulerURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("url is required")
	}
	if len(raw) > schedulerURLMaxLen {
		return fmt.Errorf("url must be %d characters or fewer", schedulerURLMaxLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url is not parseable: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must be http:// or https://")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("url is missing host")
	}
	return validateSchedulerHostNotPrivate(host)
}

// validateSchedulerHostNotPrivate resolves host (IPv4 literal,
// IPv6 literal, or DNS name) and returns nil only when every
// resolved IP is OUTSIDE the private/loopback/link-local set.
// Used by the handler at add time AND by the worker at run
// time (defense in depth — DNS could resolve differently).
func validateSchedulerHostNotPrivate(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host is empty")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("dns lookup failed for %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("dns lookup returned no addresses for %s", host)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("host %s resolves to a non-public address (%s) — scheduler must target public endpoints", host, ip.String())
		}
	}
	return nil
}

// normalizeSchedulerMethod returns the HTTP method to actually
// use given an action_kind + an optional method override.
// Defaults to GET for http_get, POST for http_post. If the
// caller passes a method it must be the same case (uppercase).
func normalizeSchedulerMethod(actionKind string, method *string) string {
	if method != nil {
		m := strings.ToUpper(strings.TrimSpace(*method))
		if m == "GET" || m == "POST" {
			return m
		}
	}
	if actionKind == "http_post" {
		return "POST"
	}
	return "GET"
}

// validateSchedulerHeaders enforces:
//   - max 10 entries
//   - every name matches headerNameRe
//   - no name in blockedHeaderNames (case-insensitive)
//   - every value ≤ headerValueMaxLen chars
//
// Returns the cleaned map + a descriptive error on first
// failure.
func validateSchedulerHeaders(in map[string]string) (map[string]string, error) {
	if len(in) == 0 {
		return map[string]string{}, nil
	}
	if len(in) > allowedSchedulerHeaderMax {
		return nil, fmt.Errorf("at most %d headers are allowed", allowedSchedulerHeaderMax)
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if !headerNameRe.MatchString(k) {
			return nil, fmt.Errorf("header name %q must match [A-Za-z0-9-_]{1,64}", k)
		}
		if _, blocked := blockedHeaderNames[strings.ToLower(k)]; blocked {
			return nil, fmt.Errorf("header %q is not allowed (forbidden headers: %s)", k, strings.Join(blockedHeaderNamesForLog, ", "))
		}
		if len(v) > headerValueMaxLen {
			return nil, fmt.Errorf("header %q value must be %d characters or fewer", k, headerValueMaxLen)
		}
		out[k] = v
	}
	return out, nil
}

// validateSchedulerBody enforces the per-action body rules:
//   - http_get: body MUST be empty (defense against a future
//     bug that pipes the body to the request anyway)
//   - http_post: body MUST be ≤ schedulerBodyMaxBytes bytes
//
// Content-type sniffing happens separately (the handler checks
// the matching content-type header against
// allowedSchedulerBodyContentTypes). The body itself is
// stored verbatim.
func validateSchedulerBody(actionKind string, body string) error {
	body = strings.TrimSpace(body)
	if actionKind == "http_get" {
		if body != "" {
			return fmt.Errorf("http_get jobs must not include a body")
		}
		return nil
	}
	// http_post
	if len(body) > schedulerBodyMaxBytes {
		return fmt.Errorf("body must be %d bytes or fewer", schedulerBodyMaxBytes)
	}
	return nil
}


// validateSchedulerSchedule is the handler-side wrapper. The
// actual parser lives in handlers_homelab_scheduler_cron.go
// (kept here so the dependency direction stays one-way: jobs
// → types, and types → cron via ValidateCron).
func validateSchedulerSchedule(schedule string) error {
	schedule = strings.TrimSpace(schedule)
	if schedule == "" {
		return fmt.Errorf("schedule is required")
	}
	return ValidateCron(schedule)
}

// RedactSchedulerURL strips any query-string values that look
// like credentials (api_key, token, secret, password, key,
// sig, signature). Used by the worker + handler when logging
// or surfacing an error that mentions the URL.
//
// Strategy: if the URL has no query string, return as-is. If
// it has one, replace each sensitive parameter's value with
// "REDACTED" while preserving the key.
func RedactSchedulerURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return raw
	}
	sensitive := map[string]struct{}{
		"api_key": {}, "apikey": {}, "token": {}, "secret": {},
		"password": {}, "key": {}, "sig": {}, "signature": {},
	}
	q := u.Query()
	changed := false
	for k, vs := range q {
		if _, ok := sensitive[strings.ToLower(k)]; !ok {
			continue
		}
		for i := range vs {
			vs[i] = "REDACTED"
			changed = true
		}
		q[k] = vs
	}
	if !changed {
		return raw
	}
	u.RawQuery = q.Encode()
	return u.String()
}
