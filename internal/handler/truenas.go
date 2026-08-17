// Package handler holds the TrueNAS sidecar's HTTP handlers.
//
// The sidecar is stateless except for a small JSON host registry kept on
// disk (dataDir/truenas_hosts.json). The api-gateway calls these handlers
// to register / list / test / connect to TrueNAS systems.
//
// Sub-feature handlers (pools, datasets, etc.) are split into separate
// files matching the T2.x labels in MASTER_BUILD_PLAN.md.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/client/truenas"
	"github.com/stackwatch/platform/internal/service"
)

// Host is one TrueNAS registration the sidecar knows about.
type Host struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	VerifyTLS bool   `json:"verify_tls"`
	Status    string `json:"status"`
	LastError string `json:"last_error,omitempty"`
	CreatedAt string `json:"created_at"`
}

// HostsJSON is the on-disk store.
type HostsJSON struct {
	Hosts []Host `json:"hosts"`
}

// hostStore guards the in-memory cache + on-disk file.
type hostStore struct {
	mu    sync.RWMutex
	hosts map[string]*Host
	path  string
}

// NewHostStore is the constructor (lowercase-new pattern). It loads any
// existing on-disk registry, or starts fresh if no file is present.
func NewHostStore(dataDir string) *hostStore {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		panic(fmt.Errorf("truenas-connector: cannot create data dir: %w", err))
	}
	s := &hostStore{
		hosts: map[string]*Host{},
		path:  filepath.Join(dataDir, "truenas_hosts.json"),
	}
	_ = s.load()
	return s
}

func (s *hostStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var wrap HostsJSON
	if err := json.Unmarshal(data, &wrap); err != nil {
		return err
	}
	for i := range wrap.Hosts {
		s.hosts[wrap.Hosts[i].ID] = &wrap.Hosts[i]
	}
	return nil
}

func (s *hostStore) persist() error {
	out := HostsJSON{Hosts: make([]Host, 0, len(s.hosts))}
	for _, h := range s.hosts {
		out.Hosts = append(out.Hosts, *h)
	}
	sort.Slice(out.Hosts, func(i, j int) bool { return out.Hosts[i].CreatedAt < out.Hosts[j].CreatedAt })
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *hostStore) create(h *Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	if h.CreatedAt == "" {
		h.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if h.Status == "" {
		h.Status = "unknown"
	}
	s.hosts[h.ID] = h
	return s.persist()
}

func (s *hostStore) list() []Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Host, 0, len(s.hosts))
	for _, h := range s.hosts {
		out = append(out, *h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func (s *hostStore) get(id string) (*Host, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.hosts[id]
	return h, ok
}

func (s *hostStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.hosts, id)
	return s.persist()
}

func (s *hostStore) updateStatus(id, status, lastErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.hosts[id]; ok {
		h.Status = status
		h.LastError = lastErr
		_ = s.persist()
	}
}

// ConnectRequest is the body for POST /hosts (and most proxy endpoints).
type ConnectRequest struct {
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	VerifyTLS bool   `json:"verify_tls"`

	// Fields for proxy endpoints (pools, datasets, etc.) — embed per-call:
	HostID string `json:"host_id,omitempty"`
}

// ConnectResponse is returned for proxy endpoints where the client also
// wants the refreshed host status.
type ConnectResponse struct {
	Host   *Host  `json:"host,omitempty"`
	Status string `json:"status,omitempty"`
}

// CredsFromHost converts a stored Host into service.Creds.
func CredsFromHost(h *Host) service.Creds {
	return service.Creds{
		BaseURL:   h.BaseURL,
		APIKey:    h.APIKey,
		Username:  h.Username,
		Password:  h.Password,
		VerifyTLS: h.VerifyTLS,
	}
}

// ResolveHost pulls a host from store by ID.
// Returns (host, true) on success, (nil, false) when not found.
func ResolveHost(store *hostStore, id string) (*Host, bool) {
	if id == "" {
		return nil, false
	}
	return store.get(id)
}

// Common shell helpers used by every feature handler.

// OK responds 200 with the body.
func OK(c *gin.Context, body any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": body})
}

// Bad responds 4xx with an error envelope.
func Bad(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{"ok": false, "error": gin.H{"code": code, "message": msg}})
}

// Err translates a TrueNAS client error into a 4xx/5xx.
func Err(c *gin.Context, err error) {
	msg := err.Error()
	// truenas client uses fixed strings — keep simple for now
	c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": gin.H{"code": "TRUENAS", "message": msg}})
}

// ResolveCreds is the universal dispatcher. Handlers MUST have already
// bound the body and pass the host_id directly to avoid re-binding.
// Pass hostID="" to fall back to inline-credentials mode.
func ResolveCreds(c *gin.Context, store *hostStore, hostID string, inlineCreds *ConnectRequest) (*truenas.Client, *Host, bool) {
	if hostID == "" {
		if inlineCreds != nil && inlineCreds.BaseURL != "" {
			creds := service.Creds{
				BaseURL:   inlineCreds.BaseURL,
				APIKey:    inlineCreds.APIKey,
				Username:  inlineCreds.Username,
				Password:  inlineCreds.Password,
				VerifyTLS: inlineCreds.VerifyTLS,
			}
			cli, err := service.Client(creds)
			if err != nil {
				Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
				return nil, nil, false
			}
			return cli, nil, true
		}
		Bad(c, http.StatusBadRequest, "BAD_REQUEST", "host_id required")
		return nil, nil, false
	}
	h, ok := store.get(hostID)
	if !ok {
		Bad(c, http.StatusNotFound, "NOT_FOUND", "host not found")
		return nil, nil, false
	}
	cli, err := service.Client(CredsFromHost(h))
	if err != nil {
		Bad(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return nil, nil, false
	}
	return cli, h, true
}

// ResolveCredsFromBody resolves the host by ID without re-reading the
// request body. Use when the handler has already bound the body and
// extracted host_id.
func ResolveCredsFromBody(c *gin.Context, store *hostStore, hostID string) (*truenas.Client, *Host, bool) {
	if hostID == "" {
		Bad(c, http.StatusBadRequest, "BAD_REQUEST", "host_id required")
		return nil, nil, false
	}
	h, ok := store.get(hostID)
	if !ok {
		Bad(c, http.StatusNotFound, "NOT_FOUND", "host not found")
		return nil, nil, false
	}
	cli, err := service.Client(CredsFromHost(h))
	if err != nil {
		Bad(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return nil, nil, false
	}
	return cli, h, true
}
