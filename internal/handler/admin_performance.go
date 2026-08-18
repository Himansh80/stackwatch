// Tier 4 S9: Performance (per-core CPU, memory, disk I/O, per-NIC, OOM).
//
//   GET /api/v1/admin/performance?connection_id=X
//
// Reads from /proc and /sys on the target host. All commands are
// non-mutating and cheap.
package handler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CPUCore is one row from /proc/stat (cpu0, cpu1, ...).
type CPUCore struct {
	Core  string  `json:"core"`   // "cpu0", "cpu1", ...
	User  float64 `json:"user_pct"`
	Sys   float64 `json:"system_pct"`
	IO    float64 `json:"iowait_pct"`
	Idle  float64 `json:"idle_pct"`
	Steal float64 `json:"steal_pct"`
}

// DiskIO is one row from /proc/diskstats.
type DiskIO struct {
	Device        string `json:"device"`
	Reads         uint64 `json:"reads_completed"`
	Writes        uint64 `json:"writes_completed"`
	ReadSectors   uint64 `json:"read_sectors"`
	WriteSectors  uint64 `json:"write_sectors"`
	ReadTimeMs    uint64 `json:"read_time_ms"`
	WriteTimeMs   uint64 `json:"write_time_ms"`
}

// NetIO is one row from /proc/net/dev.
type NetIO struct {
	Interface   string `json:"interface"`
	RxBytes     uint64 `json:"rx_bytes"`
	TxBytes     uint64 `json:"tx_bytes"`
	RxPackets   uint64 `json:"rx_packets"`
	TxPackets   uint64 `json:"tx_packets"`
	RxErrors    uint64 `json:"rx_errors"`
	TxErrors    uint64 `json:"tx_errors"`
	RxDropped   uint64 `json:"rx_dropped"`
	TxDropped   uint64 `json:"tx_dropped"`
}

// MemoryInfo is parsed from /proc/meminfo.
type MemoryInfo struct {
	Total       uint64 `json:"total_bytes"`
	Free        uint64 `json:"free_bytes"`
	Available   uint64 `json:"available_bytes"`
	Buffers     uint64 `json:"buffers_bytes"`
	Cached      uint64 `json:"cached_bytes"`
	Shared      uint64 `json:"shared_bytes"`
	Dirty       uint64 `json:"dirty_bytes"`
	Writeback   uint64 `json:"writeback_bytes"`
	SwapTotal   uint64 `json:"swap_total_bytes"`
	SwapFree    uint64 `json:"swap_free_bytes"`
}

// ListPerformance returns per-second CPU + memory + disk I/O + per-NIC + OOM events.
func (h *AdminHandler) ListPerformance(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// We do TWO samples of each /proc file with a 1s sleep, so we can
	// compute per-second deltas. Simpler: just return the absolute values
	// and let the frontend show "snapshot" — the frontend already polls
	// every 30s, so per-second deltas will be visible there.
	//
	// For Tier 4 v1 we return the raw values plus delta fields, computed
	// from sample1 + sample2 separated by a 1s sleep on the remote host.

	// nilSafe returns &sshExecResponse{} if v is nil so we can read .Stdout.
	nilSafe := func(v *sshExecResponse, err error) *sshExecResponse {
		if v == nil {
			return &sshExecResponse{}
		}
		return v
	}

	// Sample 1
	cpu1Raw := nilSafe(h.SSHExec(c, cid, "cat /proc/stat 2>/dev/null", 5000))
	memRaw := nilSafe(h.SSHExec(c, cid, "cat /proc/meminfo 2>/dev/null", 5000))
	diskRaw := nilSafe(h.SSHExec(c, cid, "cat /proc/diskstats 2>/dev/null", 5000))
	netRaw := nilSafe(h.SSHExec(c, cid, "cat /proc/net/dev 2>/dev/null", 5000))

	// Sleep 1 second, then sample 2
	_, _ = h.SSHExec(c, cid, "sleep 1", 2000)
	cpu2Raw := nilSafe(h.SSHExec(c, cid, "cat /proc/stat 2>/dev/null", 5000))
	disk2Raw := nilSafe(h.SSHExec(c, cid, "cat /proc/diskstats 2>/dev/null", 5000))
	net2Raw := nilSafe(h.SSHExec(c, cid, "cat /proc/net/dev 2>/dev/null", 5000))

	// OOM events: dmesg | grep -i 'out of memory' | tail -10
	oomRaw := nilSafe(h.SSHExec(c, cid, "dmesg 2>/dev/null | grep -i 'out of memory' | tail -10", 5000))

	cpus := computeCPUDeltas(cpu1Raw.Stdout, cpu2Raw.Stdout)
	mem := parseMeminfo(memRaw.Stdout)
	disks := computeDiskDeltas(diskRaw.Stdout, disk2Raw.Stdout)
	nets := computeNetDeltas(netRaw.Stdout, net2Raw.Stdout)
	ooms := splitLines(oomRaw.Stdout)

	c.JSON(200, gin.H{
		"cpu":      cpus,
		"memory":   mem,
		"disks":    disks,
		"net":      nets,
		"oom_events": ooms,
	})
}

