// Package handler — Tier 4 Server Admin (Cockpit parity).
//
// Admin endpoints all run SSH commands via the web-terminal sidecar.
// The flow is:
//
//	browser → api-gateway /api/v1/admin/*
//	        → POST http://web-terminal:8085/api/v1/admin/exec
//	        → SSH session → command on target server
//	        → parse stdout (JSON or text) → return structured response
//
// The handlers in this file (admin_services.go, admin_storage.go, etc.)
// all use the shared helpers below: SSHExec (raw command) and SSHExecJSON
// (auto-parse JSON output). This keeps each admin handler small.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminHandler bundles Tier-4 admin endpoints.
//
// webTerminalURL is the URL of the web-terminal sidecar (e.g.
// http://127.0.0.1:8085). The api-gateway POSTs to
// <webTerminalURL>/api/v1/admin/exec and forwards the JWT in the header.
type AdminHandler struct {
	webTerminalURL string
	httpClient     *http.Client
}

// NewAdminHandler constructs the handler.
func NewAdminHandler(webTerminalURL string) *AdminHandler {
	return &AdminHandler{
		webTerminalURL: strings.TrimRight(webTerminalURL, "/"),
		httpClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// sshExecRequest is the body we POST to web-terminal /admin/exec.
type sshExecRequest struct {
	ConnectionID string `json:"connection_id"`
	Command      string `json:"command"`
	TimeoutMs    int    `json:"timeout_ms,omitempty"`
}

// sshExecResponse matches what web-terminal returns.
type sshExecResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// SSHExec runs a single command on the target connection via web-terminal.
// Returns the parsed response or an error. The caller is responsible for
// JSON parsing if the command returns JSON output.
//
// The JWT is taken from the gin context (set by RequireAuth middleware).
func (h *AdminHandler) SSHExec(c *gin.Context, connectionID, command string, timeoutMs int) (*sshExecResponse, error) {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	body, err := json.Marshal(sshExecRequest{
		ConnectionID: connectionID,
		Command:      command,
		TimeoutMs:    timeoutMs,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(),
		time.Duration(timeoutMs+10000)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		h.webTerminalURL+"/api/v1/admin/exec", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// Forward JWT — web-terminal re-verifies it
	if auth := c.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Web-terminal always returns JSON if it set the Content-Type correctly.
	// If Content-Type is text/plain (e.g. on an error that wrote before the
	// JSON header), we synthesize a JSON envelope so the caller always
	// sees a parseable response.
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		// Wrap the text body as a stderr so the caller sees it as an error
		out := &sshExecResponse{
			Stdout:     "",
			Stderr:     string(rawBody),
			ExitCode:   -1,
			DurationMs: 0,
		}
		return out, nil
	}

	var out sshExecResponse
	if err := json.Unmarshal(rawBody, &out); err != nil {
		return nil, err
	}

	// If web-terminal returned an error envelope ({"error": ..., "service": "web-terminal"}),
	// surface it as an error so handlers can render the right HTTP status.
	if resp.StatusCode >= 400 {
		// Try to parse as generic error first
		var errEnv struct {
			Error   string `json:"error"`
			Service string `json:"service"`
			Status  int    `json:"status"`
		}
		_ = json.Unmarshal(rawBody, &errEnv)
		if errEnv.Error != "" {
			return &out, &AdminProxyError{
				StatusCode: resp.StatusCode,
				Message:    errEnv.Error,
			}
		}
	}

	// Non-zero exit is not a transport error — caller decides what to do
	return &out, nil
}

// AdminProxyError is returned when web-terminal rejected the request.
// The handlers check for this to surface the underlying error message.
type AdminProxyError struct {
	StatusCode int
	Message    string
}

func (e *AdminProxyError) Error() string {
	return e.Message
}

// SSHExecJSON runs a command and parses its stdout as JSON into `target`.
// Convenience wrapper around SSHExec.
func (h *AdminHandler) SSHExecJSON(c *gin.Context, connectionID, command string, timeoutMs int, target interface{}) error {
	resp, err := h.SSHExec(c, connectionID, command, timeoutMs)
	if err != nil {
		return err
	}
	if resp.ExitCode != 0 {
		return &AdminExecError{
			Command:  command,
			ExitCode: resp.ExitCode,
			Stderr:   resp.Stderr,
		}
	}
	return json.Unmarshal([]byte(resp.Stdout), target)
}

// AdminExecError is returned when a remote command exited non-zero.
type AdminExecError struct {
	Command  string
	ExitCode int
	Stderr   string
}

func (e *AdminExecError) Error() string {
	return e.Command + " exited " + itoa(e.ExitCode) + ": " + e.Stderr
}

// itoa avoids importing strconv just for one call.
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

// requireConnectionID extracts the connection_id query param or returns
// "" + sets 400. Used by every admin handler.
func requireConnectionID(c *gin.Context) (string, bool) {
	cid := c.Query("connection_id")
	if cid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing query param: connection_id",
		})
		return "", false
	}
	return cid, true
}
