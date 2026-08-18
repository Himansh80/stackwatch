# Scenarios: Tier 4 — Server Admin (acceptance tests)

## S1. Services
- Given a server with at least one systemd unit (e.g. sshd)
- When GET /api/v1/admin/services?server_id=X
- Then response includes sshd with state=active and a "actions" array listing
  start/stop/restart/enable/disable
- When POST /api/v1/admin/services/<id>/<action>
- Then action is attempted; success returns 200, failure returns 500 with error

## S2. Storage
- Given a server with at least one block device and one mount
- When GET /api/v1/admin/storage?server_id=X
- Then response includes at least one block_device (sda or vda), at least one
  mount (/, /boot), and SMART health data

## S3. Network
- Given a server with at least one network interface
- When GET /api/v1/admin/network?server_id=X
- Then response includes at least one interface with IPs and at least one
  default route

## S4. Processes
- Given a server with running processes
- When GET /api/v1/admin/processes?server_id=X&sort=cpu&limit=10
- Then response includes 10 processes sorted by cpu descending
- Each row has pid, user, cpu_pct, mem_pct, command

## S5. Updates
- Given a server (Ubuntu or RHEL family)
- When GET /api/v1/admin/updates?server_id=X
- Then response includes array of updates with name, current_version, new_version
- Empty array is acceptable if no updates available (or apt/dnf not present)

## S6. Logs
- Given a server with systemd and /var/log/messages or journald
- When GET /api/v1/admin/logs?server_id=X&unit=ssh&limit=20
- Then response includes array of log entries with timestamp, priority, message
- Empty array is acceptable if no matching logs

## S7. Users
- Given a server with system users
- When GET /api/v1/admin/users?server_id=X
- Then response includes root user (uid=0) and any other system users
- sudoers section may be empty (no sudoers.d files is OK)

## S8. Timers
- Given a server with systemd timers
- When GET /api/v1/admin/timers?server_id=X
- Then response includes any active timers with schedule, last_run, next_run
- cron section may be empty

## S9. Performance
- Given a server with CPU, memory, disk, network
- When GET /api/v1/admin/performance?server_id=X
- Then response includes cpu_per_core array, memory_breakdown, disk_io_per_dev,
  net_per_nic, oom_events (empty array OK if no OOM)

## Cross-tier
- All endpoints return 401 without JWT
- All endpoints return 403 for cross-tenant access (when server belongs to another tenant)
- Verifier runs 4+ times with zero flake