// parseProcStat parses /proc/stat, returning per-CPU stats.
// We capture only the "cpuN" lines (not "cpu" which is aggregate).
func parseProcStat(s string) map[string]cpuTimes {
	out := map[string]cpuTimes{}
	for _, line := range splitLines(s) {
		if !strings.HasPrefix(line, "cpu") || strings.HasPrefix(line, "cpu ") || len(line) < 4 || line[3] == ' ' {
			continue
		}
		// Match "cpu0 ..." etc
		if line[3] < '0' || line[3] > '9' {
			continue
		}
		// Format: cpuN user nice system idle iowait irq softirq steal guest guest_nice
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		name := fields[0]
		// jiffies
		get := func(idx int) float64 {
			v, _ := strconv.ParseFloat(fields[idx], 64)
			return v
		}
		out[name] = cpuTimes{
			user: get(1),
			nice: get(2),
			sys:  get(3),
			idle: get(4),
			iow:  get(5),
			irq:  get(6),
			sirq: get(7),
			steal: get(8),
		}
	}
	return out
}

type cpuTimes struct {
	user, nice, sys, idle, iow, irq, sirq, steal float64
}

// total returns the sum of all jiffies.
func (c cpuTimes) total() float64 {
	return c.user + c.nice + c.sys + c.idle + c.iow + c.irq + c.sirq + c.steal
}

// computeCPUDeltas returns per-core %CPU by diffing two /proc/stat samples.
// If the second sample can't be parsed, returns empty slice.
func computeCPUDeltas(sample1, sample2 string) []CPUCore {
	t1 := parseProcStat(sample1)
	t2 := parseProcStat(sample2)
	out := []CPUCore{}
	for name, c1 := range t1 {
		c2, ok := t2[name]
		if !ok {
			continue
		}
		dTotal := c2.total() - c1.total()
		if dTotal <= 0 {
			continue
		}
		pct := func(a, b float64) float64 {
			return (b - a) / dTotal * 100.0
		}
		out = append(out, CPUCore{
			Core:  name,
			User:  pct(c1.user, c2.user),
			Sys:   pct(c1.sys, c2.sys),
			IO:    pct(c1.iow, c2.iow),
			Idle:  pct(c1.idle, c2.idle),
			Steal: pct(c1.steal, c2.steal),
		})
	}
	return out
}

// parseMeminfo parses /proc/meminfo.
//
// Example: "MemTotal:       16384524 kB"
func parseMeminfo(s string) MemoryInfo {
	out := MemoryInfo{}
	for _, line := range splitLines(s) {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		valStr := strings.TrimSpace(line[idx+1:])
		// value is "<num> kB"
		fields := strings.Fields(valStr)
		if len(fields) < 1 {
			continue
		}
		v, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		// Convert kB to bytes
		v = v * 1024
		switch key {
		case "MemTotal":
			out.Total = v
		case "MemFree":
			out.Free = v
		case "MemAvailable":
			out.Available = v
		case "Buffers":
			out.Buffers = v
		case "Cached":
			out.Cached = v
		case "Shmem":
			out.Shared = v
		case "Dirty":
			out.Dirty = v
		case "Writeback":
			out.Writeback = v
		case "SwapTotal":
			out.SwapTotal = v
		case "SwapFree":
			out.SwapFree = v
		}
	}
	return out
}

