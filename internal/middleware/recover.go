// Package middleware holds gin middleware. One file per middleware.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/kernel"
)

// Recover converts panics into 500 responses without killing the server.
func Recover(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID, _ := c.Get("request_id")
				logger.Error("panic recovered",
					"request_id", reqID,
					"panic", rec,
					"stack", string(debug.Stack()),
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, kernel.ErrorResponse{
						Error: "internal error",
						Code:  "panic",
					})
				}
			}
		}()
		c.Next()
	}
}

// RequestID assigns a UUID to every request. Used by logging + Recover.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}
