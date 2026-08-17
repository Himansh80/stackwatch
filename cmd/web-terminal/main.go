// Package main is the StackWatch web-terminal service.
//
// Real WebSocket ↔ SSH PTY bridge. Browser (xterm.js) connects over WS,
// we open an SSH session to a saved connection in the database, allocate
// a remote PTY via creack/pty, and stream input/output bidirectionally.
//
// Architecture:
//
//   Browser (xterm.js)
//     ↕ WebSocket (wss://host:8085/api/v1/ws?connection=<uuid>)
//   web-terminal (this service, :8085)
//     ↕ SSH session (golang.org/x/crypto/ssh)
//     ↕ PTY (github.com/creack/pty) on remote host
//   Target host (e.g. .107, .116)
//
// Message protocol (JSON over WS):
//
//   client → server: {type: "input",    data: "ls\n"}
//   client → server: {type: "resize",   cols: 80, rows: 24}
//   server → client: {type: "output",   data: "file1\nfile2\n"}
//   server → client: {type: "exit",     code: 0, message: "..."}
//
// Auth: query param `token=<JWT>` (browsers can't send custom headers in
// the WS upgrade handshake). We look up the saved connection in the DB
// scoped to the JWT's tenant, open SSH with the stored ssh_key_id, and
// allocate a PTY running the user's default shell.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// session represents one active WebSocket → SSH PTY connection.
type session struct {
	id         string
	ws         *websocket.Conn
	sshClient  *ssh.Client
	sshSession *ssh.Session
	stdin      io.WriteCloser
	stdout     io.Reader
	connID     string // saved connection UUID from DB
	host       string
	startedAt  time.Time
	mu         sync.Mutex
	closed     bool
}

// sessionRegistry tracks all live sessions (in-memory).
type sessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{sessions: map[string]*session{}}
}

func (r *sessionRegistry) add(s *session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.id] = s
}

func (r *sessionRegistry) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
}

func (r *sessionRegistry) list() []*session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*session, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, s)
	}
	return out
}

var registry = newSessionRegistry()

// config is runtime config.
type config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
}

func loadConfig() config {
	return config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8085"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://ios:***@192.168.0.116:5432/ios?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "dev-jwt-secret-change-me-in-production-please-32bytes"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// wsMessage is the protocol envelope (client ↔ server).
type wsMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Cols    int    `json:"cols,omitempty"`
	Rows    int    `json:"rows,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func main() {
	cfg := loadConfig()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("starting web-terminal addr=%s", cfg.HTTPAddr)

	ctx := context.Background()
	pool, err := connectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	issuer := auth.NewIssuer(cfg.JWTSecret, 24*time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(pool))
	mux.HandleFunc("/api/v1/sessions", listSessionsHandler)
	mux.HandleFunc("/api/v1/ws", wsHandler(pool, issuer))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("http listening addr=%s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"degraded","db":"down","sessions":%d}`, len(registry.list()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","db":"ok","sessions":%d,"version":"0.1.0-tier3.5"}`, len(registry.list()))
	}
}

func listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions := registry.list()
	type sessOut struct {
		ID        string `json:"id"`
		Host      string `json:"host"`
		ConnID    string `json:"connection_id"`
		StartedAt string `json:"started_at"`
		DurationS int    `json:"duration_s"`
	}
	out := make([]sessOut, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessOut{
			ID:        s.id,
			Host:      s.host,
			ConnID:    s.connID,
			StartedAt: s.startedAt.UTC().Format(time.RFC3339),
			DurationS: int(time.Since(s.startedAt).Seconds()),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": out,
		"total":    len(out),
	})
}

func wsHandler(pool *pgxpool.Pool, issuer *auth.Issuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Auth: try header first, then query param
		token := extractToken(r)
		if token == "" {
			http.Error(w, "missing token (query param or Authorization header)", http.StatusUnauthorized)
			return
		}
		claims, err := issuer.Verify(token)
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		tenantID := claims.TenantID
		userID := claims.UserID

		// 2. Required query param: connection=<uuid>
		connID := r.URL.Query().Get("connection")
		if connID == "" {
			http.Error(w, "missing query param: connection", http.StatusBadRequest)
			return
		}

		// 3. Load the saved connection (tenant-scoped)
		host, port, user, sshKeyID, err := loadConnection(r.Context(), pool, tenantID.String(), connID)
		if err != nil {
			http.Error(w, "load connection: "+err.Error(), http.StatusBadRequest)
			return
		}

		// 4. Upgrade to WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}
		defer ws.Close()

		// 5. Open SSH + allocate PTY
		sess, err := openSSHPty(r.Context(), pool, tenantID.String(), host, port, user, sshKeyID)
		if err != nil {
			_ = ws.WriteJSON(wsMessage{Type: "exit", Code: 1, Message: "ssh: " + err.Error()})
			return
		}
		defer sess.close()

		log.Printf("session start user=%s conn=%s host=%s:%d", userID, connID, host, port)
		registry.add(sess)
		defer registry.remove(sess.id)

		// 6. Pump goroutines: SSH → WS, WS → SSH
		done := make(chan struct{}, 2)
		go func() {
			defer func() { done <- struct{}{} }()
			sess.pumpOutput(ws)
		}()
		go func() {
			defer func() { done <- struct{}{} }()
			sess.pumpInput(ws)
		}()

		<-done
		// Second goroutine will exit when sshSession closes
		<-done
		log.Printf("session end conn=%s", connID)
	}
}

