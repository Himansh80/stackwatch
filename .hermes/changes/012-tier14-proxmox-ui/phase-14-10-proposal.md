# Tier 14 Phase 14.10 — Tasks + Backup + Replication + HA + Certs + SDN CRUD

## Why this phase

The final phase of Tier 14 wraps up the Proxmox coverage. Phase 14.10 ships the remaining Proxmox surfaces that aren't already covered: tasks panel, backup jobs, replication, HA groups, certificates + ACME, and full SDN CRUD.

## Goals

1. **Tasks panel** at `/proxmox/tasks` — live cluster task log (filter by node, status, type)
2. **Backup jobs** at `/proxmox/backup` — list + create + run-now + delete
3. **Replication** at `/proxmox/replication` — list + create + delete
4. **HA groups** at `/proxmox/ha` — list + create + delete
5. **Certificates + ACME** at `/proxmox/certificates` — list + view + renew
6. **SDN full CRUD** — turn Phase 14.8's read-only into create/delete

## Backend

ZERO new endpoints — all use Tier 1.

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxTasksPanel.tsx` | 280 | Live task log with filters |
| `ProxmoxBackupJobs.tsx` | 280 | Backup CRUD + run-now |
| `ProxmoxReplication.tsx` | 220 | Replication list + CRUD |
| `ProxmoxHaGroups.tsx` | 220 | HA list + CRUD |
| `ProxmoxCertificates.tsx` | 220 | Cert list + renew + view |
| Upgrade `ProxmoxSdnZones.tsx` | +120 | Add create/delete |

## Routes (added in Phase 14.10)

- `/proxmox/tasks` — tasks panel
- `/proxmox/backup` — backup jobs
- `/proxmox/replication` — replication
- `/proxmox/ha` — HA groups
- `/proxmox/certificates` — certs + ACME

## Acceptance criteria

1. Tasks panel: live updates, filter by node/status/type
2. Backup jobs: CRUD + run-now
3. Replication: CRUD
4. HA: CRUD
5. Certificates: list + view + renew
6. SDN zones: full CRUD (was read-only in 14.8)
7. Empty states everywhere
8. Mobile-responsive
9. Dark theme + Datadog style
10. **TIER 14 COMPLETE** — end-of-tier mega-test (delivered separately)
