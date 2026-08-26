# 🏆 STACKWATCH TIER 14 — END-OF-TIER MEGA-TEST

**Scope:** All 10 phases of Tier 14 (Proxmox full web UI replacement)
**Status:** Ready for your A-to-Z test
**Total test cases:** 80+

---

## Setup

1. **Open browser** to `https://stackwatch.smarthomelab.fun` (or `http://192.168.0.115:8090`)
2. **Hard refresh** (`Ctrl+Shift+R`)
3. **Log in** with your super_admin account
4. **Pick a Proxmox host** with at least 1 VM, 1 LXC, 1 ISO, 1 template

**Browsers:** Desktop (Chrome/Firefox/Edge) + Mobile view (DevTools responsive)

---

## Phase 14.1 — VM List Page (12 tests)

| # | Test | Expected |
|---|------|----------|
| 1.1 | Navigate `/proxmox-vms` | Page loads in <1s |
| 1.2 | Host selector dropdown | Lists your registered Proxmox hosts |
| 1.3 | Host online status | Green dot if reachable |
| 1.4 | KPI strip | 6 cards: Total / Running / Stopped / Paused / Cluster CPU / Cluster RAM |
| 1.5 | KPI animated count-up | Numbers animate from 0 to value on load |
| 1.6 | Search box | Filters table by name/VMID/IP as you type |
| 1.7 | Status filter chips | "All/Running/Stopped/Paused" — click filters table |
| 1.8 | Filter chip counts | Each chip shows its count |
| 1.9 | Desktop table view | Columns: Status / VMID / Type / Name / Node / IP / CPU / RAM / Uptime / Actions |
| 1.10 | Status pill colors | Green=running, slate=stopped, amber=paused |
| 1.11 | Action buttons per row | Start (green), Shutdown (amber), Reboot (blue) |
| 1.12 | Mobile card layout | Resize <768px — table becomes card list |

## Phase 14.2 — VM Detail Page (12 tests)

| # | Test | Expected |
|---|------|----------|
| 2.1 | Click VM row | URL becomes `/proxmox-vms/{hostId}/{node}/{vmid}#summary` |
| 2.2 | Breadcrumb | "Proxmox › {node} › VM {vmid}" |
| 2.3 | VM name + status pill | Large name, colored pill below |
| 2.4 | Meta row | ID / Type / Node / Uptime / Host |
| 2.5 | 5 action buttons | Start / Shutdown / Reboot / Migrate / Delete |
| 2.6 | Summary tab default | Identity / Resource usage / Tags cards |
| 2.7 | CPU/MEM/DISK bars | Green ≤65%, amber ≤85%, red >85% |
| 2.8 | Tags display | Chips for each tag, "No tags" muted if empty |
| 2.9 | Hardware tab | CPU / Memory / Firmware / Boot order / Network cards |
| 2.10 | Network tab | Shows interfaces OR "agent not running" hint |
| 2.11 | Console tab | "ships in v2.3" placeholder, disabled "Open noVNC" |
| 2.12 | Snapshots tab | Empty state OR list of snapshots |

## Phase 14.3 — VM Console + Snapshot/Firewall CRUD + Migrate (15 tests)

| # | Test | Expected |
|---|------|----------|
| 3.1 | Click Console tab | noVNC connection attempts (or error if no VM agent) |
| 3.2 | Console toolbar | Reconnect / Ctrl+Alt+Del / Fullscreen buttons |
| 3.3 | Console status indicator | "connecting…" then "connected" (or "error" if VM is off) |
| 3.4 | Click "+ Create snapshot" | Modal: name + description + RAM state checkbox |
| 3.5 | Create snapshot | List refreshes with new row |
| 3.6 | Click snapshot Rollback | Confirm modal → task starts |
| 3.7 | Click snapshot Delete | Confirm → snapshot removed |
| 3.8 | Click "+ Add rule" (firewall tab) | Modal: type/action/source/dest/proto/dport/comment |
| 3.9 | Edit rule | Form pre-filled, save updates |
| 3.10 | Delete rule | Confirm → rule removed |
| 3.11 | Click Migrate button | Dialog: target node dropdown + online toggle |
| 3.12 | Click Delete VM | Type-DELETE confirmation → redirects to /proxmox-vms |
| 3.13 | Type "delete" (lowercase) in confirm | Delete button stays disabled |
| 3.14 | Type "DELETE" (uppercase) | Delete button enables |
| 3.15 | Auto-refresh status | After 5s, status pill updates without manual refresh |

## Phase 14.4 — LXC List / Detail / Create Wizard (10 tests)

| # | Test | Expected |
|---|------|----------|
| 4.1 | Navigate `/proxmox-lxc` | LXC list (not VMs) |
| 4.2 | Click LXC row | Detail page with same tab structure as VM |
| 4.3 | Click "Create LXC" sidebar link | /proxmox-lxc/new wizard |
| 4.4 | Wizard step 1 | Template grid (or "No LXC templates found") |
| 4.5 | Wizard step 2 | VMID / hostname / cores / RAM / swap / disk / storage |
| 4.6 | Wizard step 3 | Bridge / IPv4 / IPv6 / VLAN |
| 4.7 | Wizard step 4 | Review all settings + "Start after creation" toggle |
| 4.8 | Wizard submit | API call → redirect to detail page |
| 4.9 | Wizard Back/Next | State persists across step navigation |
| 4.10 | LXC console tab | "LXC serial console — ships in v2.7" placeholder |

## Phase 14.5 — VM Create Wizard (6 tests)

