// Package main is the StackWatch web-terminal service.
//
// It provides a WebSocket bridge from browser xterm.js to a remote SSH
// shell (or local bash). The terminal multiplexes input/output over a
// single WebSocket connection using framed messages.
//
// Architecture:
//
//   Browser xterm.js
//     ↕ WebSocket (wss://host:8085/api/v1/ws)
//   web-terminal (.115:8085)
//     ↕ SSH session (creack/pty)
//   Target host (e.g. .107, .108, .130)
//
// Message protocol (JSON over WS):
//
//   client → server: {type: "input",    data: "ls\n"}
//   client → server: {type: "resize",   cols: 80, rows: 24}
//   client → server: {type: "disconnect"}
//   server → client: {type: "stdout",   data: "file1\nfile2\n"}
//   server → client: {type: "stderr",   data: "warning\n"}
//   server → client: {type: "exit",     code: 0}
//   server → client: {type: "error",    message: "..."}
//
// Auth: query param `token=<JWT>` (because browsers can't send headers
// during WS upgrade) or `Authorization: Bearer <JWT>` header.
//
// Note: this is a minimal v1 — see todo.txt for follow-ups (recording,
// port forwarding, SFTP, etc.).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/stackwatch/platform/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// session represents one active WebSocket → PTY connection.
type session struct {
	ws      *websocket.Conn
	cmd     *exec.Cmd
	ptmx    io.ReadWriteCloser // = pty.Pty (interface)
	mu      sync.Mutex
	closed  bool
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8085"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "tier0-secret-32bytes-long-please-rotate"
	}

	authSvc := auth.NewIssuer(jwtSecret, 24*time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/ws", authMiddleware(authSvc, handleTerminal))
	mux.HandleFunc("/api/v1/exec", authMiddleware(authSvc, handleExec))

	log.Printf("web-terminal: listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("web-terminal: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"web-terminal"}`))
}

func authMiddleware(authSvc *auth.Issuer, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try header first, then query param (for WS)
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := authSvc.Verify(token)
		if err != nil {
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		_ = claims
		next(w, r)
	}
}

func handleTerminal(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer ws.Close()

	// Read the first message — must be a connect request with target info.
	_, first, err := ws.ReadMessage()
	if err != nil {
		ws.WriteJSON(map[string]any{"type": "error", "message": "no initial message"})
		return
	}
	var req struct {
		Type   string `json:"type"`
		Host   string `json:"host"`     // "host:port" — ssh target
		User   string `json:"user"`     // ssh username
		Passwd string `json:"password"` // ssh password (optional)
		Key    string `json:"key"`      // ssh private key (optional)
		Cols   uint16 `json:"cols"`
		Rows   uint16 `json:"rows"`
	}
	if err := json.Unmarshal(first, &req); err != nil {
		ws.WriteJSON(map[string]any{"type": "error", "message": "invalid JSON: " + err.Error()})
		return
	}
	if req.Type != "connect" {
		ws.WriteJSON(map[string]any{"type": "error", "message": "expected type=connect"})
		return
	}

	// Resolve target: empty host = local bash
	var cmd *exec.Cmd
	if req.Host == "" {
		// Local bash
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		cmd = exec.Command(shell, "-i")
	} else {
		// SSH to target host
		host := req.Host
		user := req.User
		if user == "" {
			user = "root"
		}
		// Build ssh command
		sshArgs := []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "LogLevel=ERROR",
		}
		if req.Passwd != "" {
			sshArgs = append(sshArgs, "-o", "PasswordAuthentication=yes",
				"-o", "PubkeyAuthentication=no")
		}
		sshArgs = append(sshArgs, "-tt", fmt.Sprintf("%s@%s", user, host))
		cmd = exec.Command("ssh", sshArgs...)
		// Set password via env (sshpass-style) — note: real prod uses sshpass
		// or proper key auth. For v1, expect sshpass if password is provided.
		if req.Passwd != "" {
			cmd = exec.Command("sshpass", append([]string{"-p", req.Passwd, "ssh"},
				append(sshArgs, "")...)...)
		}
	}

	// Set initial window size
	size := pty.Winsize{Rows: 24, Cols: 80}
	if req.Cols > 0 { size.Cols = req.Cols }
	if req.Rows > 0 { size.Rows = req.Rows }

	ptmx, err := pty.StartWithSize(cmd, &size)
	if err != nil {
		ws.WriteJSON(map[string]any{"type": "error", "message": "pty start: " + err.Error()})
		return
	}
	sess := &session{ws: ws, cmd: cmd, ptmx: ptmx}
	defer sess.close()

	// Welcome message
	ws.WriteJSON(map[string]any{
		"type": "stdout",
		"data": fmt.Sprintf("\x1b[32mConnected to %s\x1b[0m\r\n", req.Host),
	})

	// Goroutine: PTY → WS
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("pty read: %v", err)
				}
				ws.WriteJSON(map[string]any{"type": "exit", "code": -1})
				sess.close()
				return
			}
			ws.WriteJSON(map[string]any{"type": "stdout", "data": string(buf[:n])})
		}
	}()

	// Main: WS → PTY
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var m struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Cols uint16 `json:"cols"`
			Rows uint16 `json:"rows"`
		}
		if err := json.Unmarshal(msg, &m); err != nil {
			continue
		}
		switch m.Type {
		case "input":
			ptmx.Write([]byte(m.Data))
		case "resize":
			if m.Cols > 0 && m.Rows > 0 {
				pty.Setsize(ptmx, &pty.Winsize{Rows: m.Rows, Cols: m.Cols})
			}
		case "disconnect":
			sess.close()
			return
		}
	}
}

func (s *session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	if s.ptmx != nil {
		s.ptmx.Close()
	}
}

// handleExec accepts a one-shot exec request (no WS needed).
// POST /api/v1/exec  body: {"host":"h:p","user":"r","command":"ls -la","password":"..."}
// Returns the output.
func handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Host    string `json:"host"`
		User    string `json:"user"`
		Passwd  string `json:"password"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Command == "" {
		http.Error(w, "command required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	output, err := execSSH(ctx, req.Host, req.User, req.Passwd, req.Command)
	if err != nil {
		http.Error(w, "exec: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"host":    req.Host,
		"command": req.Command,
		"output":  output,
	})
}

func execSSH(ctx context.Context, host, user, passwd, command string) (string, error) {
	if host == "" {
		// Local exec
		out, err := exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
		if err != nil {
			return string(out), err
		}
		return string(out), nil
	}
	if user == "" {
		user = "root"
	}
	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-o", "PasswordAuthentication=yes",
		"-o", "PubkeyAuthentication=no",
		"-o", "ConnectTimeout=10",
		"-p", "22",
	}
	// Extract port from host if present
	h, p, err := splitHostPort(host)
	if err == nil && p != "" {
		sshArgs[len(sshArgs)-1] = p
		host = h
	}
	args := append(sshArgs, fmt.Sprintf("%s@%s", user, host), command)
	cmd := exec.CommandContext(ctx, "sshpass", append([]string{"-p", passwd}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func splitHostPort(hostport string) (host, port string, err error) {
	u, err := url.Parse("ssh://" + hostport)
	if err != nil {
		return "", "", err
	}
	return u.Hostname(), u.Port(), nil
}
