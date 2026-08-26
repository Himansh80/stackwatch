# Tier 14 Phase 14.3 Specification

## 1. Real Console

**Goal:** Browser-based graphical VNC console inside StackWatch.

### 1.1 noVNC client setup
- npm package: `novnc-next` (already supports WSS + auth ticket)
- Connects to `wss://stackwatch.smarthomelab.fun/api/v1/proxmox/hosts/{id}/vnc-ws?node=Y&vmid=Z`
- API gateway proxies to `wss://proxmox-host:8006/api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket`

### 1.2 Authentication
Proxmox VNC websocket requires a ticket. Two options:
- **Option A (chosen):)** Use the API token as the VNC password. Proxmox accepts `PVEAPIToken=USER@REALM!TOKENID=UUID` as the password parameter.
- The api-gateway proxies the VNC connection without exposing the token to the browser.

### 1.3 New backend endpoints
- `POST /api/v1/proxmox/hosts/:id/vnc-ticket` — returns `{ticket, port, node}` (using the host's API token as the ticket)
- `GET /api/v1/proxmox/hosts/:id/vnc-ws?node=Y&vmid=Z` — proxies the VNC WebSocket

### 1.4 Frontend console component
```tsx
<ProxmoxDetailConsole hostId node vmid />
  // on mount:
  // 1. POST /vnc-ticket → get {ticket, port}
  // 2. Initialize noVNC client with:
  //    url: wss://stackwatch/api/v1/proxmox/hosts/{id}/vnc-ws
  //    password: ticket
  // 3. Render <canvas ref={canvasRef} /> inside container
  // 4. On unmount: disconnect client
```

### 1.5 Console UI
- Full-width black canvas
- Top-right toolbar: Reconnect, Send Ctrl+Alt+Del, Fullscreen
- Status indicator: Connected / Disconnected / Connecting / Error
- Loading state: "Connecting to VNC..."

## 2. Snapshot CRUD

### 2.1 Snapshot list
**Endpoint:** `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot`

**Response:**
```json
{
  "snapshots": [
    {
      "name": "pre-upgrade",
      "date": "2026-08-15 10:30:00",
      "vmstate": true,
      "size": 2147483648
    }
  ]
}
```

### 2.2 Create snapshot
**Endpoint:** `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot`
**Body:** `{snapname: string, vmstate: boolean, description?: string}`
**Response:** 202 Accepted with UPID

### 2.3 Delete snapshot
**Endpoint:** `DELETE /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot?snapname=X`
**Response:** 202 Accepted with UPID

### 2.4 Rollback snapshot
**Endpoint:** `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot/:snapname/rollback`
**Response:** 202 Accepted with UPID

### 2.5 UI
- "+ Create snapshot" button (top-right)
- Per-snapshot row: Name | Date | VM State | Size | [Rollback] [Delete]
- Create modal: name (required) + "Include RAM state" checkbox + "Description" textarea
- Delete confirmation: "Delete snapshot 'X'? This cannot be undone."
- Rollback confirmation: "Rollback VM to snapshot 'X'? Current state will be lost."

## 3. Firewall CRUD

### 3.1 List VM rules
**Endpoint:** `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/firewall/rules`

**Response:**
```json
{
  "rules": [
    {
      "pos": 0,
      "type": "in",
      "action": "ACCEPT",
      "enable": 1,
      "source": "192.168.0.0/24",
      "dest": "",
      "proto": "tcp",
      "dport": "22",
      "comment": "SSH from LAN"
    }
  ]
}
```

### 3.2 Add rule
**Endpoint:** `POST .../firewall/rules`
**Body:** `{type, action, enable, source, dest, proto, dport, sport, iface, comment, macro, log}`

### 3.3 Update rule
**Endpoint:** `PUT .../firewall/rules/:pos`
**Body:** same as add

### 3.4 Delete rule
**Endpoint:** `DELETE .../firewall/rules/:pos`

### 3.5 UI
- "+ Add rule" button (top-right)
- Rule row: Pos | Action badge | Source | Dest | Proto | Dport | Comment | [Edit] [Delete]
- Rule form modal: Type (in/out/forward), Action, Source, Dest, Proto, Dport, Sport, Iface, Comment, Macro
- Delete confirmation: "Delete rule #N?"
- Edit: same modal, prefilled

## 4. Migrate dialog

### 4.1 New component: `ProxmoxDetailMigrateDialog`
- Dropdown: target node (other nodes from `/nodes` endpoint)
- Toggle: "Online" (live migration, no downtime) vs "Offline" (shutdown first)
- Execute button → POST /api/v1/.../migrate

### 4.2 Behavior
- Disabled if only one node in cluster
- "Online" requires VM to be running
- "With" local storage flag (checkbox = for move storage too)