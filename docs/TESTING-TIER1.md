# StackWatch — Tier 1 Testing Guide (Proxmox)

**Last updated:** 2026-08-19
**Status:** Tier 1 backend and frontend deployed on `.115`; fresh live verification is required before every release.

## Scope

Tier 1 covers the Proxmox replacement surface:

- Host registration, listing, connection test, and tenant isolation
- Cluster nodes, status, info, resources, VMs, and LXC
- VM/LXC configuration, lifecycle actions, and deletion
- Storage pools, content, upload/download routes, and deletion
- Network interfaces and firewall rules/IP sets
- Disks and ZFS
- Proxmox users and API tokens
- Cluster/node tasks
- Pools
- Backup jobs and run-now
- Certificates and ACME accounts/plugins/challenge metadata
- The deployed Proxmox workspace frontend and static assets

The live verifier uses a real registered Proxmox host but performs only **read operations and deliberately invalid write requests**. It does not create or delete real VMs, containers, storage, network interfaces, firewall rules, users, tokens, pools, backups, or certificates.

## Live URLs

- API: `http://192.168.0.115:8080`
- Web UI: `http://192.168.0.115:8090`
- Health: `http://192.168.0.115:8080/health`

## Manual test flow

### 1. Login

```bash
curl -sS -X POST http://192.168.0.115:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"YOUR_EMAIL","password":"YOUR_PASSWORD"}'
```

Save the returned JWT as `TOKEN`. Never place a real password or Proxmox API token in source control or documentation.

```bash
TOKEN='paste-jwt-here'
AUTH="Authorization: Bearer $TOKEN"
```

### 2. List registered Proxmox hosts

```bash
curl -sS -H "$AUTH" http://192.168.0.115:8080/api/v1/proxmox/hosts
```

Expected: a tenant-scoped `hosts` array. Proxmox API secrets must not appear in the response.

Set the host ID returned by the API:

```bash
HID='host-uuid-here'
BASE="http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID"
```

### 3. Check host and cluster data

```bash
curl -sS -H "$AUTH" "$BASE"
curl -sS -H "$AUTH" "$BASE/test"
curl -sS -H "$AUTH" "$BASE/nodes"
curl -sS -H "$AUTH" "$BASE/cluster/status"
curl -sS -H "$AUTH" "$BASE/cluster/info"
curl -sS -H "$AUTH" "$BASE/cluster/resources"
curl -sS -H "$AUTH" "$BASE/vms"
```

Expected: HTTP 200 JSON. On older Proxmox versions, unsupported native operations return HTTP 200 with `supported:false`, not an unexplained 500.

### 4. Check VM/LXC data

Use a real node name and VMID from the previous response:

```bash
NODE='router'
VMID='100'
CTID='101'

curl -sS -H "$AUTH" "$BASE/nodes/$NODE/qemu/$VMID"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/lxc/$CTID"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/qemu/$VMID/status/current"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/lxc/$CTID/status/current"
```

### 5. Check storage, network, firewall, and disks

```bash
curl -sS -H "$AUTH" "$BASE/storage"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/storage/local/content"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/network"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/firewall/rules"
curl -sS -H "$AUTH" "$BASE/firewall/ipsets"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/disks/list"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/disks/zfs"
```

### 6. Check users, tokens, tasks, pools, and backups

```bash
curl -sS -H "$AUTH" "$BASE/access/users"
curl -sS -H "$AUTH" "$BASE/access/users/root@pam"
curl -sS -H "$AUTH" "$BASE/access/users/root@pam/token"
curl -sS -H "$AUTH" "$BASE/cluster/tasks"
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/tasks"
curl -sS -H "$AUTH" "$BASE/pools"
curl -sS -H "$AUTH" "$BASE/cluster/backup"
```

### 7. Check certificates and ACME

```bash
curl -sS -H "$AUTH" "$BASE/nodes/$NODE/certificates"
curl -sS -H "$AUTH" "$BASE/cluster/acme/account"
curl -sS -H "$AUTH" "$BASE/cluster/acme/plugins"
curl -sS -H "$AUTH" "$BASE/cluster/acme/challenge-schema"
curl -sS -H "$AUTH" "$BASE/cluster/acme/directories"
curl -sS -H "$AUTH" "$BASE/cluster/acme/info"
```

For an absent ACME account, the expected result is HTTP 404. On older PVE, an account lookup may otherwise return a misleading 403/500; StackWatch normalizes the absent-resource case.

## Safe validation of write routes

The automated verifier sends malformed JSON to every Tier 1 POST/PUT route. Expected behavior is HTTP 400/404/422 before mutation. It also probes fake delete targets:

- Missing resources → HTTP 404
- Invalid identifiers or malformed parameters → HTTP 400
- Permission failures → HTTP 403
- Unsupported old-PVE operations → HTTP 200 with `supported:false`
- Idempotent deletion on supported Proxmox endpoints → HTTP 200

Do not run destructive create/delete commands against production merely to test routing. Use a disposable Proxmox VM or explicit user approval for real lifecycle round-trips.

## Frontend test

1. Open `http://192.168.0.115:8090`.
2. Sign in with a valid StackWatch account.
3. Open the Proxmox workspace.
4. Confirm the host list loads.
5. Open the host and confirm the tabs load: Overview, VMs, Storage, Network, Firewall, Users, and Operations.
6. Confirm tables show real Proxmox data or an explicit empty state.
7. Confirm unsupported old-PVE capabilities show a clear unsupported state rather than a blank page.
8. Refresh the page and confirm the SPA remains usable.

## Automated verifier

Run from Windows:

```bash
python "C:/Users/himan/AppData/Local/Temp/hermes-tier1-full-live.py"
```

The verifier performs one complete pass with one JWT. To meet the release gate, run it four times, restarting the API between passes only to clear the in-memory login limiter. The verifier must report:

```text
FULL PASS RESULT: 78 passed, 0 failed, total=78
```

The separate original read/frontend verifier is also retained at:

```text
C:/Users/himan/AppData/Local/Temp/hermes-tier1-4pass-live.py
```

It must reuse one JWT across its four passes; logging in once per pass can trigger the production login limiter and is not valid evidence.

## Known platform constraints

- The current API login limiter is in-memory. Restarting the API clears its counters; a future Redis-backed limiter will remove this verifier coordination requirement.
- The target Proxmox host is an older PVE release. Some native endpoints are unsupported and correctly report `supported:false`.
- Real destructive lifecycle tests require a disposable VM/CT and must be run separately from the safe release verifier.
