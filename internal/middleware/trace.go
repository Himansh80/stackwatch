// Package middleware — trace propagation between http handlers.
//
// TraceID is assigned once per HTTP request (or inherited from the
// "X-Trace-Id" / "traceparent" incoming header) and stored in the gin
// context. All downstream calls (logs, sql, outgoing http to sidecars)
// can read it and propagate further.
package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDHeader = "X-Trace-Id"
const TraceParentHeader = "traceparent" // W3C Trace Context
const TraceIDKey = "trace_id"

// TraceID middleware: sets a stable trace_id on the request context.
//
// Read from X-Trace-Id or traceparent (fallback to a fresh UUIDv4).
// Then stamped on the response header so client can grep it.
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := extractTraceID(c)
		c.Set(TraceIDKey, traceID)
		c.Writer.Header().Set(TraceIDHeader, traceID)
		c.Next()
	}
}

// TraceFromContext returns the trace_id or empty string.
func TraceFromContext(c *gin.Context) string {
	v, ok := c.Get(TraceIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractTraceID picks the most appropriate trace id:
//   1. X-Trace-Id header if present and well-formed
//   2. traceparent (W3C "00-<trace-id>-<parent-id>-<flags>") — uses the
//      middle field
//   3. Generated UUIDv4
func extractTraceID(c *gin.Context) string {
	if h := c.GetHeader(TraceIDHeader); h != "" && isValidTraceID(h) {
		return h
	}
	if tp := c.GetHeader(TraceParentHeader); tp != "" {
		// Format: 00-<32-hex>-<16-hex>-<2-hex>
		parts := splitN(tp, "-", 4)
		if len(parts) >= 2 && len(parts[1]) == 32 {
			return parts[1] // already a 32-char hex string
		}
	}
	return uuid.NewString() // 36 chars, includes hyphens; still unique enough
}

func isValidTraceID(s string) bool {
	// accept either a uuid (with hyphens) or a 32-char hex string
	if len(s) == 32 {
		for _, ch := range s {
			if !isHex(ch) {
				return false
			}
		}
		return true
	}
	if _, err := uuid.Parse(s); err == nil {
		return true
	}
	return false
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func splitN(s, sep string, n int) []string {
	return splitStringN(s, sep, n)
}

func splitStringN(s, sep string, n int) []string {
	parts := []string{}
	cur := ""
	for _, ch := range s {
		if string(ch) == sep && len(parts) < n-1 {
			parts = append(parts, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	parts = append(parts, cur)
	return parts
}

// AddTraceToRequest logs a request with both request_id + trace_id. Used by
// middleware.Logging internally; not exported.
// RequestLogEntry has the shape used in each request log line.
type RequestLogEntry struct {
	RequestID string `json:"request_id"`
	TraceID   string `json:"trace_id"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Duration  int64  `json:"duration_ms"`
	Client    string `json:"client_ip"`
}

// LogRequestEntry writes a single combined log entry with request_id + trace_id.
func LogRequestEntry(logger *slog.Logger, entry RequestLogEntry) {
	logger.Info("request",
		"request_id", entry.RequestID,
		"trace_id", entry.TraceID,
		"method", entry.Method,
		"path", entry.Path,
		"status", entry.Status,
		"duration_ms", entry.Duration,
		"client_ip", entry.Client,
	)
}

// HTTPErrorResponder converts the current request's trace+request ids into
// an RFC 9457 problem detail payload when a handler errors out.
func WriteProblemJSON(c *gin.Context, status int, title, detail string) {
	c.JSON(status, gin.H{
		"type":   "about:blank",
		"title":  title,
		"status": status,
		"detail": detail,
		// trace context is only set if middleware ran
		"trace_id":   TraceFromContext(c),
		"request_id": c.GetString("request_id"),
	})
}

// IsHTTPSuccess returns true for 2xx.
func IsHTTPSuccess(code int) bool { return code >= 200 && code < 300 }
