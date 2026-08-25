# Tier 10 — Homelab Dashboard (Homarr Parity)

## Why this tier

After 9 tiers of enterprise monitoring + intelligence + security, our tenant
needs a **homelab-style command center** where individual users (not just
admins) can pin services, jot personal notes, see their upcoming calendar,
track their media server queue, and run personal task automations. The
inspiration is **Homarr** — a popular open-source homelab dashboard —
but our version is multi-tenant, RBAC-scoped, and built on top of the
data we already have.

This is the **personal/glanceable** tier. Dashboards like Datadog answer
"What is the system doing?" — Homelab Dashboard answers "What am I doing
right now and what's coming up?"

## User Stories

### US-1 — Personal Glanceable Dashboard (H1)
**As** any tenant member, **I want** a single page that shows everything
personal to me in 30 seconds (services I pinned, my notes, my calendar,
my media, my tasks) **so that** I can start my day without opening 10
different tabs.

**Priority**: HIGHEST. The whole tier exists because of this.

### US-2 — Pin Services + Service Status (H2)
**As** a user, **I want** to pin services I care about (Proxmox VM,
Jellyfin server, Pi-hole, my dashboard, etc.) with a URL + icon + name,
**so that** I can one-click into them and see "Operational / Degraded /
Down" badges computed from heartbeats or HTTP probes.

**Priority**: HIGH. Without it, the dashboard is just notes + calendar.

### US-3 — Personal Notes + Todos (H3)
**As** a user, **I want** a quick Markdown notes widget (with checklist
support) and a separate todos widget with priority + due date, **so that**
I can jot down homelab todos ("buy new SSD for NAS") and tag-team with
my co-founder without leaving the dashboard.

**Priority**: HIGH. Notes are the most-used widget in any homelab
dashboard (Reddit /r/homelab, r/selfhosted feedback).

### US-4 — Calendar (H4)
**As** a user, **I want** to overlay my personal calendar (Google Cal,
Outlook, iCal URL) plus see upcoming agent maintenance windows, **so that**
I can plan reboots around my schedule.

**Priority**: MEDIUM. A nice-to-have that ties personal life to ops.

### US-5 — Download Stats (H5)
**As** a user with a media stack (Sonarr/Radarr/qBittorrent/SABnzbd),
**I want** to see "in progress" / "queue size" / "today's downloaded GB"
at a glance, **so that** I know whether my media server is choking.

**Priority**: MEDIUM. Specific to media-server users, but those are
exactly the homelab demographic.

### US-6 — Media Server (H6)
**As** a user with Plex/Jellyfin/Emby, **I want** to see "now playing"
on my server, recent additions, and transcoding session count, **so
that** I know whether the server is busy or idle.

**Priority**: MEDIUM. Same audience as H5.

### US-7 — Search (H7)
**As** any user, **I want** a global search bar at the top of the
dashboard that hits services (Proxmox VMs, my notes, todos, integrations),
**so that** I can find anything in <2 seconds.

**Priority**: MEDIUM. Power-user accelerator.

### US-8 — Per-User Preferences (H8)
**As** any user, **I want** my dashboard layout, pinned services,
widget order, theme, and refresh interval to persist per user (not
per tenant or globally), **so that** my homelab doesn't change
under me when my co-founder rearranges theirs.

**Priority**: HIGH. Without this, multi-user homelabs are broken.

### US-9 — RSS / Activity Feed (H9)
**As** any user, **I want** an RSS reader that polls blog feeds I care
about (r/selfhosted, my favorite dev blog, my self-hosted service's
release notes) **so that** I keep up with the ecosystem without
opening a separate reader.

**Priority**: LOW. Nice-to-have.

### US-10 — Task Scheduler (H10)
**As** any user, **I want** to schedule personal cron-like jobs
("reboot the Pi every Sunday at 3am", "ping the garage door daily"),
**so that** I don't have to maintain a separate cron system.

**Priority**: MEDIUM. Powerful for power users, scary for security.

## Scope (what's IN)

- 11 DB tables (1 per sub-feature, plus shared cross-cutting tables)
- 40+ protected routes
- ~25 public/internal widgets in the dashboard
- One unified `HomelabPage` with grid layout
- Per-user preference storage
- Service health polling (HTTP/HTTPS/TCP/ICMP)
- Markdown notes editor (no full WYSIWYG — keep simple)
- Calendar ingest from iCal URLs (Google/Outlook work via standard iCal)
- Sonarr/Radarr/qBittorrent API clients (read-only stats)
- Plex/Jellyfin API clients (read-only now-playing)
- RSS reader (background goroutine polling)
- Scheduler with safe-by-default action allowlist
- All sub-features tenant-scoped + RBAC-aware (new permission: `homelab:*`)

## Out of Scope (Tier 11+)

- Push-button deploy (Tier 11.1)
- Self-service signup (Tier 11.3)
- Mobile widgets (Tier 13)
- Widget marketplace
- Custom widget developer API

## Risks

1. **Scheduler (H10) is a security concern.** A user-schedulable cron
   is an SSRF / RCE risk if not carefully gated. Mitigation: allowlist
   of commands per user; no shell-out, only HTTP probes by default.
2. **iCal parsing** is a known footgun. Use a vetted stdlib parser
   (e.g. `github.com/lukechampine/ical`) — never hand-roll.
3. **Multi-user layout persistence** must not regress Tier 7 dashboard
   builder — keep the per-user homelab layout as a SEPARATE table
   from the tenant-wide dashboards.
4. **Service health polling** can DoS if a user adds 100 services
   pointing to the same host. Mitigation: per-user pin count cap (50).

## Architectural Decisions (high-level)

1. **Single Go binary** (api-gateway) hosts all routes — no new service.
2. **Single migration file** (`041_homelab.sql`) for all 11 tables.
3. **Polling workers** in `internal/homelab/`:
   - `servicehealth` (ticker every 60s)
   - `rss` (ticker every 5min)
   - `scheduler` (ticker every 60s, dispatches due jobs)
   - `calendar` (refresh every 30min per iCal URL)
   - `mediastats` (poll every 60s)
4. **Frontend**: 1 page (`HomelabPage.tsx`), 11 widgets, grid layout
   via existing `react-grid-layout`. Each widget is its own component
   (no widget >200 LOC).
5. **No new dependencies** except:
   - `github.com/lukechampine/ical` for iCal parsing
   - `github.com/mmcdole/gofeed` for RSS
6. **Modular discipline**: every file <400 LOC. Split handlers by domain
   (notes, todos, services, calendar, downloads, media, search, prefs,
   rss, scheduler, integrations).
