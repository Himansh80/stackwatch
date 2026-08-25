// Package homelab — Task Scheduler (H10) runtime URL
// validation helpers (split out of scheduler_run.go to keep
// that file under the 400-LOC cap).
//
// validateSchedulerURLHostNonPrivate is the runtime mirror
// of handler.validateSchedulerURLHostNotPrivate. We can't
// import the handler package here (circular dep), so the
// two implementations are intentionally close but use a
// slightly different surface — the runtime check skips the
// DNS re-lookup because the add-time check already
// validated and re-resolving on every run adds latency
// without meaningful security value.
package homelab

import (
	"errors"
	"fmt"
	"strings"
)

// validateSchedulerURLHostNonPrivate returns nil when host
// is empty or a non-banned literal; an error otherwise.
// Does NOT re-resolve DNS — that's the handler's job at
// insert time.
func validateSchedulerURLHostNonPrivate(host string) error {
	if strings.TrimSpace(host) == "" {
		return errors.New("host is empty")
	}
	if isBannedHostLiteral(host) {
		return fmt.Errorf("host %q is a banned literal", host)
	}
	return nil
}

// isBannedHostLiteral returns true for hostnames that should
// never be scheduled even if DNS were to return a public IP
// (e.g. "localhost" resolves to 127.0.0.1 in /etc/hosts).
func isBannedHostLiteral(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "localhost", "ip6-localhost", "ip6-loopback", "broadcasthost":
		return true
	}
	// ".local" mDNS is also bad — same rationale.
	if strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".localdomain") {
		return true
	}
	return false
}
