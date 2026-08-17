# StackWatch — Tier 1 Testing Guide (Proxmox)

**Last updated:** 2026-08-17
**For:** Himan (the user)
**Date created:** 2026-08-16
**Status:** Tier 1 (Proxmox full replacement) shipped. **14/14 live tests pass** on real Proxmox at .107.

---

## What's live

API endpoints added:
- `POST   /api/v1/proxmox/hosts` — register a Proxmox host
- `GET    /api/v1/proxmox/hosts` — list hosts for your tenant
- `DELETE /api/v1/proxmox/hosts/:id` — remove a host
- `GET    /api/v1/proxmox/hosts/:id/test` — test connection
- `GET    /api/v1/proxmox/hosts/:id/nodes` — list cluster nodes
- `GET    /api/v1/proxmox/hosts/:id/vms` — list VMs + LXC
- `GET    /api/v1/proxmox/hosts/:id/storage?node=X` — list storage pools

All endpoints are authenticated (JWT), tenant-scoped, and tested with your real Proxmox at `https://192.168.0.107:8006`.

---

## How to test (live against real .107)

### 1. Sign up

```bash
curl -X POST http://192.168.0.115:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"your@email.com","password":"StrongPassword2026","full_name":"Your Name","tenant_name":"Your Org"}'
```

### 2. Save the token

```bash
TOKEN="paste-token-here"
```

### 3. Add your real Proxmox host

```bash
curl -X POST http://192.168.0.115:8080/api/v1/proxmox/hosts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "lab-proxmox",
    "base_url": "https://192.168.0.107:8006",
    "api_token": "PVEAPIToken=root@pam!monitor=613a1a19-718c-4534-93ad-9efb679ffb58",
    "verify_tls": false
  }'
```

Expected: 201 with `status: "online"` (it pings the host before persisting).

### 4. List your hosts

```bash
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts
```

### 5. Test the connection

```bash
# Copy the id from step 4
HID="host-id-here"
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID/test
```

Expected: `{"ok": true}`

### 6. Discover VMs (real data from .107)

```bash
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID/vms
```

Expected: 13 VMs/LXC including pf-sense (qemu), adguard (lxc), nginxproxymanager (lxc), esphome (lxc), homeassistant (qemu), iot-server (qemu), termix (lxc), portfolio (qemu), mail.bluecubeautomations.com (lxc).

### 7. Discover nodes

```bash
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID/nodes
```

Expected: 1 node named "router", status "online", 4 CPU, 16 GB RAM, 65 GB disk.

### 8. Discover storage

```bash
curl -H "Authorization: Bearer $TOKEN" "http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID/storage?node=router"
```

Expected: 2 storage pools — `local` (dir) and `local-lvm` (lvmthin), with used/total/avail bytes.

### 9. Test isolation (different tenant sees nothing)

```bash
# Sign up a SECOND tenant
curl -X POST http://192.168.0.115:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"other@x.com","password":"StrongPassword2026","full_name":"Other","tenant_name":"Other Org"}'

TOKEN2="paste-second-token"
curl -H "Authorization: Bearer $TOKEN2" http://192.168.0.115:8080/api/v1/proxmox/hosts
```

Expected: `{"hosts": [], "total": 0}` — tenant isolation works.

### 10. Verify no token leaks

```bash
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts
```

The JSON should NEVER contain `PVEAPIToken=...`. Only `id`, `name`, `base_url`, `verify_tls`, `status`.

### 11. Delete the host

```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/proxmox/hosts/$HID
```

---

## What you CAN'T do yet (Tier 1 next steps)

- Create VMs (`POST /proxmox/hosts/:id/vms`)
- Start/stop/reboot VMs (`POST /proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/start`)
- LXC lifecycle
- Storage create/delete
- Networking (bridges, VLANs, bonds, firewall)
- Backup + restore
- Templates + cloud-init
- HA + cluster
- Per-VM monitoring stats (already have data, just need UI)

These are Tier 1.2 – 1.9 in the master plan.

---

## Automated verifier

Run from Windows:
```bash
python "C:/Users/himan/AppData/Local/Temp/hermes-verify-tier1-proxmox-2026-08-16.py"
```

**Latest result:** 14/14 PASS (verified 4 consecutive runs).

---

## What I need from you

After you've tested, tell me one of:

1. **"next"** — move to Tier 1.2 (VM lifecycle: create/start/stop/reboot/delete)
2. **"fix X"** — describe the issue
3. **"change Y"** — describe the change

I'll fix or move on. No claim of "done" until you say it works.

---

## Known gaps (Tier 1 partial — addressed in Tier 1.2+)

- **API tokens stored as plaintext in DB** — Tier 9 will encrypt at rest (AES-256-GCM)
- **No per-VM UI in dashboard yet** — Tier 1.10 frontend (next after backend)
- **No auto-refresh / WebSocket** — Tier D1 ships this
- **No VM console (noVNC)** — Tier 3
