# Tier 11 Requirements Checklist — Platform & Commerce

## Functional

- [ ] **PL1** One-command deploy: `curl | bash` script + install.sh + token + status
- [ ] **PL2** Usage metering: events + hourly aggregation + history + summary + CSV export
- [ ] **PL3** Self-service signup: email/password + verify + resend + rate-limit 10/day/IP
- [ ] **PL4** Tenant limits: free/starter/pro/enterprise + dry-run check + usage warnings
- [ ] **PL5** Backup/restore: encrypted .tar.gz + manual + scheduled + restore roundtrip
- [ ] **PL6** Multi-region/HA: regions list + add + per-region health
- [ ] **PL7** Rate limiting: token-bucket + per-tier limits + headers + middleware
- [ ] **PL8** Platform health: summary + regions + top tenants + capacity forecast + alerts

## Non-functional

- [ ] Every Go file < 400 LOC
- [ ] Every TSX file < 400 LOC
- [ ] All 34 routes live on .115 + verified
- [ ] All 11 DB tables created + indexes
- [ ] All 5 background workers running without panic
- [ ] Tenant isolation enforced on every query
- [ ] New role `platform_admin` (separate from `super_admin`)
- [ ] INSTALL_MODE env var: cloud | self_hosted
- [ ] License key validation (cloud mode only)
- [ ] Backup encryption: AES-256-GCM with tenant-derived key
- [ ] Rate limiter: soft 429 with `Retry-After` header

## Verification gates

- [ ] `go build ./cmd/api-gateway` exit 0
- [ ] `go vet ./cmd/api-gateway` exit 0
- [ ] `cd web && npm run type-check` exit 0
- [ ] `cd web && npm run lint` no new errors
- [ ] `cd web && npm run build` exit 0
- [ ] All 34 routes return 401 without auth
- [ ] All 34 routes return 200/201/202 with valid JWT
- [ ] Tier 0-10 routes still work (regression gate)
- [ ] /health returns 200
- [ ] Backup + restore round-trip works (manual test)

## Deployment

- [ ] Binary built on Windows (`export GOOS=linux; export GOARCH=amd64`)
- [ ] Binary deployed to .115 via scp + systemctl restart
- [ ] Binary md5 verified
- [ ] All 5 workers started without panic

## Out-of-scope gates

- [ ] NO Tier 12 features (marketing site, pricing page, etc.)
- [ ] NO Tier 13 features (mobile)
- [ ] NO email service integration beyond dev mode
- [ ] NO public marketing site (Tier 12)
- [ ] NO Razorpay webhook HMAC validation (Tier 12)
