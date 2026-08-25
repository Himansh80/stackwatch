# Tier 13 Requirements Checklist — Mobile (React Native + Push)

## Functional

- [ ] **MO1** React Native mobile app (Expo): 6 screens (Login, Home, Servers, Alerts, Profile, Settings)
- [ ] **MO2** Push notification dispatch: 30s ticker + FCM for Android + APNS stub for iOS
- [ ] **MO3** Offline quick view: cached snapshot via AsyncStorage + last_fetched_at timestamp
- [ ] **MO4** Deep linking: push notification opens AlertDetail screen directly
- [ ] **MO5** Device registration: register device on app launch, heartbeat polling, refresh token

## Non-functional

- [ ] Every Go file < 400 LOC
- [ ] Every TS file < 400 LOC
- [ ] All 8 backend routes live on .115 + verified
- [ ] All 3 DB tables on .116 + indexes
- [ ] All 6 mobile screens render without TypeScript errors
- [ ] Mobile APK builds successfully (expo run:android OR expo build:android)
- [ ] Push latency <5s from alert fire to device
- [ ] Tenant isolation enforced on every query (tenant_id from JWT)
- [ ] Per-user (not per-tenant) on push devices/tokens (separate from tenant dashboards)
- [ ] Auth reuse: same JWT as web, no separate mobile auth flow

## Verification gates

- [ ] `go build ./cmd/api-gateway` exit 0
- [ ] `go vet ./cmd/api-gateway` exit 0
- [ ] `cd web && npm run type-check` exit 0
- [ ] `cd web && npm run lint` no new errors
- [ ] `cd web && npm run build` exit 0
- [ ] All 8 backend routes return 401 without auth
- [ ] All 8 backend routes return 200/201/202 with valid JWT
- [ ] Tier 0-12 routes still work (regression gate)
- [ ] /health returns 200
- [ ] `cd mobile && npm install` success
- [ ] `cd mobile && npx tsc --noEmit` passes

## Deployment

- [ ] Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- [ ] Binary deployed to .115 via scp + systemctl restart
- [ ] Binary md5 verified
- [ ] Push dispatch worker started without panic (journalctl evidence)
- [ ] Mobile APK built successfully

## Security

- [ ] HTTPS only (no HTTP fallback)
- [ ] JWT auth reuse (no new auth flow for mobile)
- [ ] Device registration requires auth
- [ ] Push log is immutable (no UPDATE/DELETE)

## Out-of-scope gates

- [ ] NO Apple Developer account work (Phase 2)
- [ ] NO App Store / Play Store uploads (Phase 2+)
- [ ] NO native iOS/Android code (Expo managed)
- [ ] NO watch app
- [ ] NO Phone call alert channel (already covered by Tier 11)