# Tier 11 — Platform & Commerce (the money phase)

## Why this tier

After 10 tiers of building infrastructure, monitoring, intelligence,
security, and homelab, we have a great product. Now we need to **make
money from it** — turn it into a real commercial SaaS without breaking
the self-hosted model.

This is the **commercial platform** tier. It adds:
1. **One-command deployment** (so non-engineers can self-host)
2. **Usage metering** (so we can bill accurately)
3. **Self-service signup** (so we don't manually onboard every tenant)
4. **Tenant limits** (so free users can't run away with our infra)
5. **Backup/restore** (so a single bug doesn't destroy 1,000 tenants)
6. **Multi-region/HA** (so one datacenter going down doesn't take us down)
7. **Per-tier rate limiting** (so a noisy tenant doesn't starve others)
8. **Platform health** (so we know our platform is healthy from the inside)

## User Stories

### US-1 — One-Command Self-Host Deploy (PL1)
**As** a small DevOps team / indie hacker, **I want** to run my own
StackWatch instance with a single `curl | bash` command, **so that** I
don't need to hire a backend engineer to deploy it.

**Priority**: HIGH. Every indie hacker is a potential paying customer.

### US-2 — Usage Metering + Billing (PL2)
**As** a SaaS operator, **I want** accurate usage tracking per tenant
(servers, alerts, dashboard panels, API calls, storage GB) **so that**
I can bill correctly, detect abuse, and forecast capacity.

**Priority**: HIGH. Without this, we can't charge for the platform.

### US-3 — Self-Service Signup (PL3)
**As** a prospective customer, **I want** to sign up for a free tier
with just my email + password (no sales call) **so that** I can try
the product immediately.

**Priority**: HIGH. Today every customer requires manual provisioning.

### US-4 — Tenant Limits (PL4)
**As** a SaaS operator, **I want** free tier capped at 3 servers /
7-day retention, pro tier at 50 servers / 90-day, enterprise unlimited,
**so that** the free tier doesn't bankrupt us and the paid tier is
clearly differentiated.

**Priority**: HIGH. Margin protection.

### US-5 — Backup/Restore (PL5)
**As** any tenant admin, **I want** to export my full tenant config
(servers, alerts, dashboards, homelab data) as a single .tar.gz
**so that** I can restore to a new instance or recover from disaster.

**Priority**: HIGH. Single biggest trust-builder.

### US-6 — Multi-Region/HA (PL6)
**As** a SaaS operator, **I want** to deploy StackWatch across 2+ regions
with active-active Postgres replication, **so that** an AZ outage
doesn't take the entire platform offline.

**Priority**: MEDIUM. Required for paying customers, but most can
tolerate one region initially.

### US-7 — Rate Limiting (PL7)
**As** a SaaS operator, **I want** per-tenant rate limits based on
plan tier (free=10 req/min, pro=60, enterprise=1000) **so that** one
noisy tenant doesn't starve the rest.

**Priority**: HIGH. Required for any multi-tenant SaaS.

### US-8 — Platform Health (PL8)
**As** a SaaS operator (the SUPER_ADMIN of multi-tenant instances),
**I want** a single dashboard showing per-region health, per-tenant
top-N, billing summary, support tickets, capacity forecasts **so that**
I can spot trouble before customers do.

**Priority**: HIGH. Operators live and die by their dashboards.

## Scope (what's IN)

- 8 DB tables (1 per sub-feature, plus shared cross-cutting)
- ~45 new routes (mostly under `/api/v1/platform/*`)
- 1 unified `PlatformPage` with 6 tabs (Deploy, Metering, Signup,
  Limits, Backups, Health)
- **NEW admin role**: `platform_admin` (separate from `super_admin`
  which is tenant-scoped). Platform admins see ALL tenants.
- New package `internal/platform/` (alongside `internal/homelab/`)
- Background workers for: usage aggregation (hourly), retention
  enforcement (daily), rate-limit metric collection (1min), backup
  scheduler (configurable), capacity forecast (daily)
- New env var `INSTALL_MODE` (cloud | self_hosted) — auto-detect from
  whether LICENSE_KEY is set

## Out of Scope (Tier 12+)

- Public marketing site (`stackwatch.io`) — Tier 12
- Stripe integration — Tier 11.2 (covered) but Razorpay, bank transfer, etc. — Tier 12
- Email service integration (Resend/SMTP) — already stubbed in Tier 0
- Mobile app — Tier 13

## Risks

1. **Self-service signup opens abuse vector** — mitigation: email
   verification + rate limit on signup endpoint (10/day per IP).
2. **Multi-region replication is complex** — mitigate by using a
   managed Postgres (RDS/Neon/Supabase) for cloud mode, document
   self-host replica setup for self-host mode.
3. **Backup encryption** — encrypt .tar.gz with AES-256-GCM using a
   tenant-derived key from the master key.
4. **Capacity forecast must be cheap** — use simple linear regression
   on raw metrics; no ML models.
5. **Rate limiter must not lock out legit users** — soft limit (429)
   with retry-after, no hard cutoffs.

## Architectural Decisions (high-level)

1. **Two install modes**: `cloud` (multi-tenant, public signup, billing)
   and `self_hosted` (single-tenant, no signup, no billing). Detected
   at boot from env vars.
2. **Multi-tenant DB** (already in place — tenant_id on every table).
3. **Per-tenant API key auth** (already in place).
4. **One binary** (api-gateway) hosts all platform routes — no new
   service.
5. **New migration file** (`042_platform.sql`) for all 8+ tables.
6. **Polling workers** in `internal/platform/`:
   - `usage_meter` (hourly aggregation)
   - `retention` (daily cleanup of expired data)
   - `rate_limiter` (1min windowing)
   - `backup_scheduler` (configurable cadence)
   - `capacity_forecast` (daily linear regression)
7. **Frontend**: 1 page (`PlatformPage.tsx`), 8 sections. Each <300 LOC.
8. **No new dependencies** beyond `archive/tar` (stdlib) and
   `compress/gzip` (stdlib) for backups.
9. **Modular discipline**: every file <400 LOC. Split handlers by
   domain (deploy, metering, signup, limits, backup, ratelimit,
   health, multiregion).
