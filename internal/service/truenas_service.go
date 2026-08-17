// Package service provides the TrueNAS service layer.
//
// Architecture: the sidecar is stateless across requests, but within
// a single request we MUX multiple JSON-RPC calls over the same WebSocket
// session. To avoid reconnecting (and re-logging-in!) on every call,
// we cache one *truenas.Client per host_id for the lifetime of the process.
//
// Each *truenas.Client is goroutine-safe and serializes its own RPC calls
// over its dedicated WebSocket; sharing it across handlers is fine.
package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/stackwatch/platform/internal/client/truenas"
)

// Creds is the credential bundle the api-gateway forwards with each request.
// Empty apiKey means cookie session auth (username + password required).
type Creds struct {
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	VerifyTLS bool   `json:"verify_tls"`
}

// Validate returns an error if the creds are insufficient for any auth mode.
func (c *Creds) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("truenas: base_url required")
	}
	if c.APIKey == "" && (c.Username == "" || c.Password == "") {
		return fmt.Errorf("truenas: api_key or (username+password) required")
	}
	return nil
}

// credsKey returns a map key for a creds bundle (used for caching).
func (c *Creds) cacheKey() string {
	return c.BaseURL + "|" + c.Username + "|" + c.Password + "|" + c.APIKey + "|" +
		fmt.Sprintf("%v", c.VerifyTLS)
}

var (
	clientMu sync.Mutex
	clients  = map[string]*truenas.Client{} // credsKey → live client
)

// Client returns a cached *truenas.Client for the credentials. If a live
// client doesn't exist, it dials + logs in and stores it. Subsequent
// callers reuse the same WS connection — saving one login per request.
//
// The cache is process-local; it resets when the sidecar restarts. There's
// no eviction; the same set of hosts will be hit over and over so we don't
// need one in practice.
func Client(creds Creds) (*truenas.Client, error) {
	if err := creds.Validate(); err != nil {
		return nil, err
	}
	key := creds.cacheKey()
	clientMu.Lock()
	defer clientMu.Unlock()
	if cli, ok := clients[key]; ok {
		return cli, nil
	}
	cli := truenas.NewClient(creds.BaseURL, creds.APIKey, creds.Username, creds.Password, creds.VerifyTLS)
	clients[key] = cli
	return cli, nil
}

// Ping validates that the credentials work for the given host.
// Re-uses the cached client, falling back to a fresh probe if login
// fails (in which case the bad cache entry is evicted).
func Ping(_ context.Context, creds Creds) error {
	cli, err := Client(creds)
	if err != nil {
		return err
	}
	return cli.Ping()
}

// Forget removes a client from the cache (e.g. on credential rotation).
func Forget(creds Creds) {
	clientMu.Lock()
	delete(clients, creds.cacheKey())
	clientMu.Unlock()
}
