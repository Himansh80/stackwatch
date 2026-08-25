// Tier 10 Phase 2 — Service Status (H2). Pure probe helpers
// (HTTP / TCP / ICMP) split out of handlers_homelab_services_probe.go
// to keep both files under 400 LOC.
//
// Used by:
//
//   - handlers_homelab_services_probe.go — the Gin handlers for
//     POST /:id/probe and POST /probe-all
//   - internal/homelab/servicehealth.go  — the 60s background tick
//
// The handler package and the internal/homelab package cannot import
// each other (Go's internal-package rule), so each has its own copy
// of the probe helpers. They MUST stay in sync by convention:
//
//   - same timeouts (probeHTTPTimeout / probeTCPTimeout / probeICMPTimeout)
//   - same status mapping (<400=up, 4xx=degraded, 5xx=down, conn err=down)
//   - same column write (tenant_id, user_id, service_id, status, latency_ms, status_code, error_message)
//
// Probe status mapping:
//
//	http/https  <400  → 'up'        (service responded with success)
//	http/https  4xx   → 'degraded'  (wrong endpoint / auth — your service but misconfigured)
//	http/https  5xx   → 'down'      (upstream failing)
//	http/https  conn refused / dns error / timeout → 'down'
//	tcp         connect ok  → 'up'
//	tcp         connect refused / timeout → 'down'
//	icmp        echo reply → 'up'
//	icmp        no reply (timeout) → 'down'
//
// Failure to start the probe (bad URL, no DNS) → 'unknown' so the
// UI can distinguish "service is broken" from "we couldn't reach
// it to find out".
package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Probe timeouts. HTTP probes get 5s — matches a "real" request
// budget and prevents one slow service from blocking the worker.
// TCP gets 3s (just a connect, no data exchange). ICMP gets 2s
// (echo reply is usually <100ms).
const (
	probeHTTPTimeout = 5 * time.Second
	probeTCPTimeout  = 3 * time.Second
	probeICMPTimeout = 2 * time.Second
)

// IPv4 ICMP type codes (echo / echo reply). Use the typed constants
// from golang.org/x/net/ipv4 (the icmp package's Message struct
// requires Type to satisfy the icmp.Type interface, which only
// ipv4.ICMPType / ipv6.ICMPType do). IPv6 would use
// golang.org/x/net/ipv6 — Phase 2 ships IPv4 only because every
// homelab target on the LAN is IPv4.
var (
	ipv4ICMPTypeEcho      = ipv4.ICMPTypeEcho
	ipv4ICMPTypeEchoReply = ipv4.ICMPTypeEchoReply
)

