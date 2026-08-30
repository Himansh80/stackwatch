package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging logs every request with structured fields.
// Track request in metrics before calling the handler.
func Logging(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)
		reqID, _ := c.Get("request_id")
		TraceID := TraceFromContext(c)
		// Update metrics endpoint counters
		TrackRequest(c.Request.URL.Path, c.Writer.Status())
		logger.Info("request",
			"request_id", reqID,
			"trace_id", TraceID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"size", c.Writer.Size(),
			"duration_ms", dur.Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
