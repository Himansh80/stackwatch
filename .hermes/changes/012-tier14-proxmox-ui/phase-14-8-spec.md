# Tier 14 Phase 14.8 — Specification: SDN + Node Firewall + IPSets + Aliases

## 1. URL pattern
- `/proxmox/nodes/:hostId/:node/firewall` — node firewall
- `/proxmox/ipsets` — IPSets (datacenter-scoped)
- `/proxmox/aliases` — aliases (datacenter-scoped)
- `/proxmox/sdn/zones` — SDN zones

## 2. Node Firewall

- Table: Pos | Action | Type | Source | Dest | Proto | Dport | Comment | Enable | Actions
- "+ Add rule" button (reuses ProxmoxFirewallRuleDialog)
- Each row: Edit (opens dialog pre-filled), Delete (confirm)

## 3. IPSets

- Table: Name | Comment | # CIDRs | Actions
- "+ Create IPset" button (form: name, comment)
- Click row → expand to show CIDR list
- "+ Add CIDR" inline
- Delete CIDR per row

## 4. Aliases

- Table: Name | CIDR | Comment | Actions
- "+ Create alias" button (form: name, cidr, comment)
- Delete per row

## 5. SDN Zones

- Table: Name | Type | Bridge | DHCP | State
- (read-only for Phase 14.8 — full CRUD is Phase 14.10)
- "+ Create zone" button (deferred)

## 6. Acceptance criteria

1. Node firewall rules add/edit/delete
2. IPSet + CIDR create/delete
3. Alias create/delete
4. SDN zones display (read-only)
5. Empty states
6. Mobile-responsive
