# Tier 14 Phase 14.10 — Specification

## 1. Tasks Panel
- Live cluster task log
- Auto-refresh 5s
- Filter: node / status (running/OK/error/stopped) / type
- Table: UPID | Node | Type | Status | Started | Duration | Actions (stop)
- Click row → task log

## 2. Backup Jobs
- List: ID | Schedule | Selection mode | Storage | Keep | Enabled
- "+ New" → form (id, schedule, all/selected, storage, days of week, retention)
- Run-now (POST .../run)
- Delete
- Detail: list past runs

## 3. Replication
- List: ID | Job (VMID) | Target | Schedule | Enabled
- Create form
- Delete

## 4. HA Groups
- List: ID | Resources | State | Comment
- Create form
- Delete

## 5. Certificates
- List: filename | subject | issuer | notBefore | notAfter | SANs
- View detail
- Renew (ACME)
- Delete custom cert

## 6. SDN Full CRUD
- Upgrade Phase 14.8's read-only to add:
- "+ Create zone" → form (zone name, type, bridge)
- Delete per row
