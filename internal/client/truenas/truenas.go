// Package truenas is a client for TrueNAS SCALE 25.10+ via JSON-RPC 2.0
// over WebSocket at `/api/current`.
//
// Auth: WebSocket connection to `wss://<host>/api/current`, then call
// `auth.login(app, username, password)`. Subsequent calls reuse the same
// WS — no per-call handshake. Sessions auto-reconnect on protocol errors.
//
// Method shape (positional args):
//
//	c.Query("pool.query", []any{})                              → []Pool
//	c.Query("pool.dataset.query", []any{[][]any{...}})          → filtered
//	c.Call("pool.dataset.create", []any{pool.DatasetCreate{}})   → created obj
package truenas

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// callTimeout bounds any single JSON-RPC call.
const callTimeout = 30 * time.Second

// ErrAuth is returned when login fails or session expires and cannot be renewed.
var ErrAuth = errors.New("truenas: not authenticated")

// ErrRateLimited is returned when TrueNAS middleware rate-limits a request.
var ErrRateLimited = errors.New("truenas: rate limited by middleware")

// ErrCall is returned when the remote method call returned an error object.
type ErrCall struct {
	Errname string
	Reason  string
}

func (e *ErrCall) Error() string {
	if e.Reason != "" {
		return "truenas: " + e.Reason
	}
	return "truenas: " + e.Errname
}

// ErrCall.IsRateLimited reports whether this error is a rate-limit error.
func (e *ErrCall) IsRateLimited() bool {
	if e == nil {
		return false
	}
	if e.Errname == "EBUSY" {
		return true
	}
	return strings.Contains(e.Reason, "Rate Limit") ||
		strings.Contains(e.Reason, "rate limit") ||
		strings.Contains(e.Reason, "EBUSY")
}

// Client is a TrueNAS JSON-RPC over WebSocket client bound to one host.
type Client struct {
	baseURL    string
	username   string
	password   string
	verifyTLS  bool
	httpClient *http.Client

	mu      sync.Mutex
	conn    *websocket.Conn
	idSeq   int
	pendng  map[int]chan rpcResp
	closed  bool

	// rateState tracks recent EBUSY hits so we stop trying to log in
	// during the cool-down. When non-zero, login attempts return
	// ErrRateLimited without contacting the server.
	rateMu   sync.Mutex
	rateLock time.Time // last time we went into rate-limit mode
	cooldown time.Duration
}

// isRateLimited returns true if we should refuse to re-login right now.
// Cooldown is per-host and gets reset on a fresh login success.
func (c *Client) isRateLimited() bool {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	if c.cooldown == 0 {
		return false
	}
	if time.Since(c.rateLock) > c.cooldown {
		return false
	}
	return true
}

func (c *Client) noteRateLimited() {
	c.rateMu.Lock()
	c.rateLock = time.Now()
	if c.cooldown == 0 {
		c.cooldown = 30 * time.Second
	}
	c.rateMu.Unlock()
}

// NewClient returns a client; auth happens lazily on first call.
func NewClient(baseURL, _, username, password string, verifyTLS bool) *Client {
	// API key / bearer auth not used by SCALE 25.10 (it uses session login).
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyTLS},
	}
	return &Client{
		baseURL:    trimSlash(baseURL),
		username:   username,
		password:   password,
		verifyTLS:  verifyTLS,
		httpClient: &http.Client{Transport: tr, Timeout: 0}, // we drive timeout on WS
	}
}

// wsURL converts the base URL to a wss URL ending in /api/current.
// e.g. https://192.168.0.112 → wss://192.168.0.112/api/current
func (c *Client) wsURL() string {
	scheme := "wss"
	if len(c.baseURL) >= 5 && c.baseURL[:5] == "http:" {
		scheme = "ws"
	}
	rest := c.baseURL
	if len(rest) >= 6 && rest[:6] == "https:" {
		rest = rest[6:]
	} else if len(rest) >= 5 && rest[:5] == "http:" {
		rest = rest[5:]
	}
	if rest == "" {
		rest = "//localhost"
	}
	return scheme + ":" + rest + "/api/current"
}

// Ping verifies login works by calling system.info.
func (c *Client) Ping() error {
	if _, err := c.Call("system.info", []any{}); err != nil {
		return err
	}
	return nil
}

// Call invokes a JSON-RPC method and waits for the response.
// `params` is the positional args array (may be empty slice).
// Returns the result as a generic any (typically a map[string]any or slice).
//
// On TRANSPORT errors (timeout, ws read closed, send failure) we reconnect
// and retry once with a fresh login. Application-level errors (EBUSY
// rate limiting, validation, ENOENT) are returned immediately without
// retry — otherwise we'd hit the SCALE middleware rate limit harder.
func (c *Client) Call(method string, params []any) (any, error) {
	if err := c.connect(); err != nil {
		return nil, err
	}
	res, err, transient := c.callOnce(method, params)
	if err == nil {
		return res, nil
	}

	// Rate-limited? Mark and refuse to retry — another login attempt
	// would just hit the same rate limit.
	if cerr, ok := err.(*ErrCall); ok && cerr.IsRateLimited() {
		c.noteRateLimited()
		return nil, err
	}
	if err == ErrRateLimited {
		return nil, err
	}
	if !transient {
		return nil, err
	}
	// transient transport error → reconnect once + retry
	_ = c.dropConnection()
	if err2 := c.connect(); err2 != nil {
		return nil, err
	}
	res, err, _ = c.callOnce(method, params)
	return res, err
}

