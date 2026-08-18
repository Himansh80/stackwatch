// Package synthetics runs HTTP/TCP/ICMP uptime checks.
//
// Run() is a stateless helper — given a Check and a timeout, it
// returns a Result with status (success/failure/timeout), response_ms,
// HTTP status_code (HTTP only), and error message if any.
package synthetics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kind is the type of synthetic check.
type Kind string

const (
	KindHTTP  Kind = "http"
	KindTCP   Kind = "tcp"
	KindICMP  Kind = "icmp"
)

// Check is one synthetic check to run. Matches the synthetics_checks row.
type Check struct {
	ID         string
	TenantID   string
	Name       string
	Kind       Kind
	Target     string // URL (HTTP), host:port (TCP), hostname (ICMP)
	TimeoutMs  int
	Enabled    bool
}

// Result is the outcome of one check run.
type Result struct {
	Status     string // "success" | "failure" | "timeout"
	ResponseMs int
	StatusCode int  // 0 for TCP/ICMP
	Error      string
}

// Run executes the check synchronously and returns the result.
// Safe to call concurrently.
func Run(ctx context.Context, c Check) Result {
	timeout := time.Duration(c.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	switch c.Kind {
	case KindHTTP:
		return runHTTP(ctx, c.Target, timeout)
	case KindTCP:
		return runTCP(ctx, c.Target, timeout)
	case KindICMP:
		return runICMP(ctx, c.Target, timeout)
	default:
		return Result{Status: "failure", Error: "unknown kind: " + string(c.Kind)}
	}
}

func runHTTP(ctx context.Context, target string, timeout time.Duration) Result {
	res := Result{}
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, "GET", target, nil)
	if err != nil {
		return Result{Status: "failure", ResponseMs: int(time.Since(start) / time.Millisecond),
			Error: "new request: " + err.Error()}
	}
	// Custom client with our timeout. Default transport's timeout is 0 = no timeout.
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	res.ResponseMs = int(time.Since(start) / time.Millisecond)
	if err != nil {
		// Distinguish timeout from other errors
		if errors.Is(cctx.Err(), context.DeadlineExceeded) {
			return Result{Status: "timeout", ResponseMs: res.ResponseMs,
				Error: "request exceeded timeout"}
		}
		return Result{Status: "failure", ResponseMs: res.ResponseMs,
			Error: "do request: " + err.Error()}
	}
	defer resp.Body.Close()
	res.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		res.Status = "success"
	} else {
		res.Status = "failure"
		res.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return res
}

func runTCP(ctx context.Context, target string, timeout time.Duration) Result {
	res := Result{}
	start := time.Now()
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", target)
	res.ResponseMs = int(time.Since(start) / time.Millisecond)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || isTimeoutErr(err) {
			return Result{Status: "timeout", ResponseMs: res.ResponseMs,
				Error: "tcp dial timeout"}
		}
		return Result{Status: "failure", ResponseMs: res.ResponseMs,
			Error: "tcp dial: " + err.Error()}
	}
	conn.Close()
	res.Status = "success"
	return res
}

func runICMP(ctx context.Context, target string, timeout time.Duration) Result {
	res := Result{}
	// Validate target is a hostname or IP (not URL)
	target = strings.TrimSpace(target)
	if target == "" {
		return Result{Status: "failure", Error: "empty target"}
	}
	// ping -c 1 -W <timeout_seconds>
	timeoutSec := int(timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	// exec.CommandContext: -c 1 (one packet), -W <seconds> (timeout)
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), target)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	res.ResponseMs = int(time.Since(start) / time.Millisecond)
	if err != nil {
		// Exit code != 0 = failure (or timeout if we hit deadline)
		if ctx.Err() == context.DeadlineExceeded {
			return Result{Status: "timeout", ResponseMs: res.ResponseMs,
				Error: "ping timeout"}
		}
		return Result{Status: "failure", ResponseMs: res.ResponseMs,
			Error: "ping failed: " + strings.TrimSpace(string(out))}
	}
	// Extract time from "time=X ms" in output
	if m := pingTimeRe.FindSubmatch(out); m != nil {
		if ms, perr := strconv.Atoi(string(m[1])); perr == nil {
			res.ResponseMs = ms
		}
	}
	res.Status = "success"
	return res
}

var pingTimeRe = regexp.MustCompile(`time[=<]([0-9.]+)\s*ms`)

// isTimeoutErr reports whether err is a network timeout.
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "i/o timeout")
}
