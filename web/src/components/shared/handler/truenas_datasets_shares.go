package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// datasetOpts mirrors truenas.DatasetCreateOptions but is used at the
// handler boundary so we don't import the client types directly.
type datasetReq struct {
	HostID      string `json:"host_id"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Compression string `json:"compression,omitempty"`
	RecordSize  string `json:"recordsize,omitempty"`
	Quota       int64  `json:"quota,omitempty"`
	RefQuota    int64  `json:"refquota,omitempty"`
	Encryption  bool   `json:"encryption,omitempty"`
	ACLMode     string `json:"aclmode,omitempty"`
}

func toDatasetOpts(r datasetReq) truenas.DatasetCreateOptions {
	opts := truenas.DatasetCreateOptions{
		Name:        r.Name,
		Type:        r.Type,
		Compression: r.Compression,
		RecordSize:  parseRecordSize(r.RecordSize),
		ACLMode:     r.ACLMode,
	}
	if r.Quota > 0 {
		q := r.Quota
		opts.Quota = &q
	}
	if r.RefQuota > 0 {
		rq := r.RefQuota
		opts.RefQuota = &rq
	}
	if r.Encryption {
		e := true
		opts.Encryption = &e
	}
	return opts
}

// --- T2.3 — Datasets ---

// ListDatasets returns datasets on a host (optionally filtered by pool).
// Body fields: host_id, pool (optional).
func ListDatasets(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r struct {
			HostID string `json:"host_id"`
			Pool   string `json:"pool,omitempty"`
		}
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCredsFromBody(c, store, r.HostID)
		if !ok {
			return
		}
		ds, err := cli.ListDatasets(r.Pool)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"datasets": ds, "total": len(ds)})
	}
}

// GetDataset returns one dataset.
func GetDataset(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name   string `json:"name"`
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
		ds, err := cli.GetDataset(r.Name)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, ds)
	}
}

// CreateDataset creates a dataset.
func CreateDataset(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r datasetReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		ds, err := cli.CreateDataset(toDatasetOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, ds)
	}
}

// datasetUpdReq is the partial-update body for PATCH/POST update.
type datasetUpdReq struct {
	HostID      string `json:"host_id"`
	Name        string `json:"name"`
	Compression string `json:"compression,omitempty"`
	RecordSize  string `json:"recordsize,omitempty"`
	Quota       int64  `json:"quota,omitempty"`
	RefQuota    int64  `json:"refquota,omitempty"`
	ACLMode     string `json:"aclmode,omitempty"`
}

func toDatasetUpd(r datasetUpdReq) truenas.DatasetUpdateOptions {
	return truenas.DatasetUpdateOptions{
		Compression: r.Compression, RecordSize: parseRecordSize(r.RecordSize),
		Quota: r.Quota, RefQuota: r.RefQuota, ACLMode: r.ACLMode,
	}
}

// UpdateDataset changes properties (quota, compression, etc.).
func UpdateDataset(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r datasetUpdReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.UpdateDataset(r.Name, toDatasetUpd(r)); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updated": true})
	}
}

// DeleteDataset removes a dataset.
func DeleteDataset(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID    string `json:"host_id"`
		Name      string `json:"name"`
		Recursive bool   `json:"recursive"`
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
		if err := cli.DeleteDataset(r.Name, r.Recursive); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

// --- T2.4 — NFS ---

func ListNFS(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListNFS()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"shares": out, "total": len(out)})
	}
}
func CreateNFS(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r nfsShareReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateNFS(nfsShareOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

type nfsShareReq struct {
	HostID   string   `json:"host_id"`
	Path     string   `json:"path"`
	Comment  string   `json:"comment,omitempty"`
	Enabled  bool     `json:"enabled"`
	ReadOnly bool     `json:"ro,omitempty"`
	Networks []string `json:"networks,omitempty"`
	Hosts    []string `json:"hosts,omitempty"`
}

func nfsShareOpts(r nfsShareReq) truenas.NFSShareCreate {
	return truenas.NFSShareCreate{
		Path: r.Path, Comment: r.Comment,
		Enabled: r.Enabled, ReadOnly: r.ReadOnly,
		Networks: r.Networks, Hosts: r.Hosts,
	}
}
func UpdateNFS(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID   string   `json:"host_id"`
		ID       int64    `json:"id"`
		Enabled  bool     `json:"enabled"`
		ReadOnly bool     `json:"ro"`
		Paths    []string `json:"paths"`
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
		if len(r.Paths) == 0 {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", "paths required for NFS update")
			return
		}
		creq := truenas.NFSShareCreate{
			Path:     r.Paths[0],
			Enabled:  r.Enabled,
			ReadOnly: r.ReadOnly,
		}
		if err := cli.UpdateNFS(r.ID, creq); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updated": true})
	}
}
func DeleteNFS(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteNFS(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

// --- T2.5 — SMB ---

func ListSMB(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListSMB()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"shares": out, "total": len(out)})
	}
}
func CreateSMB(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r smbShareReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateSMB(smbShareOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

type smbShareReq struct {
	HostID     string   `json:"host_id"`
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Comment    string   `json:"comment,omitempty"`
	Enabled    bool     `json:"enabled"`
	ReadOnly   bool     `json:"ro,omitempty"`
	Browseable bool     `json:"browseable,omitempty"`
	GuestOk    bool     `json:"guestok,omitempty"`
	HostsAllow []string `json:"hostsallow,omitempty"`
	HostsDeny  []string `json:"hostsdeny,omitempty"`
}

func smbShareOpts(r smbShareReq) truenas.SMBShareCreate {
	return truenas.SMBShareCreate{
		Name: r.Name, Path: r.Path, Comment: r.Comment,
		Enabled: r.Enabled, ReadOnly: r.ReadOnly, Browseable: r.Browseable,
		GuestOk: r.GuestOk, HostsAllow: r.HostsAllow, HostsDeny: r.HostsDeny,
	}
}
func UpdateSMB(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID   string `json:"host_id"`
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Path     string `json:"path"`
		Enabled  bool   `json:"enabled"`
		ReadOnly bool   `json:"ro"`
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
		creq := truenas.SMBShareCreate{
			Name: r.Name, Path: r.Path,
			Enabled: r.Enabled, ReadOnly: r.ReadOnly,
		}
		if err := cli.UpdateSMB(r.ID, creq); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updated": true})
	}
}
func DeleteSMB(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteSMB(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}
