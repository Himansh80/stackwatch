package handler

// monitoring_fleet.go — /api/v1/fleet/summary endpoint returning aggregated metrics.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// FleetSummary returns aggregated fleet metrics
func FleetSummary(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		type Summary struct {
			TotalServers    int     `json:"total_servers"`
			UpServers       int     `json:"up_servers"`
			DownServers     int     `json:"down_servers"`
			StaleServers    int     `json:"stale_servers"`
			AvgCPU          float64 `json:"avg_cpu"`
			AvgMemory       float64 `json:"avg_memory"`
			AvgDisk         float64 `json:"avg_disk"`
			AvgLoad         float64 `json:"avg_load"`
			TotalProcesses  int64   `json:"total_processes"`
			TotalContainers int64   `json:"total_containers"`
		}

		var summary Summary
		err := pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT 
				COUNT(*) as total_servers,
				COUNT(*) FILTER (WHERE status = 'up') as up_servers,
				COUNT(*) FILTER (WHERE status = 'down') as down_servers,
				COUNT(*) FILTER (WHERE status = 'stale') as stale_servers,
				COALESCE(AVG((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'cpu.usage' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as avg_cpu,
				COALESCE(AVG((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'memory.usage' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as avg_memory,
				COALESCE(AVG((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'disk.usage' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as avg_disk,
				COALESCE(AVG((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'system.load' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as avg_load,
				COALESCE(SUM((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'processes.count' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as total_processes,
				COALESCE(SUM((SELECT value FROM metric_points mp WHERE mp.server_id = s.id AND mp.metric_name = 'containers.count' AND mp.ts > NOW() - INTERVAL '5 minutes' ORDER BY mp.ts DESC LIMIT 1)), 0) as total_containers
			FROM servers s
			WHERE s.tenant_id = $1 AND s.deleted_at IS NULL
		`, tenantID).Scan(&summary.TotalServers, &summary.UpServers, &summary.DownServers, &summary.StaleServers,
			&summary.AvgCPU, &summary.AvgMemory, &summary.AvgDisk, &summary.AvgLoad, &summary.TotalProcesses, &summary.TotalContainers)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}

// IngestHeartbeat handles agent heartbeat + metrics ingestion
// PUBLIC endpoint — no JWT required. Tenant resolved via server_id OR hostname.
