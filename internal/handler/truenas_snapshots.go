package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// --- T2.7 — Snapshots & Replications ---

func ListSnapshots(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r struct {
			HostID  string `json:"host_id"`
			Dataset string `json:"dataset,omitempty"`
		}
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCredsFromBody(c, store, r.HostID)
		if !ok {
			return
		}
		out, err := cli.ListSnapshots(r.Dataset)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"snapshots": out, "total": len(out)})
	}
}

type snapshotReq struct {
	HostID    string `json:"host_id"`
	Dataset   string `json:"dataset"`
	Name      string `json:"name,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	Retention string `json:"retention,omitempty"`
	VMMemory  bool   `json:"vmware_sync,omitempty"`
}

func snapshotOpts(r snapshotReq) truenas.SnapshotCreate {
	return truenas.SnapshotCreate{
		Dataset: r.Dataset, Name: r.Name,
		Recursive: r.Recursive, VMSync: r.VMMemory,
	}
}

func CreateSnapshot(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r snapshotReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		name, err := cli.CreateSnapshot(snapshotOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"name": name})
	}
}

func DeleteSnapshot(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID    string `json:"host_id"`
		Name      string `json:"name"`    // accepts bare name ("my-snap") OR full id ("ds@my-snap")
		Dataset   string `json:"dataset"` // optional; if Name is bare, this is used to build the full id
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
		// If name doesn't already contain "@", prepend dataset to make a full id.
		name := r.Name
		if r.Dataset != "" && !strings.Contains(name, "@") {
			name = r.Dataset + "@" + name
		}
		if err := cli.DeleteSnapshot(name, r.Recursive); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

// RollbackSnapshot rolls back a dataset to a snapshot name.
func RollbackSnapshot(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID       string `json:"host_id"`
		Dataset      string `json:"dataset"`
		SnapshotName string `json:"snapshot"`
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
		if err := cli.CloneSnapshot(r.Dataset, r.SnapshotName, nil); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"rolled_back": true})
	}
}

// Replications

func ListReplications(store *hostStore) gin.HandlerFunc {
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
		out, err := cli.ListReplications()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"replications": out, "total": len(out)})
	}
}

type replicationReq struct {
	HostID        string `json:"host_id"`
	Name          string `json:"name"`
	Direction     string `json:"direction"`
	SourceDataset string `json:"source_datasets"`
	TargetDataset string `json:"target_dataset"`
	Recursive     bool   `json:"recursive"`
	Transport     string `json:"transport"`
	Schedule      string `json:"schedule,omitempty"`
	Enabled       bool   `json:"enabled"`
	SSHConnection int64  `json:"ssh_connection,omitempty"`
}

func replicationOpts(r replicationReq) truenas.ReplicationCreate {
	return truenas.ReplicationCreate{
		Name: r.Name, Direction: r.Direction,
		SourceDataset: []string{r.SourceDataset},
		TargetDataset: r.TargetDataset,
		Recursive:     r.Recursive, Transport: r.Transport,
		Schedule: r.Schedule, Enabled: r.Enabled,
		SSHConnection: r.SSHConnection,
	}
}

func CreateReplication(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r replicationReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateReplication(replicationOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DeleteReplication(store *hostStore) gin.HandlerFunc {
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
		if err := cli.DeleteReplication(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

func RunReplication(store *hostStore) gin.HandlerFunc {
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
		jid, runErr := cli.RunReplication(r.ID)
		if runErr != nil {
			Err(c, runErr)
			return
		}
		OK(c, gin.H{"started": true, "job_id": jid})
	}
}
