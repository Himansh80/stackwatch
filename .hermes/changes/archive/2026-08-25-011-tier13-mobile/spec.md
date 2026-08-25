# Tier 13 Specification — Mobile (React Native + Push)

## 1. Goals (measurable)

| Metric | Target |
|--------|--------|
| Tier 13 backend routes | ≥ 8 |
| Tier 13 DB tables | 3 (push_devices, push_log, mobile_sessions) |
| Mobile app LOC | ≥ 2,000 (Expo React Native) |
| Mobile screens | 6 (Login, Home, Servers, Alerts, Profile, Settings) |
| All Go files <400 LOC | Yes |
| All TS files <400 LOC | Yes |
| Push dispatch latency | <5s from alert fire to device |
| FCM/APNS message format | Rich notification with action buttons |

## 2. Sub-features

### MO1 — React Native Mobile App

**Directory**: `mobile/` at project root (NOT `web/`)

**Stack**: Expo SDK 51 + React Native 0.74 + React Navigation 6 + TypeScript

**Screens (6, each <300 LOC)**:1. `LoginScreen` — email + password + "Remember me" + tenant picker (if multi-tenant)
2. `HomeScreen` — KPI strip + quick actions + recent alerts
3. `ServersScreen` — list + searchable + status badges + refresh
4. `AlertsScreen` — firing alerts list + acknowledge/resolve buttons
5. `ProfileScreen` — user profile + avatar + tenant info + logout
6. `SettingsScreen` — push notification toggle + theme picker + about

**Offline-first** (Phase 2): cached snapshot via AsyncStorage + last_fetched_at timestamp.**Modular structure** (every file <400 LOC):
- `mobile/src/screens/` — 6 screens
- `mobile/src/components/` — 12 components (KpiCard, AlertRow, ServerCard, etc.)
- `mobile/src/hooks/` — 4 custom hooks (useServers, useAlerts, usePushNotifications, useTheme)
- `mobile/src/navigation/` — 2 files (AppNavigator, LinkingConfiguration)
- `mobile/src/lib/` — 3 files (api.ts, auth.ts, theme.ts)
- `mobile/src/constants/` — 1 file (routes.ts)
- `mobile/assets/` — icon + splash

### MO2 — Push Notifications

**Tables (3):**`push_devices`:
```sql
CREATE TABLE IF NOT EXISTS push_devices (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    user_id         uuid NOT NULL,
    platform        text NOT NULL CHECK (platform IN ('android', 'ios', 'web')),
    fcm_token       text NOT NULL,
    app_version     text,
    device_name     text,
    last_active_at  timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    enabled         boolean NOT NULL DEFAULT true,
    UNIQUE (tenant_id, platform, fcm_token)
);
```

`push_log`:
```sql
CREATE TABLE IF NOT EXISTS push_log (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    device_id       uuid NOT NULL REFERENCES push_devices(id) ON DELETE CASCADE,
    alert_id        uuid,
    title           text NOT NULL,
    body            text NOT NULL,
    data            jsonb NOT NULL DEFAULT '{}',
    status          text NOT NULL DEFAULT 'queued',
    fcm_message_id  text,
    apns_id         text,
    sent_at         timestamptz,
    error_message   text,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_push_log_tenant_time ON push_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_push_log_status     ON push_log(status) WHERE status IN ('queued', 'failed');
```

`mobile_sessions` (for dev-login tracking):
```sql
CREATE TABLE IF NOT EXISTS mobile_sessions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL,
    device_name     text,
    app_version     text,
    ip_address      text,
    user_agent      text,
    last_active_at  timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now()
);
```

**Routes (8 protected)**:
- `GET    /api/v1/mobile/push/devices` — list my registered devices
- `POST   /api/v1/mobile/push/register` — register device (called on app launch)
- `DELETE /api/v1/mobile/push/devices/:id` — unregister device
- `GET    /api/v1/mobile/push/log` — list push notification history
- `POST   /api/v1/mobile/push/test` — send test push notification
- `GET    /api/v1/mobile/push/settings` — get push preferences
- `PATCH  /api/v1/mobile/push/settings` — update preferences
- `GET    /api/v1/mobile/sessions` — list my active sessions

**Background worker**: `internal/mobile/push_dispatch.go`
- Polls `alert_notifications` table every 30s for undelivered alerts
- For each pending alert: lookup all devices for that user via JWT claims
- For each device: call FCM (Android) or APNs (iOS) to send
- Update `push_log` with status = 'sent' | 'failed' + message IDs

### MO3 — Deep Linking

React Navigation linking configuration:
```typescript
const linking = {
  prefixes: ['stackwatch://', 'https://stackwatch.smarthomelab.fun'],
  config: {
    screens: {
      Home: 'home',
      Servers: 'servers',
      Alerts: 'alerts',
      AlertDetail: 'alerts/:id',  // ← push notification deep link
      Profile: 'profile',
    },
  },
};
```

Push notification payload: `{alertId: uuid, alertTitle: string, severity: string}` — on click, app opens AlertDetail screen with that ID.

## 3. Non-functional

- Every Go file < 400 LOC
- Every TS file < 400 LOC
- Backend: 8 new routes live on .115
- DB: 3 new tables on .116
- Mobile: 6 screens, all <300 LOC each
- Push latency: <5s from alert fire to device
- Auth: reuse JWT from `Tier 0` / `Tier 3` — no new auth flow
- Encryption: HTTPS only, no HTTP fallback

## 4. Verification gates

- `go build ./cmd/api-gateway` exit 0
- `go vet ./cmd/api-gateway` exit 0
- `cd web && npm run type-check` exit 0
- `cd web && npm run lint` no new errors
- `cd web && npm run build` exit 0
- All 8 backend routes return 401 without auth
- All 8 backend routes return 200/201/202 with valid JWT
- Tier 0-12 routes still work (regression gate)
- /health returns 200
- `npx expo export` (or `npm run build`) exit 0 for mobile
- Mobile APK compiles without TypeScript errors

## 5. Deployment

- Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- Binary deployed to .115 via scp + systemctl restart
- Binary md5 verified
- Mobile APK built via `eas build --platform android` (or local `expo export`)

## 6. Out-of-scope gates

- NO Apple Developer account work (Phase 2 only)
- NO App Store / Play Store uploads (Phase 2+)
- NO native iOS/Android code (Expo managed)
- NO watch app
- NO Phone call alert channel (already covered by Tier 11)