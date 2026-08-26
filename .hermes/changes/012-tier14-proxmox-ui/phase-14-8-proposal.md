# Tier 14 Phase 14.8 — SDN + Node Firewall + IPSets + Aliases

## Why this phase

Phase 14.3 + 14.4 added VM-level firewall rules. Proxmox's primary firewall lives at the **node** level (and datacenter level). Phase 14.8 adds the node firewall viewer + IPSets + Aliases + SDN zones.

## Goals

1. **Node firewall page** at `/proxmox/nodes/:hostId/:node/firewall` — view + add/edit/delete rules
2. **IPSet manager** at `/proxmox/ipsets` — list + create + delete + entries
3. **Alias manager** at `/proxmox/aliases` — list + create + delete
4. **SDN zones** at `/proxmox/sdn/zones` — list + create (read-only for now, full CRUD = Phase 14.10)

## Backend

ZERO new endpoints — all use Tier 1:
- `GET /proxmox/hosts/:id/nodes/:node/firewall/rules` — list rules
- `POST /proxmox/hosts/:id/nodes/:node/firewall/rules` — add
- `PUT /proxmox/hosts/:id/nodes/:node/firewall/rules/:pos` — update
- `DELETE /proxmox/hosts/:id/nodes/:node/firewall/rules/:pos` — delete
- `GET /proxmox/ipsets` — list IPSets
- `POST /proxmox/ipsets` — create
- `DELETE /proxmox/ipsets/:name` — delete
- `POST /proxmox/ipsets/:name` — add CIDR
- `DELETE /proxmox/ipsets/:name/:cidr` — remove CIDR
- `GET /proxmox/aliases` — list
- `POST /proxmox/aliases` — create
- `DELETE /proxmox/aliases/:name` — delete
- `GET /proxmox/sdn/zones` — list zones

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxNodeFirewall.tsx` | 280 | Node firewall rules CRUD |
| `ProxmoxIpsetManager.tsx` | 250 | IPSet list + CIDR entries |
| `ProxmoxAliasManager.tsx` | 220 | Alias list + add/delete |
| `ProxmoxSdnZones.tsx` | 200 | SDN zones read-only list |
| `proxmox-network.css` | 180 | Shared styles for network sections |

## Routes

- `/proxmox/nodes/:hostId/:node/firewall` — node firewall
- `/proxmox/ipsets` — IPSets
- `/proxmox/aliases` — aliases
- `/proxmox/sdn/zones` — SDN zones

## Acceptance criteria

1. Node firewall lists + can add/edit/delete rules
2. IPSets list + can add entries
3. Aliases list + can add/delete
4. SDN zones list (read-only)
5. Empty states everywhere
6. Mobile-responsive
7. Dark theme + Datadog style

## Modularity discipline

- Every TSX < 400 LOC
- Reuse ProxmoxFirewallRuleDialog from Phase 14.3 (already built)
- Each page is independent