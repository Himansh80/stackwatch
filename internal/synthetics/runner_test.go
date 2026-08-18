package synthetics

import (
	"context"
	"net"
	"testing"
	"time"
)

// testServer is a minimal HTTP server that returns 200 OK.
func testServer(t *testing.T, statusCode int) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	stop := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if statusCode == 0 {
				conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
			} else {
				conn.Write([]byte("HTTP/1.1 " +
					itoa(statusCode) +
					" STATUS\r\nContent-Length: 0\r\n\r\n"))
			}
			conn.Close()
		}
	}()
	return "http://" + addr, func() {
		close(stop)
		ln.Close()
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// TestRunHTTP_200: a server returning 200 → success.
func TestRunHTTP_200(t *testing.T) {
	url, stop := testServer(t, 200)
	defer stop()

	res := Run(context.Background(), Check{
		Kind:      KindHTTP,
		Target:    url,
		TimeoutMs: 2000,
	})
	if res.Status != "success" {
		t.Errorf("status = %q, want success (error: %s)", res.Status, res.Error)
	}
	if res.StatusCode != 200 {
		t.Errorf("status_code = %d, want 200", res.StatusCode)
	}
	if res.ResponseMs < 0 || res.ResponseMs > 2000 {
		t.Errorf("response_ms = %d, out of range", res.ResponseMs)
	}
}

// TestRunHTTP_500: server returning 500 → failure.
func TestRunHTTP_500(t *testing.T) {
	url, stop := testServer(t, 500)
	defer stop()

	res := Run(context.Background(), Check{
		Kind:      KindHTTP,
		Target:    url,
		TimeoutMs: 2000,
	})
	if res.Status != "failure" {
		t.Errorf("status = %q, want failure", res.Status)
	}
	if res.StatusCode != 500 {
		t.Errorf("status_code = %d, want 500", res.StatusCode)
	}
}

// TestRunHTTP_Timeout: bad URL → timeout or failure.
func TestRunHTTP_Timeout(t *testing.T) {
	// 192.0.2.0/24 is RFC 5737 TEST-NET — guaranteed unroutable
	res := Run(context.Background(), Check{
		Kind:      KindHTTP,
		Target:    "http://192.0.2.1:81",
		TimeoutMs: 500, // 500ms timeout
	})
	// Could be timeout OR failure (connection refused)
	if res.Status != "timeout" && res.Status != "failure" {
		t.Errorf("status = %q, want timeout or failure", res.Status)
	}
	if res.ResponseMs < 0 || res.ResponseMs > 2000 {
		t.Errorf("response_ms = %d, out of range", res.ResponseMs)
	}
}

// TestRunTCP_Success: connect to a listening TCP port → success.
func TestRunTCP_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	// Accept loop so the connection doesn't hang
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	res := Run(context.Background(), Check{
		Kind:      KindTCP,
		Target:    addr,
		TimeoutMs: 2000,
	})
	if res.Status != "success" {
		t.Errorf("status = %q, want success (error: %s)", res.Status, res.Error)
	}
}

// TestRunTCP_Failure: connect to a closed port → failure.
func TestRunTCP_Failure(t *testing.T) {
	// Pick an unlikely-to-be-open port
	res := Run(context.Background(), Check{
		Kind:      KindTCP,
		Target:    "127.0.0.1:1",  // port 1 is reserved, not listening
		TimeoutMs: 500,
	})
	if res.Status != "failure" && res.Status != "timeout" {
		t.Errorf("status = %q, want failure or timeout (error: %s)", res.Status, res.Error)
	}
}

// TestRunICMP_Localhost: ping localhost → success.
func TestRunICMP_Localhost(t *testing.T) {
	if testing.Short() {
		t.Skip("skip ICMP test in short mode")
	}
	res := Run(context.Background(), Check{
		Kind:      KindICMP,
		Target:    "127.0.0.1",
		TimeoutMs: 2000,
	})
	// ping may not be available or root may not have CAP_NET_RAW
	// We just want it not to panic and to return a sensible result
	if res.Status != "success" && res.Status != "failure" {
		t.Errorf("status = %q, want success or failure", res.Status)
	}
}

// TestRun_UnknownKind: invalid kind → failure with explanatory error.
func TestRun_UnknownKind(t *testing.T) {
	res := Run(context.Background(), Check{
		Kind:      Kind("garbage"),
		Target:    "http://example.com",
		TimeoutMs: 1000,
	})
	if res.Status != "failure" {
		t.Errorf("status = %q, want failure", res.Status)
	}
	if res.Error == "" {
		t.Error("expected error message for unknown kind")
	}
}

// TestRun_ResponseTimeRecorded: response_ms is non-negative after a real call.
func TestRun_ResponseTimeRecorded(t *testing.T) {
	url, stop := testServer(t, 200)
	defer stop()

	res := Run(context.Background(), Check{
		Kind:      KindHTTP,
		Target:    url,
		TimeoutMs: 2000,
	})
	// Local socket call can complete in <1ms which rounds to 0.
	// Just assert it's a sensible value.
	if res.ResponseMs < 0 {
		t.Errorf("response_ms = %d, must be non-negative", res.ResponseMs)
	}
	if res.ResponseMs > 1000 {
		t.Errorf("response_ms = %d, suspiciously large for local test", res.ResponseMs)
	}
	_ = time.Now()
}
