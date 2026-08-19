package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

func (h *ProxmoxHandler) monitoringClient(c *gin.Context) (*proxmox.Client, bool) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return nil, false
	}
	return proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS), true
}

func (h *ProxmoxHandler) HostMonitoring(c *gin.Context) {
	cli, ok := h.monitoringClient(c)
	if !ok {
		return
	}
	node := c.Param("node")
	if node == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	points, err := cli.HostRRD(c.Request.Context(), node, c.DefaultQuery("timeframe", "hour"), c.DefaultQuery("cf", "AVERAGE"))
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": node, "points": points, "total": len(points)})
}

func (h *ProxmoxHandler) ResourceMonitoring(c *gin.Context) {
	cli, ok := h.monitoringClient(c)
	if !ok {
		return
	}
	kind := c.Param("kind")
	if kind != "qemu" && kind != "lxc" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	vmid, err := parseVMID(c.Param("vmid"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	points, err := cli.ResourceRRD(c.Request.Context(), c.Param("node"), kind, vmid, c.DefaultQuery("timeframe", "hour"), c.DefaultQuery("cf", "AVERAGE"))
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": c.Param("node"), "kind": kind, "vmid": vmid, "points": points, "total": len(points)})
}

// MonitoringAlerts evaluates the latest RRD point against caller-supplied
// thresholds. It is intentionally read-only; alert persistence belongs to the
// platform alerting tier.
func (h *ProxmoxHandler) MonitoringAlerts(c *gin.Context) {
	cli, ok := h.monitoringClient(c)
	if !ok {
		return
	}
	node := c.Param("node")
	points, err := cli.HostRRD(c.Request.Context(), node, c.DefaultQuery("timeframe", "hour"), c.DefaultQuery("cf", "AVERAGE"))
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	cpuLimit, _ := strconv.ParseFloat(c.DefaultQuery("cpu_gt", "0.9"), 64)
	memLimit, _ := strconv.ParseFloat(c.DefaultQuery("memory_gt", "0.9"), 64)
	alerts := make([]gin.H, 0)
	if len(points) > 0 {
		latest := points[len(points)-1]
		if cpu, ok := numericRRD(latest, "cpu"); ok && cpuLimit > 0 && cpu > cpuLimit {
			alerts = append(alerts, gin.H{"metric": "cpu", "value": cpu, "threshold": cpuLimit, "severity": "warning"})
		}
		if mem, ok := numericRRD(latest, "memused"); ok && memLimit > 0 && mem > memLimit {
			alerts = append(alerts, gin.H{"metric": "memused", "value": mem, "threshold": memLimit, "severity": "warning"})
		}
	}
	c.JSON(http.StatusOK, gin.H{"node": node, "alerts": alerts, "total": len(alerts), "evaluated_points": len(points)})
}

func numericRRD(point proxmox.RRDPoint, key string) (float64, bool) {
	v, ok := point[key]
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, ok
	}
}
