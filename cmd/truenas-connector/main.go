// Command truenas-connector is the Tier 2 (TrueNAS SCALE replacement) sidecar.
//
// It listens on :8088 by default (override with TRUENAS_CONNECTOR_ADDR)
// and exposes:
//   - /health                          liveness probe
//   - /hosts                           CRUD for TrueNAS systems
//   - /pools, /datasets, /nfs, /smb,
//     /iscsi/{extents,targets},
//     /snapshots, /replications, /disks,
//     /users, /groups, /acl,
//     /system/{info,bootenv,update,services},
//     /cloud/{credentials,sync}        per-feature proxy endpoints (T2.2-T2.11)
//
// The api-gateway authenticates the user + resolves the tenant's trueNAS host
// row, then forwards to these endpoints with the credentials in the body.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/handler"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr := os.Getenv("TRUENAS_CONNECTOR_ADDR")
	if addr == "" {
		addr = ":8088"
	}
	dataDir := os.Getenv("TRUENAS_CONNECTOR_DATA")
	if dataDir == "" {
		dataDir = "/var/lib/stackwatch/truenas-connector"
	}

	logger.Info("truenas-connector starting", "addr", addr, "data_dir", dataDir)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := handler.NewHostStore(dataDir)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggingMiddleware(logger))

	// Liveness
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "truenas-connector"}) })

	// Host CRUD (T2.1)
	r.GET("/hosts", handler.ListHosts(store))
	r.POST("/hosts", handler.CreateHost(store))
	r.GET("/hosts/:id", handler.GetHost(store))
	r.DELETE("/hosts/:id", handler.DeleteHost(store))
	r.POST("/hosts/:id/test", handler.TestHost(store))

	// T2.2 - Pools
	r.POST("/pools/list", handler.ListPools(store))
	r.POST("/pools/get", handler.GetPool(store))
	r.POST("/pools/create", handler.CreatePool(store))
	r.POST("/pools/extend", handler.ExtendPool(store))
	r.POST("/pools/import", handler.ImportPool(store))
	r.POST("/pools/export", handler.ExportPool(store))
	r.POST("/pools/destroy", handler.DestroyPool(store))
	r.POST("/pools/scrub", handler.ScrubPool(store))
	r.POST("/pools/scrub_state", handler.PoolScrubState(store))

	// T2.3 - Datasets
	r.POST("/datasets/list", handler.ListDatasets(store))
	r.POST("/datasets/get", handler.GetDataset(store))
	r.POST("/datasets/create", handler.CreateDataset(store))
	r.POST("/datasets/update", handler.UpdateDataset(store))
	r.POST("/datasets/delete", handler.DeleteDataset(store))

	// T2.4 - NFS shares
	r.POST("/nfs/list", handler.ListNFS(store))
	r.POST("/nfs/create", handler.CreateNFS(store))
	r.POST("/nfs/update", handler.UpdateNFS(store))
	r.POST("/nfs/delete", handler.DeleteNFS(store))

	// T2.5 - SMB shares
	r.POST("/smb/list", handler.ListSMB(store))
	r.POST("/smb/create", handler.CreateSMB(store))
	r.POST("/smb/update", handler.UpdateSMB(store))
	r.POST("/smb/delete", handler.DeleteSMB(store))

	// T2.6 - iSCSI
	r.POST("/iscsi/extents/list", handler.ListISCSIExtents(store))
	r.POST("/iscsi/extents/create", handler.CreateISCSIExtent(store))
	r.POST("/iscsi/extents/delete", handler.DeleteISCSIExtent(store))
	r.POST("/iscsi/targets/list", handler.ListISCSITargets(store))
	r.POST("/iscsi/targets/create", handler.CreateISCSITarget(store))
	r.POST("/iscsi/targets/delete", handler.DeleteISCSITarget(store))
	r.POST("/iscsi/associated/list", handler.ListISCSIAssociated(store))
	r.POST("/iscsi/associated/create", handler.AssociateISCSI(store))
	r.POST("/iscsi/associated/delete", handler.DissociateISCSI(store))

	// T2.7 - Snapshots + replication
	r.POST("/snapshots/list", handler.ListSnapshots(store))
	r.POST("/snapshots/create", handler.CreateSnapshot(store))
	r.POST("/snapshots/delete", handler.DeleteSnapshot(store))
	r.POST("/snapshots/rollback", handler.RollbackSnapshot(store))
	r.POST("/replications/list", handler.ListReplications(store))
	r.POST("/replications/create", handler.CreateReplication(store))
	r.POST("/replications/delete", handler.DeleteReplication(store))
	r.POST("/replications/run", handler.RunReplication(store))

	// T2.8 - Disks
	r.POST("/disks/list", handler.ListDisks(store))
	r.POST("/disks/smart_run", handler.SMARTDiskRun(store))
	r.POST("/disks/smart_history", handler.SMARTDiskHistory(store))
	r.POST("/disks/smart_results", handler.SMARTDiskResults(store))
	r.POST("/disks/replace", handler.ReplaceDisk(store))
	r.POST("/disks/wipe", handler.WipeDisk(store))

	// T2.9 - Users + groups + ACL
	r.POST("/users/list", handler.ListUsers(store))
	r.POST("/users/get", handler.GetUser(store))
	r.POST("/users/create", handler.CreateUser(store))
	r.POST("/users/update", handler.UpdateUser(store))
	r.POST("/users/delete", handler.DeleteUser(store))
	r.POST("/groups/list", handler.ListGroups(store))
	r.POST("/groups/create", handler.CreateGroup(store))
	r.POST("/groups/delete", handler.DeleteGroup(store))
	r.POST("/acl/get", handler.GetACL(store))
	r.POST("/acl/set", handler.SetACL(store))

	// T2.10 - System
	r.POST("/system/info", handler.SystemInfo(store))
	r.POST("/system/bootenv/list", handler.ListBootEnvironments(store))
	r.POST("/system/bootenv/activate", handler.ActivateBootEnvironment(store))
	r.POST("/system/bootenv/delete", handler.DestroyBootEnvironment(store))
	r.POST("/system/update/check", handler.CheckUpdate(store))
	r.POST("/system/update/apply", handler.ApplyUpdate(store))
	r.POST("/system/services/list", handler.ListServices(store))
	r.POST("/system/services/action", handler.RunService(store))
	r.POST("/system/services/autostart", handler.SetServiceAutoStart(store))

	// T2.11 - Cloud sync
	r.POST("/cloud/credentials/list", handler.ListCloudCredentials(store))
	r.POST("/cloud/credentials/create", handler.CreateCloudCredential(store))
	r.POST("/cloud/credentials/delete", handler.DeleteCloudCredential(store))
	r.POST("/cloud/sync/list", handler.ListCloudSyncTasks(store))
	r.POST("/cloud/sync/create", handler.CreateCloudSyncTask(store))
	r.POST("/cloud/sync/update", handler.UpdateCloudSyncTask(store))
	r.POST("/cloud/sync/delete", handler.DeleteCloudSyncTask(store))
	r.POST("/cloud/sync/run", handler.RunCloudSyncTask(store))
	r.POST("/cloud/sync/dry_run", handler.DryRunCloudSyncTask(store))

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("truenas-connector listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server crashed", "err", err)
			cancel()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stop:
		logger.Info("shutdown signal")
	case <-rootCtx.Done():
		logger.Info("ctx cancelled")
	}
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
}

// loggingMiddleware is the sidecar's own structured logger.
// We don't reuse internal/middleware because the sidecar is intentionally
// minimal — no request ID propagation, no JWT, no CORS.
func loggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("req",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(),
			"client", c.ClientIP(),
		)
	}
}