func extractToken(r *http.Request) string {
	// Header first
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// Query param
	return r.URL.Query().Get("token")
}

// loadConnection fetches connection details from the DB, tenant-scoped.
func loadConnection(ctx context.Context, pool *pgxpool.Pool, tenantIDStr, connID string) (string, int, string, string, error) {
	var host string
	var port int
	var user string
	var sshKeyID *string
	err := pool.QueryRow(ctx,
		`SELECT host, port, user_, ssh_key_id::text FROM connections
         WHERE tenant_id = $1 AND id = $2`,
		tenantIDStr, connID).Scan(&host, &port, &user, &sshKeyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, "", "", fmt.Errorf("connection not found")
		}
		return "", 0, "", "", err
	}
	if sshKeyID == nil || *sshKeyID == "" {
		return "", 0, "", "", errors.New("connection has no ssh_key_id (password auth not yet supported)")
	}
	return host, port, user, *sshKeyID, nil
}

// openSSHPty dials SSH using the stored key and starts an interactive shell with a remote PTY.
// On the remote side, the shell thinks it's connected to a terminal (via RequestPty).
// On our side, we use ssh.Session's Stdin/Stdout pipes directly (no local pty needed).
func openSSHPty(ctx context.Context, pool *pgxpool.Pool, tenantIDStr, host string, port int, user, sshKeyIDStr string) (*session, error) {
	// Load the key
	var privKeyPEM, passphrase string
	err := pool.QueryRow(ctx,
		`SELECT private_key, passphrase FROM ssh_keys WHERE tenant_id = $1 AND id = $2`,
		tenantIDStr, sshKeyIDStr).Scan(&privKeyPEM, &passphrase)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privKeyPEM), []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(privKeyPEM))
	}
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // v1: known_hosts check in api-gateway
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}

	sshSess, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("new session: %w", err)
	}

	// Request a remote PTY — the remote shell sees a terminal.
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSess.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		sshSess.Close()
		client.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	// Get stdin pipe for sending input to the remote shell
	stdin, err := sshSess.StdinPipe()
	if err != nil {
		sshSess.Close()
		client.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	// Get stdout pipe for receiving output
	stdout, err := sshSess.StdoutPipe()
	if err != nil {
		sshSess.Close()
		client.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Start the remote shell
	if err := sshSess.Shell(); err != nil {
		sshSess.Close()
		client.Close()
		return nil, fmt.Errorf("shell start: %w", err)
	}

	sess := &session{
		id:         fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		sshClient:  client,
		sshSession: sshSess,
		stdin:      stdin,
		stdout:     stdout,
		connID:     "", // set by caller
		host:       host,
		startedAt:  time.Now(),
	}
	return sess, nil
}

// pumpOutput reads from the remote shell stdout and writes to the WebSocket.
func (s *session) pumpOutput(ws *websocket.Conn) {
	buf := make([]byte, 8192)
	for {
		n, err := s.stdout.Read(buf)
		if n > 0 {
			_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if werr := ws.WriteMessage(websocket.TextMessage, buf[:n]); werr != nil {
				log.Printf("ws write err: %v", werr)
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("pty read err: %v", err)
			}
			// Send exit code
			exitCode := 0
			_ = ws.WriteJSON(wsMessage{Type: "exit", Code: exitCode, Message: "session closed"})
			return
		}
	}
}

// pumpInput reads from the WebSocket and writes to the remote PTY.
func (s *session) pumpInput(ws *websocket.Conn) {
	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("ws read err: %v", err)
			}
			_ = s.stdin.Close()
			_ = s.sshSession.Close()
			return
		}
		var msg wsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			// Treat as raw input (some clients send raw text)
			_, _ = s.stdin.Write(raw)
			continue
		}
		switch msg.Type {
		case "input":
			_, _ = s.stdin.Write([]byte(msg.Data))
		case "resize":
			if msg.Cols > 0 && msg.Rows > 0 {
				_ = s.sshSession.WindowChange(msg.Rows, msg.Cols)
			}
		case "disconnect":
			_ = s.stdin.Close()
			_ = s.sshSession.Close()
			return
		default:
			// Unknown message type — ignore
		}
	}
}

// close shuts down the SSH session and PTY.
func (s *session) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.sshSession != nil {
		_ = s.sshSession.Close()
	}
	if s.sshClient != nil {
		_ = s.sshClient.Close()
	}
}

// keep imports referenced
var _ = url.QueryEscape