// probeHTTP sends a GET to rawURL with a 5s timeout and maps the
// status code to a serviceProbeResult.
//
// Notes:
//   - User-Agent is set to "StackWatch-Homelab/1.0" so upstream
//     servers can recognize the probe (some block default UAs).
//   - We don't follow redirects — a 30x to an unexpected host
//     shouldn't be silently reclassified as 'up'.
//   - TLS handshake errors (cert expired, hostname mismatch) are
//     treated as 'down' so the user knows to fix the cert.
func probeHTTP(ctx context.Context, rawURL string) serviceProbeResult {
	parsed, perr := url.Parse(rawURL)
	if perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return serviceProbeResult{Status: "unknown", ErrorMessage: "url is not http/https"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return serviceProbeResult{Status: "unknown", ErrorMessage: err.Error()}
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab/1.0")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{
		Timeout: probeHTTPTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		msg := err.Error()
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return serviceProbeResult{Status: "down", LatencyMS: &latency, ErrorMessage: msg}
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	status := "up"
	if code >= 500 {
		status = "down"
	} else if code >= 400 {
		status = "degraded"
	}
	return serviceProbeResult{
		Status:     status,
		LatencyMS:  &latency,
		StatusCode: &code,
	}
}

// probeTCP opens a TCP connection to addr (host:port) with a 3s
// timeout. Connection success → 'up'. Failure → 'down'.
//
// ICMP is preferred when you want liveness, but TCP is the
// pragmatic alternative when the target doesn't respond to ping
// (most cloud VMs block ICMP by default).
func probeTCP(ctx context.Context, addr string) serviceProbeResult {
	dialer := net.Dialer{Timeout: probeTCPTimeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		msg := err.Error()
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return serviceProbeResult{Status: "down", LatencyMS: &latency, ErrorMessage: msg}
	}
	_ = conn.Close()
	return serviceProbeResult{Status: "up", LatencyMS: &latency}
}

// probeICMP sends a single ICMP echo (ping) to addr with a 2s
// timeout. Echo reply → 'up'. Timeout / error → 'down'.
//
// Uses golang.org/x/net/icmp because the standard library doesn't
// ship ICMP. On Linux the kernel requires CAP_NET_RAW — when the
// binary doesn't have it the ListenPacket call fails; we surface
// that as 'unknown' (not 'down') so the user can tell the probe
// couldn't run vs the host being unreachable.
//
// Production note: ICMP is not always reachable from containers
// or non-root processes. We accept this as a known limitation —
// the user can fall back to 'tcp' or 'http' for the same target.
func probeICMP(ctx context.Context, addr string) serviceProbeResult {
	start := time.Now()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
	if err != nil || len(ips) == 0 {
		msg := "dns resolution failed"
		if err != nil {
			msg = err.Error()
			if len(msg) > 200 {
				msg = msg[:200] + "…"
			}
		}
		latency := int(time.Since(start) / time.Millisecond)
		return serviceProbeResult{Status: "down", LatencyMS: &latency, ErrorMessage: msg}
	}

	network := "ip4:icmp"
	if ips[0].IP.To4() == nil {
		network = "ip6:icmp"
	}
	conn, err := icmp.ListenPacket(network, "0.0.0.0")
	if err != nil {
		// Most likely CAP_NET_RAW missing. Surface as 'unknown'
		// so the user can debug without thinking the host is down.
		latency := int(time.Since(start) / time.Millisecond)
		return serviceProbeResult{Status: "unknown", LatencyMS: &latency,
			ErrorMessage: "icmp socket unavailable: " + err.Error()}
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 0x5357, // arbitrary, must fit in 16 bits
			Seq: 1,
			Data: []byte("stackwatch-homelab"),
		},
	}
	bin, err := msg.Marshal(nil)
	if err != nil {
		return serviceProbeResult{Status: "unknown", ErrorMessage: err.Error()}
	}

	if err := conn.SetDeadline(time.Now().Add(probeICMPTimeout)); err != nil {
		return serviceProbeResult{Status: "unknown", ErrorMessage: err.Error()}
	}
	if _, err := conn.WriteTo(bin, &net.IPAddr{IP: ips[0].IP}); err != nil {
		latency := int(time.Since(start) / time.Millisecond)
		return serviceProbeResult{Status: "down", LatencyMS: &latency, ErrorMessage: err.Error()}
	}

	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		return serviceProbeResult{Status: "down", LatencyMS: &latency,
			ErrorMessage: "no icmp echo reply: " + err.Error()}
	}
	parsed, err := icmp.ParseMessage(0, reply[:n])
	if err != nil {
		return serviceProbeResult{Status: "unknown", LatencyMS: &latency, ErrorMessage: err.Error()}
	}
	if parsed.Type != ipv4ICMPTypeEchoReply {
		return serviceProbeResult{Status: "down", LatencyMS: &latency,
			ErrorMessage: fmt.Sprintf("unexpected icmp type %d", parsed.Type)}
	}
	return serviceProbeResult{Status: "up", LatencyMS: &latency}
}

// probeOne is the dispatcher — picks the right helper based on the
// pin's `kind`. Returns the result so the caller can INSERT it.
//
// Failure to find a helper for an unknown kind returns 'unknown'
// — shouldn't happen because CreateHomelabService / PatchHomelabService
// validate the kind, but we don't trust the DB.
func probeOne(ctx context.Context, kind, urlStr string) serviceProbeResult {
	switch kind {
	case "http", "https":
		return probeHTTP(ctx, urlStr)
	case "tcp":
		return probeTCP(ctx, urlStr)
	case "icmp":
		return probeICMP(ctx, urlStr)
	default:
		return serviceProbeResult{Status: "unknown", ErrorMessage: "unsupported kind: " + kind}
	}
}