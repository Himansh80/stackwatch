# StackWatch — Errors

> Append-only. Every error encountered, full detail, fix.

---

## 2026-08-16 — Tier 0 bugs found during verification

### E1: JWT tokens expired in 11 seconds
- **Symptom:** `GET /auth/me` returned 401 immediately after `POST /auth/login` (same token)
- **Investigation:** Decoded JWT payload, saw `exp` was ~11s from `iat`
- **Root cause:** `db.go` passed `24*60*60` (number 86400) to `auth.NewIssuer(secret, ttl time.Duration)`. JWT lib's `jwt.NewNumericDate(now.Add(ttl))` interprets `ttl` as a `time.Duration` (nanoseconds), so 86400 nanoseconds = 86.4 microseconds from `now` — token expired almost immediately.
- **Fix:** Changed to `auth.NewIssuer(secret, 24*time.Hour)` and imported `time` in `cmd/api-gateway/db.go`
- **Lesson:** When passing a duration literal, ALWAYS use `time.Hour`, `time.Minute`, etc. Never raw integers.

### E2: Bearer token prefix parse mismatch
- **Symptom:** Same 401 as E1, even after JWT was fixed
- **Investigation:** Compared `Authorization: Bearer eyJ...` length (8 chars minimum for "Bearer ") vs handler check (`h[:7]`)
- **Root cause:** Handler used `if len(h) < 7 || h[:7] != "Bearer "` — `len("Bearer ")` is 7, but the prefix check needs length 7+ for `h[:7]`. The `< 7` was off-by-one.
- **Fix:** Changed to `if len(h) < 8 || !strings.HasPrefix(h, "Bearer ")` and `token := strings.TrimPrefix(h, "Bearer ")`
- **Lesson:** For string prefix matching, use `strings.HasPrefix` not manual slicing — fewer bugs.

### E3: ChangePassword rejected valid old passwords
- **Symptom:** With a fresh valid JWT, `POST /auth/change-password` returned 401 even with the correct old password
- **Investigation:** Looked at handler — `userFromContext(c)` returned a User constructed from JWT claims (UserID, TenantID, Role, Email). JWT claims DO NOT include `password_hash`. So `u.PasswordHash = ""`. `checkHash("", "hunter22")` always returns false.
- **Root cause:** Handler used JWT-only user instead of fetching from DB.
- **Fix:** `ChangePassword` now calls `h.lookupUserAndTenant(c.Request.Context(), claimsUser.Email)` to fetch the full user record before checking the password.
- **Lesson:** Never use JWT claims as a substitute for DB lookup when you need password_hash or other private fields. JWT is for authentication, not data hydration.

### E4: Duplicate signup returned 500
- **Symptom:** Second signup with the same email returned HTTP 500
- **Investigation:** Postgres throws error code `23505` (unique_violation) on duplicate email, which propagated as a generic 500
- **Fix:** Added `isUniqueViolation(err error) bool` helper using `pgconn.PgError` type assertion. Returns `true` if `pgErr.Code == "23505"`. Both `tenants` and `users` INSERT handlers now check and return 409.
- **Lesson:** Always classify DB errors. Generic 500s are useless to API clients.

### E5: Security header typo
- **Symptom:** Verifier test for security headers failed
- **Investigation:** Header was `X-Frame-Options: DENI` (typo). Standard is `DENY`.
- **Fix:** Changed middleware to set `DENY`.
- **Lesson:** Verifier tests catch typos the linter misses. The verifier is part of the spec — write tests for header values too.

### E6: Background gateway restart via SSH `nohup &` didn't survive
- **Symptom:** Gateway died after SSH session closed. `nohup ... & disown` left the process orphaned but somehow it still died.
- **Root cause:** When the SSH session disconnects, the controlling TTY goes away and some shells kill the process group despite `nohup`.
- **Fix:** Use `setsid sh -c '... > log 2>&1' </dev/null >/dev/null 2>&1 &` — this creates a new session detached from the parent.
- **Lesson:** Always use `setsid` for processes that must outlive the SSH session.

## 2026-08-18 — Tier 0 verifier launch quoting error

### E7: Remote verifier command was truncated by nested Bash quoting
- **Symptom:** The verifier did not reach the API; Git Bash returned `unexpected EOF while looking for matching \`'\``.
- **Root cause:** A base64 Python payload was embedded inside multiple nested single-quote layers (`terminal` → Bash → SSH → remote Bash → Python).
- **Impact:** No remote application state changed; the failure occurred before the SSH command executed.
- **Recovery:** Rerun using a transport that avoids nested shell quoting and then verify the live gateway.

## 2026-08-18 — Tier 0 live A-to-Z verification

### E8: Running binary is behind the source route table
- **Symptom:** Local and remote source `routes.go` include profile, tenant-create, magic-link, accept-invite, and API-key-delete routes, but the running binary returns 404 for those endpoints.
- **Evidence:** `/opt/stackwatch/bin/api-gateway-linux` and `/proc/<pid>/exe` both report md5 `99d39c8662fc8ca20617cef52c432c49`; live probes repeatedly returned 404.
- **Impact:** The deployed system is not equivalent to the current source tree.
- **Recovery:** Rebuild the canonical source and deploy only after the complete Tier 0 verifier passes.

### E9: Tier 0 live verifier found deterministic functional failures
- **Symptom:** Four consecutive full runs returned `56/68 PASS`.
- **Failures:** `PATCH /auth/profile` 404; `POST /tenants` 404; `POST /auth/magic-link` 404; `POST /auth/accept-invite` 404; `DELETE /api-keys/:id` 404; forgot-password returns no usable dev token; reset with an invalid token returns HTTP 200; brute-force audit event count remains 0.
- **Impact:** Tier 0 is not end-to-end complete on the running deployment.
- **Recovery:** Do not claim Tier 0 complete until the live binary is rebuilt/deployed and the same verifier returns 68/68 across four runs.
