package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// --- T2.10 — System (info, boot envs, update, services) ---

func SystemInfo(store *hostStore) gin.HandlerFunc {
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
		info, err := cli.GetSystemInfo()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, info)
	}
}

func ListBootEnvironments(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListBootEnvironments()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"bootenvs": out, "total": len(out)})
	}
}

func ActivateBootEnvironment(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name string `json:"name"`
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
		if err := cli.ActivateBootEnvironment(r.Name); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"activated": r.Name})
	}
}

func DestroyBootEnvironment(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name string `json:"name"`
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
		if err := cli.DestroyBootEnvironment(r.Name); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": r.Name})
	}
}

func CheckUpdate(store *hostStore) gin.HandlerFunc {
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
		info, err := cli.CheckUpdate()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, info)
	}
}

func ApplyUpdate(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Train        string `json:"train,omitempty"`
		DownloadOnly bool   `json:"download_only"`
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
		if err := cli.DownloadUpdate(r.Train); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updating": true})
	}
}

func ListServices(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListServices()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"services": out, "total": len(out)})
	}
}

func RunService(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name   string `json:"name"`
		Action string `json:"action"` // start | stop | restart | reload
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
		var err error
		switch r.Action {
		case "start":
			err = cli.StartService(r.Name)
		case "stop":
			err = cli.StopService(r.Name)
		case "restart":
			err = cli.RestartService(r.Name)
		case "reload":
			err = cli.ReloadService(r.Name)
		default:
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", "unknown action: "+r.Action)
			return
		}
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"ran": r.Action, "service": r.Name})
	}
}

func SetServiceAutoStart(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		Name   string `json:"name"`
		Enable bool   `json:"enable"`
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
		if err := cli.UpdateService(r.Name, truenas.ServiceUpdate{Enable: r.Enable}); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"set": r.Name, "enable": r.Enable})
	}
}

// --- T2.11 — Cloud Sync ---

func ListCloudCredentials(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListCloudCredentials()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"credentials": out, "total": len(out)})
	}
}

type cloudCredReq struct {
	HostID string `json:"host_id"`
	Name       string         `json:"name"`
	Provider   string         `json:"provider"`
	Attributes map[string]any `json:"attributes"`
}

func cloudCredOpts(r cloudCredReq) truenas.CloudCredentialCreate {
	return truenas.CloudCredentialCreate{Name: r.Name, Provider: r.Provider, Attributes: r.Attributes}
}

func CreateCloudCredential(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r cloudCredReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateCloudCredential(cloudCredOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DeleteCloudCredential(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteCloudCredential(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

func ListCloudSyncTasks(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListCloudSyncTasks()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"tasks": out, "total": len(out)})
	}
}

type cloudSyncReq struct {
	HostID string `json:"host_id"`
	Description string `json:"description"`
	Direction   string `json:"direction"`
	Path        string `json:"path"`
	Credential  int64  `json:"credentials"`
	Bucket      string `json:"bucket"`
	Folder      string `json:"folder,omitempty"`
	Encryption  bool   `json:"encryption"`
	Schedule    string `json:"schedule,omitempty"`
	Enabled     bool   `json:"enabled"`
}

func cloudSyncOpts(r cloudSyncReq) truenas.CloudSyncTaskCreate {
	return truenas.CloudSyncTaskCreate{
		Description: r.Description, Direction: r.Direction, Path: r.Path,
		Credential: r.Credential, Bucket: r.Bucket, Folder: r.Folder,
		Encryption: r.Encryption, Schedule: r.Schedule, Enabled: r.Enabled,
	}
}

func CreateCloudSyncTask(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r cloudSyncReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateCloudSyncTask(cloudSyncOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func UpdateCloudSyncTask(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID int64 `json:"id"`
		cloudSyncReq
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
		if err := cli.UpdateCloudSyncTask(r.ID, cloudSyncOpts(r.cloudSyncReq)); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updated": true})
	}
}

func DeleteCloudSyncTask(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteCloudSyncTask(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

func RunCloudSyncTask(store *hostStore) gin.HandlerFunc {
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
		jid, runErr := cli.RunCloudSyncTask(r.ID)
		if runErr != nil {
			Err(c, runErr)
			return
		}
		OK(c, gin.H{"started": true, "job_id": jid})
	}
}

func DryRunCloudSyncTask(store *hostStore) gin.HandlerFunc {
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
		jid, runErr := cli.DryRunCloudSyncTask(r.ID)
		if runErr != nil {
			Err(c, runErr)
			return
		}
		OK(c, gin.H{"dry_run": true, "job_id": jid})
	}
}
