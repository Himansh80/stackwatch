# Tier 13 Plan — Mobile (React Native + Push)

## Phase 0 — Foundation (DB + routes split)

**Tables (3)**: `push_devices`, `push_log`, `mobile_sessions`**Routes (8 under `/api/v1/platform/mobile/*`):** all protected, all <400 LOC handler files

**Files:**
- `migrations/043_mobile.sql` (1 new migration)
- `cmd/api-gateway/routes_mobile.go` (new, ~100 LOC)
- `internal/handler/handlers_mobile_push.go` (~250 LOC, 4 routes)
- `internal/handler/handlers_mobile_devices.go` (~200 LOC, 3 routes)
- `internal/handler/handlers_mobile_sessions.go` (~150 LOC, 1 route)
- `internal/handler/handlers_mobile_types.go` (~100 LOC)

## Phase 1 — Push notification dispatch (backend worker)

**New package `internal/mobile/`:**
- `mobile/types.go` (~100 LOC) — types: FCMMessage, APNsMessage, PushResult
- `mobile/push_dispatch.go` (~350 LOC) — background worker, 30s ticker
  - Polls `alert_notifications` table for `status='queued'`
  - Looks up all`push_devices` for the target user via JWT claims
  - For Android: call FCM `POST https://fcm.googleapis.com/fcm/send` with `Authorization: key=<FCM_SERVER_KEY>`
  - For iOS: call APNs `POST https://api.push.apple.com/3/device/<apns_token>` with `authorization: bearer <JWT>`  (Phase 2 only)
  - Update `push_log.status = 'sent'` or `'failed'`
  - Graceful shutdown + error logging
- `mobile/push_dispatcher_stack.go` (~200 LOC) — Stack driver, rate limiting, retry logic

**Wire into `cmd/api-gateway/main.go`.**

## Phase 2 — Mobile App Scaffold (Expo React Native)

**New directory:** `mobile/` (sibling to `web/`)

**Files:**
- `mobile/package.json` (~100 LOC) — expo + react-native + typescript + react-navigation + @expo/vector-icons+ react-native-safe-area-context + expo-notifications + expo-constants
- `mobile/app.json` (~30 LOC) — bundle id, version, icon, splash
- `mobile/babel.config.js` (~10 LOC)
- `mobile/tsconfig.json` (~25 LOC)
- `mobile/src/App.tsx` (~50 LOC) — root component
- `mobile/src/navigation/AppNavigator.tsx` (~200 LOC) — Stack + Tab navigator
- `mobile/src/navigation/LinkingConfiguration.ts` (~50 LOC) — deep linking
- `mobile/src/lib/api.ts` (~200 LOC) — REST client (reuse logger + JWT)
- `mobile/src/lib/auth.ts` (~150 LOC) — login/logout, JWT storage
- `mobile/src/lib/theme.ts` (~150 LOC) — colors + typography
- `mobile/src/constants/screens.ts` (~20 LOC)
- `mobile/src/hooks/useServers.ts` (~100 LOC)
- `mobile/src/hooks/useAlerts.ts` (~100 LOC)
- `mobile/src/hooks/usePushNotifications.ts` (~150 LOC)

## Phase 3 — Mobile Screens (6 screens, all <300 LOC)

- `mobile/src/screens/LoginScreen.tsx` (~250 LOC) — email + password + submit + errors + biomechanism
- `mobile/src/screens/HomeScreen.tsx` (~250 LOC) — KPI strip + quick actions + recent alerts
- `mobile/src/screens/ServersScreen.tsx` (~200 LOC) — searchable list + status badges + refresh
- `mobile/src/screens/AlertsScreen.tsx` (~200 LOC) — alert list + acknowledge/resolve buttons
- `mobile/src/screens/ProfileScreen.tsx` (~150 LOC) — user info + tenant + logout
- `mobile/src/screens/SettingsScreen.tsx` (~150 LOC) — push toggle + theme picker + about

## Phase 4 — Push Notification Integration (mobile side)

- `mobile/src/hooks/usePushNotifications.ts` — register for push on app launch, save FCM token, receive + handle notifications
- `mobile/src/lib/notifications.ts` (~100 LOC) — Expo push channel setup
- Wire into AppNavigator — deep link to AlertDetail when push is tapped

## Phase 5 — Finalize (TIER 13 COMPLETE)

- Update `docs/` if needed
- `journal-tier13-final.md` written
- Archive `.hermes/changes/011-tier13-mobile/`
- Final commit: `docs(tier13): TIER 13 COMPLETE marker + archive + journal`
- **TIER 13 COMPLETE marker**

## Total (estimated)

- **3 DB tables**
- **8 backend routes** (all <400 LOC)
- **6 mobile screens** (all <300 LOC)
- **12 hooks + 5 lib modules + 2 navigation modules**
- **~25 mobile files** (all <400 LOC)
- **1 new background worker** (push dispatch, 30s)
- **No new dependencies beyond Expo/React Native ecosystem**

## Risks

1. **Expo EAS build complexity** — mitigate by using `expo prebuild` + `expo run:android` locally (no cloud builds needed for dev)
2. **Firebase Cloud Messaging key management** — needs .env var`FCM_SERVER_KEY`
3. **iOS push certificates** — deferred to Phase 2 (requires $99 Apple Developer account)
4. **App signing** — Phase 1 sideload only; Phase 2 uses Play Store if user wants

## Modular approach

Every file <400 LOC. Backend: handlers split by domain (`handlers_mobile_push.go`, `handlers_mobile_devices.go`, `handlers_mobile_sessions.go`). Mobile: screens split by screen, components by feature, hooks by data domain.

**Datadog-style UI:**
- KPI cards with colored stripes
- Status pills (up/degraded/down)
- Dark theme (system-aware)
- Tight typography
- Refresh + loading states
- Empty states with helpful instructions

**Cross-platform consistency**: all screens follow the same component patterns as the web dashboard.