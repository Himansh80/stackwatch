package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// ListPools returns all ZFS pools on a host.
func ListPools(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
				var r struct {
			HostID string `json:"host_id"`
		}
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		pools, err := cli.ListPools()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"pools": pools, "total": len(pools)})
	}
}

// GetPool returns one pool by id.
func GetPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID     int64  `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		p, err := cli.GetPool(r.ID)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, p)
	}
}

// CreatePool creates a new pool from a vdev topology spec.
func CreatePool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name          string         `json:"name"`
		Topology      map[string]any `json:"topology"`
		Deduplication bool           `json:"deduplication"`
		Encryption    bool           `json:"encryption"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreatePool(truenas.PoolCreateOptions{
			Name:       r.Name,
			Topology:   r.Topology,
			Encryption: r.Encryption,
			Dedup:      boolToDedup(r.Deduplication),
		})
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

// ExtendPool adds a vdev to an existing pool.
func ExtendPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		NewVDev map[string]any `json:"new_vdev"`
		ID      int64          `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.ExtendPool(r.ID, []map[string]any{r.NewVDev}); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"extended": true})
	}
}

// ImportPool imports an existing pool by GUID.
func ImportPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		GUID string `json:"guid"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.ImportPool(r.GUID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"imported": true})
	}
}

// ExportPool exports (unmounts) a pool.
func ExportPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID    int64 `json:"id"`
		Force bool  `json:"force"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.ExportPool(r.ID, false, r.Force); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"exported": true})
	}
}

// DestroyPool wipes a pool (caller must confirm).
func DestroyPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID int64 `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.ExportPool(r.ID, true, false); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"destroyed": true})
	}
}

// ScrubPool starts a pool scrub.
func ScrubPool(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID int64 `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.ScrubPool(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"scrubbing": true})
	}
}

// PoolScrubState returns current scrub state.
func PoolScrubState(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID int64 `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		state, err := cli.PoolScrub(r.ID)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, state)
	}
}

// silence unused import (if all client methods aren't used)
var _ = truenas.Pool{}

// parseRecordSize converts a TrueNAS-style record size string ("128K",
// "1M") to bytes for the SCALE API.
func parseRecordSize(s string) int64 {
	if s == "" {
		return 0
	}
	multiplier := int64(1)
	last := s[len(s)-1]
	num := s
	switch last {
	case 'K', 'k':
		multiplier = 1024
		num = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		num = s[:len(s)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		num = s[:len(s)-1]
	}
	var n int64
	for _, c := range num {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	return n * multiplier
}

// boolToDedup converts a boolean toggle into the SCALE API value.
func boolToDedup(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}
