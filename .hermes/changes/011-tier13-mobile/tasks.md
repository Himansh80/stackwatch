# Tier 13 Tasks — Mobile (React Native + Push)

## Phase 0 — Foundation (DB + Routes)

- [ ] 0.1 Migration `migrations/043_mobile.sql`: 3 tables (push_devices, push_log, mobile_sessions)
- [ ] 0.2 Apply migration to prod `.116` DB
- [ ] 0.3 Create `cmd/api-gateway/routes_mobile.go` (stub `mountMobileRoutes`)
- [ ] 0.4 Modify `cmd/api-gateway/routes.go` to call `mountMobileRoutes(protected, pool)` after `mountPlatformRoutes`
- [ ] 0.5 Create `internal/handler/handlers_mobile_types.go` (~100 LOC)
- [ ] 0.6 `go build` passes, `go vet` passes, commit: `feat(tier13): Mobile Phase 0 — foundation`

## Phase 1 — Push Notification Dispatch (Backend Worker)

- [ ] 1.1 Create directory `internal/mobile/` (package mobile)
- [ ] 1.2 `internal/mobile/types.go` (~100 LOC) — types: FCMMessage, APNsMessage, PushResult, dispatchResult
- [ ] 1.3 `internal/mobile/push_dispatch.go` (~350 LOC) — PushDispatcher with 30s ticker
  - Query `alert_notifications` for `status='queued'`
  - For each alert, resolve user via JWT claims (tenant_id + user_id from alert's row)
  - For each user, query `push_devices` where tenant_id/user_id + enabled=true
  - For Android: FCM `POST https://fcm.googleapis.com/fcm/send`
  - For iOS: stub (returns `not_implemented` for Phase 1)
  - Update `push_log.status = 'sent'` or `'failed'`
- [ ] 1.4 `internal/mobile/push_dispatcher_stack.go` (~200 LOC) — request builder + HTTP client + rate limiter
- [ ] 1.5 Wire into `cmd/api-gateway/main.go` (alongside other workers)
- [ ] 1.6 Verify: `go build` exit 0, `go vet` exit 0
- [ ] 1.7 Deploy binary to .115, restart service
- [ ] 1.8 Verify: `journalctl -u stackwatch-api-gateway --no-pager -n 50 | grep` shows push dispatcher started
- [ ] 1.9 Commit: `feat(tier13): Push dispatch worker (Phase 1)`

## Phase 2 — Push API Endpoints (Backend Routes)

- [ ] 2.1 `internal/handler/handlers_mobile_push.go` (~250 LOC, 4 routes):
  - `GET /api/v1/platform/mobile/push/devices` — list my registered devices
  - `POST /api/v1/platform/mobile/push/register` — register device (body: {platform, fcm_token, device_name, app_version})
  - `DELETE /api/v1/platform/mobile/push/devices/:id` — unregister
  - `GET /api/v1/platform/mobile/push/log` — list push history (paginated)
- [ ] 2.2 `internal/handler/handlers_mobile_devices.go` (~200 LOC, 3 routes):
  - `GET /api/v1/platform/mobile/devices` — list my devices (with last_active_at)
  - `POST /api/v1/platform/mobile/devices/heartbeat` — device heartbeat (updates last_active_at)
  - `POST /api/v1/platform/mobile/devices/refresh-token` — refresh FCM token
- [ ] 2.3 `internal/handler/handlers_mobile_sessions.go` (~150 LOC, 2 routes):
  - `GET /api/v1/platform/mobile/sessions` — list my sessions
  - `POST /api/v1/platform/mobile/test-push` — send test push notification (dev only)
- [ ] 2.4 Register all 9 routes in `mountMobileRoutes`
- [ ] 2.5 Deploy + verify + commit: `feat(tier13): Push API endpoints (Phase 2)`

## Phase 3 — Mobile App Scaffold (Expo)

- [ ] 3.1 Create `mobile/` directory structure (package.json, app.json, babel.config.js, tsconfig.json)
- [ ] 3.2 `mobile/src/App.tsx` (~50 LOC) — root component with SafeArea + StatusBar + Navigation
- [ ] 3.3 `mobile/src/navigation/AppNavigator.tsx` (~200 LOC) — Stack + Tab navigator
- [ ] 3.4 `mobile/src/navigation/LinkingConfiguration.ts` (~50 LOC) — deep link config
- [ ] 3.5 `mobile/src/lib/api.ts` (~200 LOC) — typed fetch wrapper + JWT header
- [ ] 3.6 `mobile/src/lib/auth.ts` (~150 LOC) — login, logout, token storage
- [ ] 3.7 `mobile/src/lib/theme.ts` (~150 LOC) — colors + typography
- [ ] 3.8 `mobile/src/constants/screens.ts` (~20 LOC)
- [ ] 3.9 Verify: `cd mobile && npm install` succeeds, `npx tsc --noEmit` passes
- [ ] 3.10 Commit: `feat(tier13): Mobile scaffold (Phase 3)`

## Phase 4 — Mobile Screens

- [ ] 4.1 `mobile/src/screens/LoginScreen.tsx` (~250 LOC) — email/password + tenant picker + forgot password
- [ ] 4.2 `mobile/src/screens/HomeScreen.tsx` (~250 LOC) — KPI strip + quick actions + recent alerts + pull-to-refresh
- [ ] 4.3 `mobile/src/screens/ServersScreen.tsx` (~200 LOC) — searchable list + status badges + filter
- [ ] 4.4 `mobile/src/screens/AlertsScreen.tsx` (~200 LOC) — firing alerts with severity badges + acknowledge + resolve
- [ ] 4.5 `mobile/src/screens/ProfileScreen.tsx` (~150 LOC) — avatar + user info + tenant + logout
- [ ] 4.6 `mobile/src/screens/SettingsScreen.tsx` (~150 LOC) — push notification toggle + about
- [ ] 4.7 Shared components: `mobile/src/components/KpiCard.tsx`, `AlertRow.tsx`, `ServerCard.tsx`, `EmptyState.tsx` (~400 LOC total)
- [ ] 4.8 Hooks: `mobile/src/hooks/useServers.ts`, `useAlerts.ts`, `usePushNotifications.ts` (~300 LOC total)
- [ ] 4.9 Verify: `npm run type-check` passes, `npm run lint` no new errors
- [ ] 4.10 Commit: `feat(tier13): Mobile screens (Phase 4)`

## Phase 5 — Push Notification Wiring (Mobile Side)

- [ ] 5.1 `mobile/src/lib/notifications.ts` (~100 LOC) — Expo Notifications setup, channel creation, deep link handling
- [ ] 5.2 Wire push token registration into AppNavigator (on app start, after login)
- [ ] 5.3 Wire push notification tap → deep link to alert detail screen
- [ ] 5.4 Add `expo-notifications` + `expo-constants` to package.json
- [ ] 5.5 Verify: APK builds (via `expo run:android` on device OR `expo build:android` if EAS configured)
- [ ] 5.6 Commit: `feat(tier13): Push notification wiring (Phase 5)`

## Phase 6 — Finalize (TIER 13 COMPLETE)

- [ ] 6.1 Verify all 8 Tier 13 backend routes return 401 without auth
- [ ] 6.2 Verify all 6 mobile screens render without TypeScript errors
- [ ] 6.3 Verify push log shows at least 1 test entry (from test-push route)
- [ ] 6.4 Cross-tier regression (Tier 0-12 still work)
- [ ] 6.5 Archive `.hermes/changes/011-tier13-mobile/` + `specs/011-tier13-mobile/`
- [ ] 6.6 Write `journal-tier13-final.md`
- [ ] 6.7 Commit: `docs(tier13): TIER 13 COMPLETE marker + archive + journal`
