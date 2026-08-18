# Tasks: Tier 4 — Server Admin (revised — SSH-based, no new binary)

## 1. Web-terminal: SSH exec endpoint
- [ ] 1.1 Add POST /api/v1/admin/exec to cmd/web-terminal/main.go
- [ ] 1.2 Body: {connection_id, command, timeout_ms (default 30000)}
- [ ] 1.3 Returns: {stdout, stderr, exit_code, duration_ms}
- [ ] 1.4 Reuse openSSH helper but skip RequestPty (non-interactive session)

## 2. Handler layer (internal/handler/admin_*.go)
- [ ] 2.1 admin_proxy.go — common helper to forward to web-terminal /admin/exec
- [ ] 2.2 admin_services.go — list systemctl units + 5 actions
- [ ] 2.3 admin_storage.go — lsblk -J + findmnt -J + smartctl (if available)
- [ ] 2.4 admin_network.go — ip -j addr + ip -j route + /etc/resolv.conf
- [ ] 2.5 admin_processes.go — ps aux parsed top N
- [ ] 2.6 admin_updates.go — apt list --upgradable or dnf check-update
- [ ] 2.7 admin_logs.go — journalctl + tail /var/log
- [ ] 2.8 admin_users.go — getent passwd + /etc/sudoers.d
- [ ] 2.9 admin_timers.go — systemctl list-timers + /etc/cron.d
- [ ] 2.10 admin_performance.go — /proc/stat per-core + /proc/meminfo + /proc/net/dev

## 3. Routes
- [ ] 3.1 Register 30+ routes in cmd/api-gateway/routes.go

## 4. Build + restart
- [ ] 4.1 go build (api-gateway + web-terminal)
- [ ] 4.2 Restart web-terminal (port :8085)
- [ ] 4.3 Restart api-gateway (port :8080)
- [ ] 4.4 Verify both /health

## 5. Verifier
- [ ] 5.1 Write hermes-verify-tier4-full-2026-08-17.py (≥30 checks)
- [ ] 5.2 Run 5 back-to-back; verify all PASS
- [ ] 5.3 Cross-check T0/1/2/3 still clean

## 6. Frontend (deferred to Tier 12 polish, skip for v1)

## Done criteria
- [ ] POST /api/v1/admin/exec returns {stdout, stderr, exit_code} for arbitrary commands
- [ ] 30+ admin routes registered and respond with real data
- [ ] hermes-verify-tier4-full-2026-08-17.py: ≥30 checks, 5/5 back-to-back PASS
- [ ] T0+T1+T2+T3 cross-check: still clean
- [ ] All commits in working tree (or pushed if user enables)