// parseDiskstats parses /proc/diskstats.
//
// Format (whitespace-separated):
//   major minor name reads_completed reads_merged sectors_read time_reading
//   writes_completed writes_merged sectors_written time_writing [etc]
func parseDiskstats(s string) map[string]diskRaw {
	out := map[string]diskRaw{}
	for _, line := range splitLines(s) {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		out[name] = diskRaw{
			reads:      parseU64(fields[3]),
			readSec:    parseU64(fields[5]),
			readTime:   parseU64(fields[6]),
			writes:     parseU64(fields[7]),
			writeSec:   parseU64(fields[9]),
			writeTime:  parseU64(fields[10]),
		}
	}
	return out
}

type diskRaw struct {
	reads, readSec, readTime, writes, writeSec, writeTime uint64
}

// computeDiskDeltas returns per-disk I/O rate (per-second) by diffing two samples.
func computeDiskDeltas(s1, s2 string) []DiskIO {
	t1 := parseDiskstats(s1)
	t2 := parseDiskstats(s2)
	out := []DiskIO{}
	for name, a := range t1 {
		b, ok := t2[name]
		if !ok {
			continue
		}
		out = append(out, DiskIO{
			Device:       name,
			Reads:        b.reads - a.reads,
			Writes:       b.writes - a.writes,
			ReadSectors:  b.readSec - a.readSec,
			WriteSectors: b.writeSec - a.writeSec,
			ReadTimeMs:   b.readTime - a.readTime,
			WriteTimeMs:  b.writeTime - a.writeTime,
		})
	}
	return out
}

// parseNetDev parses /proc/net/dev.
//
// Format: "  iface: rx_bytes rx_packets ... tx_bytes tx_packets ... lo: ..."
func parseNetDev(s string) map[string]netRaw {
	out := map[string]netRaw{}
	for _, line := range splitLines(s) {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		fields := strings.Fields(line[idx+1:])
		if len(fields) < 16 {
			continue
		}
		out[name] = netRaw{
			rxBytes:   parseU64(fields[0]),
			rxPackets: parseU64(fields[1]),
			rxErr:     parseU64(fields[2]),
			rxDrop:    parseU64(fields[3]),
			txBytes:   parseU64(fields[8]),
			txPackets: parseU64(fields[9]),
			txErr:     parseU64(fields[10]),
			txDrop:    parseU64(fields[11]),
		}
	}
	return out
}

type netRaw struct {
	rxBytes, rxPackets, rxErr, rxDrop, txBytes, txPackets, txErr, txDrop uint64
}

// computeNetDeltas returns per-NIC throughput (per-second) by diffing two samples.
func computeNetDeltas(s1, s2 string) []NetIO {
	t1 := parseNetDev(s1)
	t2 := parseNetDev(s2)
	out := []NetIO{}
	for name, a := range t1 {
		b, ok := t2[name]
		if !ok {
			continue
		}
		out = append(out, NetIO{
			Interface: name,
			RxBytes:   b.rxBytes - a.rxBytes,
			TxBytes:   b.txBytes - a.txBytes,
			RxPackets: b.rxPackets - a.rxPackets,
			TxPackets: b.txPackets - a.txPackets,
			RxErrors:  b.rxErr - a.rxErr,
			TxErrors:  b.txErr - a.txErr,
			RxDropped: b.rxDrop - a.rxDrop,
			TxDropped: b.txDrop - a.txDrop,
		})
	}
	return out
}

func parseU64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// regex to keep imports used
var _ = regexp.MustCompile
