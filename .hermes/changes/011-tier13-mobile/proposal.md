# Tier 13 — Mobile (React Native + Push Notifications)

## Why this tier

After 12 tiers of building the platform backend + web dashboard + docs,
we have a complete commercial SaaS for infrastructure monitoring. But
customers can't see it from their phones. This tier adds the **mobile
experience** — a React Native app that works on iOS + Android with
push notifications.

This is the **final tier** of the product — the "everywhere you are" tier.
After this, StackWatch is feature-complete and can be sold to real customers.

## User Stories

### US-1 — Cross-Platform Mobile App (MO1)
**As** a customer (enterprise admin or individual operator),
**I want** a native iOS + Android app where I can see my servers,
alerts, dashboards, and homelab widgets **so that** I can monitor
my infrastructure from wherever I am.

**Priority**: HIGH. Without mobile, we're not a complete product.

### US-2 — Push Notifications (MO2)
**As** the same customer, **I want** real-time push notifications
for critical alerts (server down, high CPU, custom threshold breach)
**so that** I don't have to open the app to learn about problems.

**Priority**: HIGH. Alerts without push = unread = useless.

### US-3 — Offline-First Quick View (MO3)
**As** a mobile user with flaky connectivity, **I want** the app to
show me cached snapshot data when I'm offline, **so that** I can still
diagnose issues on the plane or in a tunnel.

**Priority**: MEDIUM. Nice-to-have for the UX.

### US-4 — Mobile-Optimized Dashboards (MO4)
**As** the same customer, **I want** dashboards that render cleanly
on a narrow phone screen (not just scaled-down desktop), **so that**
I can actually read the charts.

**Priority**: MEDIUM. Important for usability, not core MVP.

## Scope (what's IN)

- **Mobile app** (Expo React Native): login + dashboard + server list + alerts + settings
- **Push notification backend**: new service + FCM/APNS dispatch
- **3 new DB tables**: `push_devices`, `push_log`, `mobile_sessions`
- **6 mobile screens**: Login, Home, Servers, Alerts, Profile, Settings
- **Background push dispatch worker**: polls `alert_notifications` table every 30s
- **Deep linking**: push notification opens specific alert page in app
- **Modular discipline**: every Go + TS file <400 LOC

## Out of Scope (deferred)

- Mobile-specific GTM (App Store / Play Store uploads) — separate track
- Watch app (Apple Watch / Wear OS) — future
- Mobile widgets — future
- Native iOS/Android features (Face ID, fingerprint auth, etc.) — Phase 2
- Phone-call alert channel — already covered by Tier 11 (no extra work needed)
- Full offline-first sync engine — Phase 2 (only cached snapshot for Phase 1)

## Risks

1. **Expo/EAS push notification complexity** — mitigate by using
   Expo's managed `PushNotifications` API (not bare React Native).
2. **iOS certificates + provisioning** — mitigate by deferring to
   Phase 2; Phase 1 uses Android APK only (which is free to sideload).
3. **Apple Developer account needed** for push notifications — user
   needs to pay $99/year to Apple for production push. Without that,
   Android via FCM is free.
4. **Mobile app signed APK distribution** — Phase 1 uses sideload via
   direct download; Phase 2 uses Play Store if desired.

## Architecture Decisions (high-level)

1. **One backend (api-gateway)** hosts all mobile routes — no new service.
2. **Push dispatch is in-process** via a new background worker —
   not a separate microservice.
3. **Mobile app is Expo-managed** (not bare React Native) for faster
   iteration + over-the-air updates.
4. **Push providers**: FCM for Android + APNs for iOS (Phase 2 only if
   user has developer account).
5. **Auth reuse**: same JWT auth as web — no separate mobile auth flow.
6. **Deep linking**: push → opens app → navigates to alert detail
   via React Navigation deep linking.
7. **Mobile app is OPTIONAL** — dashboard still works fine without it.