// callOnce does a single RPC round-trip with a 30s deadline.
// transient=true means the connection probably broke (worth a retry).
func (c *Client) callOnce(method string, params []any) (any, error, bool) {
	_, respCh, err := c.send(method, params)
	if err != nil {
		return nil, err, true // send failure → transient
	}
	select {
	case resp := <-respCh:
		if resp.Error != nil {
			if resp.Error.unauth() {
				return nil, errors.New("unauthorized"), false
			}
			return nil, resp.Error.asErr(), false // API error → not transient
		}
		return resp.Result, nil, false
	case <-time.After(callTimeout):
		_ = c.dropConnection()
		return nil, fmt.Errorf("truenas: %s timed out after %s", method, callTimeout), true
	}
}

// dropConnection closes the current WS connection so the next connect()
// dials a fresh one.
func (c *Client) dropConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.closed = true
	// drain pending
	for id, ch := range c.pendng {
		select {
		case ch <- rpcResp{Error: &rpcError{Message: "connection reset"}}:
		default:
		}
		delete(c.pendng, id)
	}
	return nil
}

// Query is the most common JSON-RPC shape: pass filter rows (or empty) and
// get a list back. SCALE middleware rejects nil filters — must be a
// non-nil, properly-typed [][]any so it JSON-encodes as `[[]]` not `[null]`.
func (c *Client) Query(method string, filters [][]any, extraOpts ...map[string]any) ([]any, error) {
	var filterArr [][]any = filters
	if filterArr == nil {
		filterArr = [][]any{}
	}
	// Force the inner slice to type []any so JSON encoding is `[]` not `null`.
	wrapped := make([]any, len(filterArr))
	for i, row := range filterArr {
		wrapped[i] = toAnySlice(row)
	}

	params := []any{wrapped}
	if len(extraOpts) > 0 {
		opts := map[string]any{}
		for _, e := range extraOpts {
			for k, v := range e {
				opts[k] = v
			}
		}
		params = append(params, opts)
	}
	res, err := c.Call(method, params)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	if arr, ok := res.([]any); ok {
		return arr, nil
	}
	return []any{res}, nil
}

// toAnySlice converts any []T (or nil) into []any for proper JSON encoding.
func toAnySlice(s any) []any {
	switch v := s.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case []int:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case []int64:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = x
		}
		return out
	case nil:
		return []any{}
	default:
		// best effort: return single-element slice
		return []any{s}
	}
}

// Create-style call: pass a single argument struct, get back the new object ID or full row.
// Accepts a positional args slice like c.doCreate("pool.dataset.create", []any{obj})
func (c *Client) doCreate(method string, args []any) (any, error) {
	return c.Call(method, args)
}

// doDelete: pass an int64 ID (positional).
func (c *Client) doDelete(method string, id int64) error {
	_, err := c.Call(method, []any{id})
	return err
}

// -- transport --

type rpcReq struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcErrData struct {
	Errname string `json:"errname"`
	Reason  string `json:"reason"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *rpcErrData `json:"data"`
}

func (e *rpcError) unauth() bool {
	return e.Data != nil && (e.Data.Errname == "ENOTAUTHENTICATED" ||
		strings.Contains(strings.ToLower(e.Data.Reason), "not authenticated"))
}

func (e *rpcError) asErr() error {
	if e.Data != nil && e.Data.Reason != "" {
		return &ErrCall{Errname: e.Data.Errname, Reason: e.Data.Reason}
	}
	return &ErrCall{Errname: fmt.Sprintf("code=%d", e.Code), Reason: e.Message}
}

type rpcResp struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  any       `json:"result"`
	Error   *rpcError `json:"error"`
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil && !c.closed {
		return nil
	}
	if c.isRateLimited() {
		return ErrRateLimited
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDialContext:   nil, // use default dialer
	}
	if !c.verifyTLS {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	conn, _, err := dialer.Dial(c.wsURL(), nil)
	if err != nil {
		return fmt.Errorf("truenas: ws dial: %w", err)
	}
	c.conn = conn
	c.pendng = map[int]chan rpcResp{}
	c.idSeq = 0
	c.closed = false
	go c.readPump()
	// login
	c.mu.Unlock()
	err = c.login()
	c.mu.Lock()
	if err != nil {
		if cerr, ok := err.(*ErrCall); ok && cerr.IsRateLimited() {
			c.noteRateLimited()
		}
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("truenas: login: %w", err)
	}
	return nil
}

func (c *Client) login() error {
	res, err := c.Call("auth.login", []any{c.username, c.password})
	if err != nil {
		return err
	}
	if b, ok := res.(bool); ok && b {
		return nil
	}
	return fmt.Errorf("truenas: login returned false")
}

func (c *Client) reLogin() error {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
	return c.connect()
}

func (c *Client) send(method string, params []any) (int, chan rpcResp, error) {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return 0, nil, errors.New("truenas: not connected")
	}
	c.idSeq++
	id := c.idSeq
	ch := make(chan rpcResp, 1)
	c.pendng[id] = ch
	req := rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pendng, id)
		c.mu.Unlock()
		return 0, nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		delete(c.pendng, id)
		return 0, nil, err
	}
	return id, ch, nil
}

func (c *Client) readPump() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			for _, ch := range c.pendng {
				ch <- rpcResp{Error: &rpcError{Message: "ws read: " + err.Error()}}
			}
			c.pendng = map[int]chan rpcResp{}
			c.closed = true
			c.mu.Unlock()
			return
		}
		var resp rpcResp
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}
		c.mu.Lock()
		if ch, ok := c.pendng[resp.ID]; ok {
			delete(c.pendng, resp.ID)
			ch <- resp
			c.mu.Unlock()
			continue
		}
		c.mu.Unlock()
	}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
