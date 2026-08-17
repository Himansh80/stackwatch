package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/service"
)

// CreateHost registers a new TrueNAS system.
// Tests the connection before persisting; returns 400 if the host is unreachable.
func CreateHost(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConnectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if req.Name == "" {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", "name required")
			return
		}
		if req.BaseURL == "" {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", "base_url required")
			return
		}
		creds := CredsFromHost(&Host{
			BaseURL:   req.BaseURL,
			APIKey:    req.APIKey,
			Username:  req.Username,
			Password:  req.Password,
			VerifyTLS: req.VerifyTLS,
		})
		if err := creds.Validate(); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if err := service.Ping(c.Request.Context(), creds); err != nil {
			Bad(c, http.StatusBadRequest, "UNREACHABLE", err.Error())
			return
		}
		h := &Host{
			Name:      req.Name,
			BaseURL:   req.BaseURL,
			APIKey:    req.APIKey,
			Username:  req.Username,
			Password:  req.Password,
			VerifyTLS: req.VerifyTLS,
			Status:    "online",
		}
		if err := store.create(h); err != nil {
			Bad(c, http.StatusInternalServerError, "INTERNAL", err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"host": h})
	}
}

// ListHosts returns all known TrueNAS systems (no secrets stripped for the sidecar).
func ListHosts(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"hosts": store.list()})
	}
}

// GetHost returns one host by ID.
func GetHost(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		h, ok := store.get(id)
		if !ok {
			Bad(c, http.StatusNotFound, "NOT_FOUND", "host not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"host": h})
	}
}

// DeleteHost removes a host from the registry.
func DeleteHost(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := store.delete(id); err != nil {
			Bad(c, http.StatusInternalServerError, "INTERNAL", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true, "id": id})
	}
}

// TestHost re-runs the ping against the stored credentials and updates the
// host's status. Useful after fixing credential rotation on the TrueNAS side.
func TestHost(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		h, ok := store.get(id)
		if !ok {
			Bad(c, http.StatusNotFound, "NOT_FOUND", "host not found")
			return
		}
		err := service.Ping(c.Request.Context(), CredsFromHost(h))
		if err != nil {
			store.updateStatus(id, "offline", err.Error())
			Bad(c, http.StatusBadGateway, "OFFLINE", err.Error())
			return
		}
		store.updateStatus(id, "online", "")
		c.JSON(http.StatusOK, gin.H{"status": "online", "host_id": id})
	}
}
