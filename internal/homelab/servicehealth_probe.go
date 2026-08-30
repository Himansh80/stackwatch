// Probe helpers for the ServiceHealthWorker — split out of
// servicehealth.go to keep that file under the 400-LOC cap.
//
// These MUST stay in lock-step with the matching helpers in
// internal/handler/handlers_homelab_services_probe_helpers.go —
// the handler package can't be imported from internal/homelab (Go's
// internal-package rule). See servicehealth.go for the sync contract.
package homelab

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

// probeResult mirrors handler.serviceProbeResult but lives in this
// package so the worker doesn't depend on internal/handler.
type probeResult struct {
	Status       string // "up" | "degraded" | "down" | "unknown"
	LatencyMS    *int
	StatusCode   *int
	ErrorMessage string
}

// probeHTTP sends a GET to rawURL with a 5s timeout and maps the
// status code to a probeResult.
func probeHTTP(ctx context.Context, rawURL string, w *ServiceHealthWorker) probeResult {
	parsed, perr := url.Parse(rawURL)
	if perr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return probeResult{Status: "unknown", ErrorMessage: "url is not http/https"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return probeResult{Status: "unknown", ErrorMessage: err.Error()}
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab-Worker/1.0")
	req.Header.Set("Accept", "*/*")
	client := &http.Client{
		Timeout: w.httpTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		return probeResult{Status: "down", LatencyMS: &latency, ErrorMessage: truncateErr(err.Error())}
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	status := "up"
	if code >= 500 {
		status = "down"
	} else if code >= 400 {
		status = "degraded"
	}
	return probeResult{Status: status, LatencyMS: &latency, StatusCode: &code}
}

// probeTCP opens a TCP connection to addr (host:port) with a 3s
// timeout. Connection success → 'up'.
func probeTCP(ctx context.Context, addr string, w *ServiceHealthWorker) probeResult {
	dialer := net.Dialer{Timeout: w.tcpTimeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		return probeResult{Status: "down", LatencyMS: &latency, ErrorMessage: truncateErr(err.Error())}
	}
	_ = conn.Close()
	return probeResult{Status: "up", LatencyMS: &latency}
}

// probeICMP sends a single ICMP echo to addr with a 2s timeout.
// Echo reply → 'up'. Missing CAP_NET_RAW → 'unknown' (not 'down').
func probeICMP(ctx context.Context, addr string, w *ServiceHealthWorker) probeResult {
	start := time.Now()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
	if err != nil || len(ips) == 0 {
		msg := "dns resolution failed"
		if err != nil {
			msg = err.Error()
		}
		latency := int(time.Since(start) / time.Millisecond)
		return probeResult{Status: "down", LatencyMS: &latency, ErrorMessage: truncateErr(msg)}
	}
	network := "ip4:icmp"
	if ips[0].IP.To4() == nil {
		network = "ip6:icmp"
	}
	conn, err := icmp.ListenPacket(network, "0.0.0.0")
	if err != nil {
		latency := int(time.Since(start) / time.Millisecond)
		return probeResult{Status: "unknown", LatencyMS: &latency,
			ErrorMessage: "icmp socket unavailable: " + err.Error()}
	}
	defer conn.Close()
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: 0x5357, Seq: 1,
			Data: []byte("stackwatch-homelab"),
		},
	}
	bin, err := msg.Marshal(nil)
	if err != nil {
		return probeResult{Status: "unknown", ErrorMessage: err.Error()}
	}
	if err := conn.SetDeadline(time.Now().Add(w.icmpTimeout)); err != nil {
		return probeResult{Status: "unknown", ErrorMessage: err.Error()}
	}
	if _, err := conn.WriteTo(bin, &net.IPAddr{IP: ips[0].IP}); err != nil {
		latency := int(time.Since(start) / time.Millisecond)
		return probeResult{Status: "down", LatencyMS: &latency, ErrorMessage: truncateErr(err.Error())}
	}
	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		return probeResult{Status: "down", LatencyMS: &latency,
			ErrorMessage: "no icmp echo reply: " + truncateErr(err.Error())}
	}
	parsed, err := icmp.ParseMessage(0, reply[:n])
	if err != nil {
		return probeResult{Status: "unknown", LatencyMS: &latency, ErrorMessage: err.Error()}
	}
	if parsed.Type != ipv4.ICMPTypeEchoReply {
		return probeResult{Status: "down", LatencyMS: &latency,
			ErrorMessage: fmt.Sprintf("unexpected icmp type %d", parsed.Type)}
	}
	return probeResult{Status: "up", LatencyMS: &latency}
}

// probeOne dispatches to the right helper by kind. Unknown kinds
// return 'unknown' so a malformed row never crashes the worker.
func probeOne(ctx context.Context, kind, urlStr string, w *ServiceHealthWorker) probeResult {
	switch kind {
	case "http", "https":
		return probeHTTP(ctx, urlStr, w)
	case "tcp":
		return probeTCP(ctx, urlStr, w)
	case "icmp":
		return probeICMP(ctx, urlStr, w)
	default:
		return probeResult{Status: "unknown", ErrorMessage: "unsupported kind: " + kind}
	}
}

// truncateErr caps an error message at 200 chars + ellipsis so a
// huge TLS chain doesn't bloat the homelab_service_health column.
func truncateErr(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
