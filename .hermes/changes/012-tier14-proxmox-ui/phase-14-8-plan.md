# Tier 14 Phase 14.8 Plan

## Approach
4 new pages + 1 shared CSS. Reuse ProxmoxFirewallRuleDialog from Phase 14.3.

## Frontend (4 new pages + 1 new CSS)

### ProxmoxNodeFirewall.tsx (~280 LOC)
- Reuse ProxmoxFirewallRuleDialog (from 14.3)
- Rules table with action buttons
- "+ Add rule" button
- Delete confirm

### ProxmoxIpsetManager.tsx (~250 LOC)
- Table of IPSets
- Click row → expand CIDR list
- Inline CIDR add form
- IPset + CIDR delete

### ProxmoxAliasManager.tsx (~220 LOC)
- Table of aliases
- Add form (name, CIDR, comment)
- Delete

### ProxmoxSdnZones.tsx (~200 LOC)
- Read-only table
- Phase 14.10 adds full CRUD

### proxmox-network.css (~180 LOC)
- Shared styles for table + form sections

## Order
1. Node firewall
2. IPSets
3. Aliases
4. SDN zones
5. CSS
6. Routes
7. Sidebar links
8. tsc + build + deploy
9. Commit
