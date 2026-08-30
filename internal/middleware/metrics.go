// metrics.go — exposes /internal/metrics + simple counters for monitoring StackWatch itself.
//
// Endpoints:
//   GET  /internal/metrics        Prometheus text format
//   GET  /internal/metrics/json   JSON summary (human eyeball)
//   GET  /internal/readyz         returns 200 iff the process has started
//   GET  /internal/livez          same (alias for LB probes)

package middleware

import (
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	startedAt     time.Time
	httpRequestTotal atomic.Int64
	httpErrorTotal   atomic.Int64
	requestCountByPath sync.Map // path → *atomic.Int64
)

// init runs before main; registers start time.
func init() {
	startedAt = time.Now()
}

// TrackRequest records one HTTP request for the metrics endpoint.
func TrackRequest(path string, status int) {
	httpRequestTotal.Add(1)
	if status >= 400 {
		httpErrorTotal.Add(1)
	}
	// increment per-path counter (lazy). Key is "route:status".
	key := path + ":" + itoa(status)
	v, _ := requestCountByPath.LoadOrStore(key, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}

func itoa(n int) string {
	// very small int → inline; keeps hot paths fast
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [12]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = digits[n%10]
		n /= 10
	}
	return string(buf[pos:])
}

// MetricsHandler returns a gin.HandlerFunc serving the Prometheus endpoint.
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		// Write basic Prometheus text format manually (no extra deps)
		body := []byte{}
		body = append(body, []byte("# HELP stackwatch_uptime_seconds Seconds since process start\n")...)
		body = append(body, []byte("# TYPE stackwatch_uptime_seconds gauge\n")...)
		body = append(body, []byte("stackwatch_uptime_seconds "+itoa(int(time.Since(startedAt).Seconds()))+"\n")...)

		body = append(body, []byte("# HELP stackwatch_http_requests_total Total HTTP requests served\n")...)
		body = append(body, []byte("# TYPE stackwatch_http_requests_total counter\n")...)
		body = append(body, []byte("stackwatch_http_requests_total "+itoa(int(httpRequestTotal.Load()))+"\n")...)

		body = append(body, []byte("# HELP stackwatch_http_errors_total Total HTTP errors (4xx/5xx)\n")...)
		body = append(body, []byte("# TYPE stackwatch_http_errors_total counter\n")...)
		body = append(body, []byte("stackwatch_http_errors_total "+itoa(int(httpErrorTotal.Load()))+"\n")...)

		n := runtime.NumGoroutine()
		body = append(body, []byte("# HELP stackwatch_goroutines Current goroutine count\n")...)
		body = append(body, []byte("# TYPE stackwatch_goroutines gauge\n")...)
		body = append(body, []byte("stackwatch_goroutines "+itoa(n)+"\n")...)

		body = append(body, []byte("# HELP stackwatch_heap_alloc_bytes Heap allocation (bytes)\n")...)
		body = append(body, []byte("# TYPE stackwatch_heap_alloc_bytes gauge\n")...)
		body = append(body, []byte("stackwatch_heap_alloc_bytes "+itoa64(int64(m.Alloc))+"\n")...)

		cWriter := c.Writer
		cWriter.Write(body)
	}
}

func itoa64(n int64) string {
	if n < 0 {
		return "-" + itoa64(-n)
	}
	if n == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [24]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = digits[n%10]
		n /= 10
	}
	return string(buf[pos:])
}

// MetricsJSON exposes a debug JSON snapshot for /internal/metrics/json.
func MetricsJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c.JSON(http.StatusOK, gin.H{
			"uptime_seconds":    int(time.Since(startedAt).Seconds()),
			"http_requests":     httpRequestTotal.Load(),
			"http_errors":       httpErrorTotal.Load(),
			"goroutines":        runtime.NumGoroutine(),
			"heap_alloc_bytes":  m.Alloc,
			"heap_sys_bytes":    m.HeapSys,
			"gc_pause_ns_total": m.PauseTotalNs,
			"gc_num_gc":         m.NumGC,
			"go_version":        runtime.Version(),
			"os":                runtime.GOOS,
			"arch":              runtime.GOARCH,
			"hostname":          hostname(),
			"pid":               os.Getpid(),
		})
	}
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// Readyz is a trivial readiness probe — always 200 once the process is up.
func Readyz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

// Livez is a trivial liveness probe — always 200 once serving.
func Livez() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	}
}