| # | Test | Expected |
|---|------|----------|
| 5.1 | Navigate `/proxmox-vms/new` | 4-step wizard appears |
| 5.2 | Step 1 ISO picker | Lists ISOs from each iso-capable storage + "No media" option |
| 5.3 | Step 2 Hardware | VMID/name/cores/RAM/disk/BIOS/machine/CPU type all functional |
| 5.4 | Step 3 Network | Single NIC by default (now upgraded by 14.6) |
| 5.5 | Step 4 Confirm | All settings shown + "Start after creation" |
| 5.6 | Submit | API call → redirect to new VM's detail page |

## Phase 14.6 — Multi-NIC + Cloud-Init + EFI/TPM (6 tests)

| # | Test | Expected |
|---|------|----------|
| 6.1 | Wizard step 2: toggle "Cloud-Init drive" | Checkbox state persists |
| 6.2 | Wizard step 2: toggle "EFI disk" | TPM checkbox becomes enabled |
| 6.3 | Wizard step 2: toggle "TPM" (with EFI on) | Both stay on, TPM follows EFI |
| 6.4 | Wizard step 2: untoggle "EFI disk" | TPM unchecks automatically |
| 6.5 | Wizard step 3: click "+ Add NIC" | New NIC row appears (max 4) |
| 6.6 | Wizard step 3: NIC at limit | "+ Add NIC" button disabled at 4 |

## Phase 14.7 — Node Dashboard + Storage Content (8 tests)

| # | Test | Expected |
|---|------|----------|
| 7.1 | Navigate `/proxmox/nodes/{hostId}/{node}` | Node dashboard with breadcrumb |
| 7.2 | KPI strip | CPU% / Memory / Disk / Load / Uptime |
| 7.3 | Disks section | Table of physical disks |
| 7.4 | ZFS pools section | Table (or "No ZFS pools" if no ZFS) |
| 7.5 | Network section | Interfaces table |
| 7.6 | Services section | systemd services with state pills |
| 7.7 | Navigate `/proxmox/storage/{hostId}/{node}/{storage}` | Storage content list |
| 7.8 | Storage upload form | URL input + Upload button (if storage supports iso/vztmpl/backup) |

## Phase 14.8 — SDN + Node Firewall + IPSets + Aliases (8 tests)

| # | Test | Expected |
|---|------|----------|
| 8.1 | Navigate `/proxmox/nodes/{hostId}/{node}/firewall` | Node firewall table |
| 8.2 | Add node firewall rule | Dialog opens, save creates rule |
| 8.3 | Edit node firewall rule | Dialog pre-filled, save updates |
| 8.4 | Delete node firewall rule | Confirm → rule removed |
| 8.5 | Navigate `/proxmox/ipsets` | IPSets list |
| 8.6 | Create IPset + add CIDR + delete CIDR + delete IPset | All work |
| 8.7 | Navigate `/proxmox/aliases` | Aliases list (may be empty) |
| 8.8 | Navigate `/proxmox/sdn/zones` | "Endpoint not available yet" message (backend pending) |

## Phase 14.9 — Users + API Tokens + Pools (8 tests)

| # | Test | Expected |
|---|------|----------|
| 9.1 | Navigate `/proxmox/users` | Users list (may be empty) |
| 9.2 | Create user | Form: ID/password/name/email/role/enabled |
| 9.3 | Edit user | Form pre-filled (no password field) |
| 9.4 | Delete user | Confirm → user removed |
| 9.5 | Click userid | Navigates to /proxmox/users/:id/tokens |
| 9.6 | Create token | Form: ID/comment/expire/privsep |
| 9.7 | Token issued dialog | Plain-text token shown once with Copy button |
| 9.8 | Navigate `/proxmox/pools` | Pools list (may be empty) |

## Phase 14.10 — Tasks + Backup (5 tests)

| # | Test | Expected |
|---|------|----------|
| 10.1 | Navigate `/proxmox/tasks` | Cluster tasks table with filters |
| 10.2 | Tasks filter by node | Table filters to that node only |
| 10.3 | Tasks filter by status | Table filters by running/OK/error/stopped |
| 10.4 | Navigate `/proxmox/backup` | Backup jobs table |
| 10.5 | Backup job create | Form: ID/schedule/storage/keep-days |

---

## Cross-cutting tests (8 tests)

| # | Test | Expected |
|---|------|----------|
| X.1 | Dark theme consistency | Every page uses same color tokens |
| X.2 | Datadog style consistency | KPI stripes, status pills, badges everywhere |
| X.3 | Framer-motion animations | Smooth transitions, no jank |
| X.4 | Mobile responsive | All 10 phases work on 375px viewport |
| X.5 | Error handling | Each page shows friendly error, doesn't crash |
| X.6 | Empty states | Every page has "no data" message |
| X.7 | Loading states | Skeleton/spinner on every async page |
| X.8 | Cache bust | Hard refresh loads new bundle (check ?v= hashes) |

---

## Out of scope (deferred)

- **Phase 14.10 Replication, HA Groups, Certificates, SDN CRUD** — backend endpoints don't exist yet. These will be added when backend lands.
- **LXC serial console** — real xterm.js + pct serial-over-pty ships in a future phase.
- **Migrate dialog real impl** — currently toast-only.

---

## How to report results

| Outcome | Say |
|---------|-----|
| All 80+ pass | "all good — Tier 14 complete" |
| Specific failures | "fix [phase].[test#]: [what you saw]" |
| Mixed | "fix [a], [b], [c]"

I'll fix the issues, retest, then mark **Tier 14 COMPLETE** in memory + journal.