# StackWatch — Security Model

**Last updated:** 2026-08-17

This document covers the current security posture of StackWatch. We
follow one rule: **be honest about what is and isn't implemented.**

---

## Threat model

StackWatch is designed for:

- Operators of small-to-medium homelabs and SMB IT
- **Trust boundary**: any user with a valid JWT is considered trusted
- **Out-of-scope**: state-actor attacks, supply-chain compromises of `go` or
  `node` modules after `go.sum`/`package-lock.json` resolution, FIPS-grade
  cryptographic requirements

If you need audit-grade security for a regulated environment, deploy
StackWatch self-hosted in an air-gapped network with `golangci-lint run`
gating.

---

## What we implemented

### Authentication

- Bcrypt password hashing (cost 12)
- JWT bearer tokens (HS256, 24h TTL)
- `iss`, `iat`, `exp` claims; server-side verification
- Per-request claims injected into gin Context (see `auth/ClaimsFromContext`)
- Refresh endpoint re-issues tokens without re-prompting credentials
- `must_change_password` flag (auth.Middleware blocks until reset)
- 24h lockout after 5 failed logins (in-memory; per-instance)

### Authorization

- Per-tenant scoping on every query (tenants see only their own data)
- `super_admin` role bypasses tenant filter (escape hatch for ops)
- Endpoint-level role checks (e.g. `POST /users` requires super-admin)
- API keys stored as `sha256(token)`; never visible after creation

### Transport

- TLS optional (terminate at your reverse proxy)
- Standard `securityHeaders` middleware: HSTS, CSP, Referrer-Policy
- CORS allowlist via env (`ALLOWED_ORIGINS`)

### Audit

- `audit_logs` table records `auth`, `user`, `tenant`, `api_key` events
- Required for `auth.change_password`, `user.create`, etc.

---

## What we explicitly DO NOT have (yet)

| Feature | Status | ETA |
|---------|--------|-----|
| Row-Level Security (RLS) in Postgres | Tier 9 | Tier 9 |
| SAML/OIDC SSO | Tier 9 | Tier 9 |
| IP allowlist on login | Pending | Tier 9 |
| Brute-force lockout persisted (table-based) | Tier 9 | Tier 9 |
| Tamper-evident audit log (hash chain) | Tier 9 | Tier 9 |
| SOC 2 controls documentation | Tier 9 | Tier 9 |
| Penetration test by third party | Not scheduled | n/a |

---

## Reporting a vulnerability

Email `security@stackwatch.io` (PGP key TBD). We respond within 48h,
fix critical within 7d, and credit the reporter in the changelog unless
asked to remain anonymous.

---

## Hardening checklist for self-hosters

- [x] Bcrypt cost ≥ 10 (we use 12)
- [x] JWT signed (HS256 with env secret)
- [x] TLS at edge (recommended; Caddy provides)
- [x] Database password in env, not code
- [x] Log lines redact sensitive headers (`sanitizeMiddleware`)
- [ ] IP allowlist (set up at proxy level)
- [ ] Periodic key rotation (admin must do manually)
- [ ] Backup of `audit_logs` (you decide retention)
- [ ] Snyk scan on `go mod` deps in CI (not yet in workflow)
- [ ] Static analysis with `gosec` (in `.golangci.yml`, runs in CI)
