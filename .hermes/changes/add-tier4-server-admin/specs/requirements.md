# Requirements: Tier 4 — Server Admin

## Functional

### F1. Services (S1)
- List systemd units: name, state (active/inactive/failed), sub-state,
  description, load state
- Start, stop, restart, enable, disable any unit
- Filter by state (active/inactive/failed/all)

### F2. Storage (S2)
- List block devices: name, size, model, serial, type (disk/part/rom/lvm/raid)
- List mounts: device, mountpoint, fstype, used, total, %
- SMART health summary: overall_passed, temperature, power_on_hours
- List ZFS pools: name, size, allocated, free, frag, health (if zpool present)

### F3. Network (S3)
- List interfaces: name, MAC, IPs (v4/v6), MTU, state, speed
- List routes: dest, gateway, iface, metric
- List DNS: resolv.conf nameservers, search domains
- List firewall rules (iptables-save parse): chain, target, proto, src, dst

### F4. Processes (S4)
- Top N processes by cpu and memory: pid, user, cpu%, mem%, command
- Sort, filter by user/name
- No kill endpoint in this tier (Tier 5+)

### F5. Updates (S5)
- Available package updates: package name, current version, new version, repo
- Security-flagged updates separate
- No install endpoint in this tier (just check, not run)

### F6. Logs (S6)
- Read last N journald lines: unit, priority, message, timestamp
- Filter by unit name, priority (emerg..debug), since (1h ago)
- Read /var/log/<file> tail (most recent 100 lines)

### F7. Users (S7)
- List system users: name, uid, gid, home, shell, last_login
- List sudoers entries (from /etc/sudoers.d)
- List SSH authorized_keys for each user (just the keys, not the private)

### F8. Timers (S8)
- List systemd timers: name, next_run, last_run, schedule, unit_triggered
- List cron jobs: from /etc/cron.d/, /var/spool/cron/<user>

### F9. Performance (S9)
- Per-second CPU breakdown: user%, system%, iowait%, steal%, idle%
- Memory breakdown: cached, buffers, dirty, writeback, swap_used
- Disk I/O per device: read_bytes/s, write_bytes/s, iops
- Per-NIC: rx_bytes/s, tx_bytes/s, rx_packets/s, tx_packets/s, errors
- OOM events from kernel ring buffer (last 10 events)

## Non-functional

### NF1. Performance
- All admin endpoints respond in <500ms p95
- Process list cached for 5 seconds (refresh on demand)
- Service list cached for 30 seconds
- Storage list cached for 60 seconds (expensive)

### NF2. Security
- All endpoints require JWT auth
- Per-tenant isolation (tenant_id on every row)
- No actions that mutate state without explicit user intent
- Only safe GET-equivalent actions: services/start|stop|restart|enable|disable are user-initiated

### NF3. Robustness
- Missing tools (e.g. zpool not installed) → empty array, not 500
- Permission denied on read → return partial with warning, not 500
- All errors wrapped via `kernel.RespondError`

## Done criteria
- All 30+ routes registered and return 200 (or appropriate auth/permission error)
- All 9 admin pages render in frontend
- One full verifier `hermes-verify-tier4-full-2026-08-17.py` with 4+ back-to-back PASS
- Real data verified against the .115 server (and at least one other server)
- Migration applied, no breakage to Tiers 0/1/2/3