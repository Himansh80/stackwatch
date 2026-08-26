# Tier 14 Phase 14.2 Specification: VM Detail Page

## 1. Goal

Click a VM row in `/proxmox-vms` → open a detail page showing that VM's hardware, network, status, actions.

## 2. URL pattern

```
/proxmox-vms/:hostId/:node/:vmid
```

Where:
- `hostId` = Proxmox host UUID (from `/api/v1/proxmox/hosts`)
- `node` = Proxmox node name (e.g. `router`)
- `vmid` = Proxmox VMID (number)

Tab is a URL hash: `#summary` `#hardware` `#network` `#console` `#snapshots` `#firewall`

Default tab: `#summary` if no hash.

## 3. Layout (top to bottom)

```
┌─────────────────────────────────────────────────────────────┐
│  Proxmox › router › VM 100                                   │
│                                                              │
│  ubuntu-test                          [RUNNING]              │
│  ID 100 · Type qemu · Node router · Uptime 3d 4h             │
│                                                              │
│  [▶ Start] [⏹ Shutdown] [↻ Reboot] [⇄ Migrate] [🗑 Delete]  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ [Summary] [Hardware] [Network] [Console] [Snapshots] [Firewall] │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ (active tab content)                                         │
└─────────────────────────────────────────────────────────────┘
```

## 4. Header (always visible)

- **Breadcrumb:** Proxmox > {node} > VM {vmid}
- **VM name** (large, bold)
- **Status pill** (green/gray/amber/red)
- **ID/Type/Node/Uptime row** (small text)
- **Action buttons** (Start / Shutdown / Reboot / Migrate / Delete)

## 5. Tab content

### 5.1 Summary tab (default)
- VM ID + Type
- Node + Cluster
- Status + Uptime
- CPU (current allocation %)
- Memory (used / total)
- Disk (used / total)
- Tags (chips)
- Pool
- HA state (if configured)

### 5.2 Hardware tab
- Cores (sockets, cores, total)
- Memory MB
- Disk size GB
- BIOS (seabios / ovmf)
- Machine type (q35 / i440fx)
- CPU type (host / kvm64 / x86-64-v2-AES)
- SCSI controller (virtio-scsi-pci / etc.)
- Network model (virtio / e1000)
- Boot order

### 5.3 Network tab
- Table: Interface | MAC | Bridge | IP | MTU | Model
- Source: QEMU agent `network-get-interfaces` (NEW endpoint)
- If agent not running: show hint "QEMU guest agent not running — install qemu-guest-agent inside the VM"

### 5.4 Console tab (Phase 14.2 = placeholder)
- Card: "Console opens in v2.3"
- Button: "Open noVNC" (disabled, with tooltip)

### 5.5 Snapshots tab
- List: Name | Date | VM State | Size
- "Create snapshot" button (deferred)
- Per-row: Delete button (deferred)
- Empty state if no snapshots

### 5.6 Firewall tab
- List: Pos | Action | Source | Dest | Proto | Dport | Comment
- "+ Add rule" (deferred)
- Per-row: Delete (deferred)
- Empty state if no rules

## 6. Backend endpoint needed (NEW)

**`GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/agent/network-get-interfaces`**

Calls Proxmox API: `GET /nodes/{node}/qemu/{vmid}/agent/network-get-interfaces`

Response:
```json
{
  "interfaces": [
    {
      "name": "eth0",
      "mac": "bc:24:11:8d:5b:9c",
      "ip-addresses": [
        { "ip-address": "192.168.0.117", "ip-address-type": "ipv4", "prefix": 24 }
      ],
      "mtu": 1500,
      "stats": { "rx-bytes": 12345, "tx-bytes": 67890 }
    }
  ]
}
```

If agent not running, Proxmox returns 500 with `QEMU guest agent is not running`. We map to HTTP 200 with `{ "interfaces": null, "agent_running": false }` so the frontend can show a hint.

## 7. State management

- `hostId`, `node`, `vmid` from URL params
- `tab` from URL hash
- VM data fetched on mount + every 5s (status, uptime)
- Config fetched once on mount + on tab change
- Hardware / Network / Console fetched lazily per tab

## 8. Empty / error states

- VM not found → red banner "VM 100 not found on router"
- Host offline → yellow banner "Proxmox host unreachable — retry"
- QEMU agent not running → info card on Network tab
- No snapshots → empty state with "+ Create" disabled

## 9. Acceptance criteria

1. Click VM row → URL becomes `/proxmox-vms/{hostId}/{node}/{vmid}#summary`
2. Page shows within 1s
3. Header shows name + status pill + action buttons
4. Click any tab → URL hash updates + content swaps
5. Hardware tab shows cores/memory/disk/BIOS read from API
6. Network tab shows interfaces OR "agent not running" hint
7. Start/Stop/Reboot actions work + show confirm + refresh status
8. Migrate button shows node-picker modal (deferred: button shows "coming in v2.3")
9. Delete shows confirm with name + vmid + "type DELETE to confirm" + redirects to /proxmox-vms
10. Mobile: tabs collapse to dropdown on <768px
11. Auto-refresh every 5s

## 10. Out of scope (Phase 14.3+)

- Real console (xterm.js + noVNC)
- Snapshot create/delete/rollback
- Firewall rule CRUD
- VM create wizard
- Live CPU/RAM/Network charts