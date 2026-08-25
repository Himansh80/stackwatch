# Tier 11 Requirements Checklist — Platform & Commerce

## Functional

- [ ] **PL1** Push-button deploy: one-liner install + token lifecycle
- [ ] **PL2** Usage metering: per-tenant, per-day event tracking + rollup
- [ ] **PL3** Self-service signup: captcha + email verification + IP rate limit
- [ ] **PL4** Tenant limits: 4 plans (free/pro/business/enterprise) enforced at API layer
- [ ] **PL5** Backup/restore: daily + on-demand pg_dump + 30d retention + checksum
- [ ] **PL6** Multi-region/HA: read replicas + replication lag tracking
- [ ] **PL7** Rate limiting: per-IP + per-tenant + per-route sliding window
- [ ] **PL8** Platform health: operator dashboard + public status page

## Non-functional

- [ ] Every Go file < 400 LOC
- [ ] Every TSX file < 400 LOC
- [ ] All 25+ Tier 11 routes live on .115 + verified
- [ ] All 8 Tier 11 DB tables on .116
- [ ] All 6 background workers running
- [ ] Tenant isolation enforced on every query
- [ ] Tier isolation enforced via plan_limits
- [ ] Admin endpoints protected by super-admin middleware
- [ ] Public endpoints (signup, install-script, status) are public
- [ ] No new frontend dependencies beyond what's already installed
- [ ] No new Go dependencies beyond standard library + Resend/Stripe SDKs (dev-mode stubs)

## Verification gates

- [ ] `go build ./cmd/api-gateway` exit 0
- [ ] `go vet ./cmd/api-gateway` exit 0
- [ ] `cd web && npm run type-check` exit 0
- [ ] `cd web && npm run lint` no new errors
- [ ] `cd web && npm run build` exit 0
- [ ] All Tier 11 routes return 401/200/201 as expected
- [ ] Public routes return 200/400 (no auth required)
- [ ] Tier 0-10 routes still work (regression gate)
- [ ] /health returns 200

## Deployment

- [ ] Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- [ ] Binary deployed to .115 via scp + systemctl restart
- [ ] Binary md5 verified
- [ ] All 6 workers started without panic
- [ ] Public status page accessible at https://stackwatch.smarthomelab.fun/status

## Out-of-scope gates

- [ ] NO marketing site (Tier 12)
- [ ] NO docs site (Tier 12)
- [ ] NO mobile app (Tier 13)
- [ ] NO auto-failover (v2)
