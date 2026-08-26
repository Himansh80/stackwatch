# Tier 14 Phase 14.10 Plan

## Frontend (5 new pages + 1 upgraded)

### ProxmoxTasksPanel.tsx (~280 LOC)
- Live log with auto-refresh
- Filters (node, status, type)
- Stop task action

### ProxmoxBackupJobs.tsx (~280 LOC)
- CRUD + run-now
- Form: id, schedule, storage, retention
- Run-now POST

### ProxmoxReplication.tsx (~220 LOC)
- CRUD list

### ProxmoxHaGroups.tsx (~220 LOC)
- CRUD list

### ProxmoxCertificates.tsx (~220 LOC)
- List + view + renew

### ProxmoxSdnZones.tsx (upgrade from 95→~215 LOC)
- Add create form + delete buttons
- Convert from read-only to full CRUD

## Backend
ZERO new endpoints.

## Routes added
- /proxmox/tasks
- /proxmox/backup
- /proxmox/replication
- /proxmox/ha
- /proxmox/certificates

## Order
1. Tasks
2. Backup
3. Replication
4. HA
5. Certificates
6. Upgrade SDN
7. Routes
8. Sidebar
9. tsc + build + deploy
10. Commit
11. **Deliver end-of-Tier-14 MEGA-TEST**
