package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// --- T2.6 — iSCSI ---

// Extents

func ListISCSIExtents(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListISCSIExtents()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"extents": out, "total": len(out)})
	}
}

type iscsiExtentReq struct {
	HostID    string `json:"host_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Disk      string `json:"disk,omitempty"`
	FilePath  string `json:"path,omitempty"`
	Blocksize int    `json:"blocksize,omitempty"`
	RPM       string `json:"rpm,omitempty"`
}

func iscsiExtentOpts(r iscsiExtentReq) truenas.ISCSIExtentCreate {
	return truenas.ISCSIExtentCreate{
		Name: r.Name, Type: r.Type, Disk: r.Disk,
		FilePath: r.FilePath, Blocksize: r.Blocksize, RPM: r.RPM,
	}
}

func CreateISCSIExtent(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r iscsiExtentReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateISCSIExtent(iscsiExtentOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DeleteISCSIExtent(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteISCSIExtent(r.ID, false); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

// Targets

func ListISCSITargets(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListISCSITargets()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"targets": out, "total": len(out)})
	}
}

type iscsiTargetReq struct {
	HostID      string   `json:"host_id"`
	Name        string   `json:"name"`
	Alias       string   `json:"alias,omitempty"`
	ModeIscsi   bool     `json:"mode_is"`
	ModeFC      bool     `json:"mode_fc"`
	AuthNetwork []string `json:"auth_networks,omitempty"`
}

func iscsiTargetOpts(r iscsiTargetReq) truenas.ISCSITargetCreate {
	return truenas.ISCSITargetCreate{
		Name: r.Name, Alias: r.Alias,
		ModeIscsi: r.ModeIscsi, ModeFC: r.ModeFC,
		AuthNetwork: r.AuthNetwork,
	}
}

func CreateISCSITarget(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r iscsiTargetReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateISCSITarget(iscsiTargetOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DeleteISCSITarget(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteISCSITarget(r.ID, false); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

// Associated targets (target <-> extent linkages)

func ListISCSIAssociated(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListISCSIAssociated()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"associated": out, "total": len(out)})
	}
}

type iscsiAssocReq struct {
	HostID string `json:"host_id"`
	Target int64  `json:"target"`
	Extent int64  `json:"extent"`
	LUNID  int    `json:"lunid"`
}

func iscsiAssocOpts(r iscsiAssocReq) truenas.ISCSIAssociateCreate {
	return truenas.ISCSIAssociateCreate{Target: r.Target, Extent: r.Extent, LUNID: r.LUNID}
}

func AssociateISCSI(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r iscsiAssocReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.AssociateISCSI(iscsiAssocOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DissociateISCSI(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DissociateISCSI(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}
