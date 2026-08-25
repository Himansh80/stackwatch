# Tier 10 Requirements Checklist — Homelab Dashboard

## Functional

- [ ] **H1** Widget framework: drag-resize-reorder, grid layout, per-widget refresh
- [ ] **H2** Service status: pin HTTP/TCP/ICMP services with health polling
- [ ] **H3** Notes (markdown) + Todos (priority/due/tags)
- [ ] **H4** Calendar: iCal URL ingest, week-view widget, upcoming events (30d)
- [ ] **H5** Download stats: Sonarr/Radarr/qBittorrent/SABnzbd clients
- [ ] **H6** Media server: Plex/Jellyfin/Emby now-playing + recent additions
- [ ] **H7** Global search: notes, todos, services, integrations
- [ ] **H8** Per-user prefs: layout, theme, refresh interval (separate from tenant dashboards)
- [ ] **H9** RSS reader: 5-min poll, configurable feeds
- [ ] **H10** Scheduler: cron-style with allowlisted actions (no shell)

## Non-functional

- [ ] Every Go file < 400 LOC
- [ ] Every TSX file < 400 LOC
- [ ] All 50 routes live on .115 + verified
- [ ] All 17 DB tables created + indexes
- [ ] All 6 background workers running without panic
- [ ] Tenant isolation enforced on every query
- [ ] RBAC `homelab:read` and `homelab:write` permissions
- [ ] Per-user service pin cap (50)
- [ ] Service poll rate limit (no DoS)
- [ ] Scheduler action allowlist (no shell, no RCE)
- [ ] iCal parser: vetted stdlib (lukechampine/ical)
- [ ] RSS parser: vetted stdlib (mmcdole/gofeed)
- [ ] No new dependencies beyond `ical` + `gofeed`

## Verification gates

- [ ] `go build ./cmd/api-gateway` exit 0
- [ ] `go vet ./cmd/api-gateway` exit 0
- [ ] `cd web && npm run type-check` exit 0
- [ ] `cd web && npm run lint` no new errors
- [ ] `cd web && npm run build` exit 0
- [ ] All 50 routes return 401 without auth (cross-tier gate)
- [ ] All 50 routes return 200/201/204 with valid JWT (auth gate)
- [ ] Tier 0-9 routes still work (regression gate)
- [ ] /health returns 200

## Deployment

- [ ] Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- [ ] Binary deployed to .115 via scp + systemctl restart
- [ ] Binary md5 verified
- [ ] All 6 workers started without panic

## Out-of-scope gates

- [ ] NO Tier 11 features (deploy, HA, signup)
- [ ] NO Tier 12 features (docs, GTM)
- [ ] NO Tier 13 features (mobile)
