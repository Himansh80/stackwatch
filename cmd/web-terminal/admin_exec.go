// Tier 4 admin SSH exec endpoint.
//
// Adds POST /api/v1/admin/exec to the web-terminal sidecar.
// Body: { connection_id, command, timeout_ms? }
// Returns: { stdout, stderr, exit_code, duration_ms }
//
// This is the SSH bridge for Tier 4 Server Admin (Cockpit parity).
// Handlers in api-gateway call this endpoint to run admin commands
// on a target server and parse the structured output.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/auth"
)

// adminExecRequest is the POST body for /api/v1/admin/exec
type adminExecRequest struct {
	ConnectionID string `json:"connection_id"`
	Command      string `json:"command"`
	TimeoutMs    int    `json:"timeout_ms,omitempty"`
}

// adminExecResponse is the response shape
type adminExecResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
}

// adminExecHandler runs a single non-interactive command over SSH and
// returns the captured output. Uses the same loadConnection flow as the
// WebSocket handler so the JWT tenant scoping is identical.
func adminExecHandler(pool *pgxpool.Pool, issuer *auth.Issuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// 1. Auth: try header first, then query param
		token := extractToken(r)
		if token == "" {
			writeAdminJSONError(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims, err := issuer.Verify(token)
		if err != nil {
			writeAdminJSONError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
			return
		}
		tenantID := claims.TenantID

		// 2. Parse body
		var req adminExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAdminJSONError(w, http.StatusBadRequest, "bad json: "+err.Error())
			return
		}
		if req.ConnectionID == "" || req.Command == "" {
			writeAdminJSONError(w, http.StatusBadRequest, "connection_id and command are required")
			return
		}

		// 3. Resolve connection (tenant-scoped)
		host, port, sshUser, sshKeyID, err := loadConnection(r.Context(), pool, tenantID.String(), req.ConnectionID)
		if err != nil {
			writeAdminJSONError(w, http.StatusBadRequest, "load connection: "+err.Error())
			return
		}

		// 4. Default timeout 30s, capped at 5min
		timeoutMs := req.TimeoutMs
		if timeoutMs <= 0 {
			timeoutMs = 30000
		}
		if timeoutMs > 300000 {
			timeoutMs = 300000
		}

		// 5. Run command
		start := time.Now()
		stdout, stderr, exitCode, runErr := runRemoteCommand(r.Context(), pool, tenantID.String(),
			host, port, sshUser, sshKeyID, req.Command, timeoutMs)
		durationMs := time.Since(start).Milliseconds()

		if runErr != nil {
			// Network / SSH failures surface as a structured response with exit code -1
			resp := adminExecResponse{
				Stdout:     stdout,
				Stderr:     stderr + "\n[stackwatch] ssh error: " + runErr.Error(),
				ExitCode:   -1,
				DurationMs: durationMs,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(adminExecResponse{
			Stdout:     stdout,
			Stderr:     stderr,
			ExitCode:   exitCode,
			DurationMs: durationMs,
		})
	}
}

// runRemoteCommand opens a non-PTY SSH session and runs a single command.
// Returns stdout, stderr, exit_code. Network/SSH errors return an error
// (and the caller surfaces them as HTTP 502).
func runRemoteCommand(ctx context.Context, pool *pgxpool.Pool, tenantIDStr, host string,
	port int, sshUser, sshKeyIDStr, command string, timeoutMs int) (string, string, int, error) {

	// Load SSH key
	var privKeyPEM, passphrase string
	if err := pool.QueryRow(ctx,
		`SELECT private_key, passphrase FROM ssh_keys WHERE tenant_id = $1 AND id = $2`,
		tenantIDStr, sshKeyIDStr).Scan(&privKeyPEM, &passphrase); err != nil {
		return "", "", -1, fmt.Errorf("load key: %w", err)
	}

	// Parse the private key
	var (
		signer ssh.Signer
		perr   error
	)
	if passphrase != "" {
		signer, perr = ssh.ParsePrivateKeyWithPassphrase([]byte(privKeyPEM), []byte(passphrase))
	} else {
		signer, perr = ssh.ParsePrivateKey([]byte(privKeyPEM))
	}
	if perr != nil {
		return "", "", -1, fmt.Errorf("parse key: %w", perr)
	}

	cfg := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // v1: known_hosts check in api-gateway
		Timeout:         15 * time.Second,
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), cfg)
	if err != nil {
		return "", "", -1, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	// Capture stdout/stderr
	var outBuf, errBuf safeBuf
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	// Apply timeout via context
	cancelCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	go func() {
		<-cancelCtx.Done()
		if cancelCtx.Err() == context.DeadlineExceeded {
			sess.Close()
		}
	}()

	runErr := sess.Run(command)
	exitCode := 0
	if runErr != nil {
		// ExitError carries the exit code
		if ee, ok := runErr.(*ssh.ExitError); ok {
			exitCode = ee.ExitStatus()
		} else {
			// Network / context error
			return outBuf.String(), errBuf.String(), -1, runErr
		}
	}
	return outBuf.String(), errBuf.String(), exitCode, nil
}

// safeBuf is a thread-safe wrapper around strings.Builder.
// SSH sessions may write concurrently from internal goroutines.
type safeBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// writeAdminJSONError writes a structured JSON error response (so the
// caller — api-gateway AdminHandler — always sees a parseable JSON body,
// even on validation failures).
func writeAdminJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   msg,
		"status":  status,
		"service": "web-terminal",
	})
}
