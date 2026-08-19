package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// truenasProxy forwards authenticated gateway requests to the local TrueNAS
// sidecar. The sidecar owns upstream JSON-RPC credentials; the gateway owns
// StackWatch JWT authentication and tenant access control.
func truenasProxy() gin.HandlerFunc {
	base := strings.TrimRight(os.Getenv("TRUENAS_CONNECTOR_URL"), "/")
	if base == "" {
		base = "http://127.0.0.1:8088"
	}
	client := &http.Client{Timeout: 90 * time.Second}

	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("path"), "/")
		if path == "" {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "TrueNAS endpoint is required"})
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 8<<20))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "unable to read request body"})
			return
		}
		target := base + "/" + path
		if c.Request.URL.RawQuery != "" {
			target += "?" + c.Request.URL.RawQuery
		}
		req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, target, bytes.NewReader(body))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "unable to create sidecar request"})
			return
		}
		if contentType := c.GetHeader("Content-Type"); contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": "TrueNAS connector unavailable"})
			return
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Status(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}
