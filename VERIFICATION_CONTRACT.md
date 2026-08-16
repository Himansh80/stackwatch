# StackWatch — VERIFICATION CONTRACT

**Date:** 2026-08-16  
**Goal:** Every claim of "done" must be backed by fresh, live test output.

---

## The 5-step cycle (per feature)

```
1. CODE      → I write the code
2. TESTS     → I write unit + integration tests
3. LIVE      → I run live tests against the running system
4. TESTING.md → I write step-by-step guide for YOU to test
5. WAIT      → I wait for you to test + report errors
```

**NEVER skip step 3.** I never claim "done" without fresh test output in the same turn.

**NEVER skip step 4.** Every feature gets a TESTING.md so you can test it.

**NEVER skip step 5.** I never move to next feature until you confirm.

---

## What "live" means

Every test output must come from a tool call in the SAME turn as the claim.

```
✅ GOOD: "Test passed" — followed by the curl output from this very turn
❌ BAD:  "Test passed" — quoted from a previous turn
❌ BAD:  "Test passed" — assumed without running
```

---

## Where live test output goes

```
tests/
  output/
    2026-08-16_tier-0_health.txt
    2026-08-16_tier-0_login.txt
    2026-08-16_tier-1_proxmox_list.txt
    ...
  scripts/
    live_test.sh
    live_test.py
  CI-LOG.md                    # aggregate of all CI runs
```

Each file named: `YYYY-MM-DD_tier-X_feature.txt`

---

## Test categories (per feature)

### 1. Unit tests (in code)
```go
// internal/auth/jwt_test.go
func TestIssueAndVerifyToken(t *testing.T) {
  // ...
}
```

### 2. Integration tests (in tests/)
```go
// tests/auth_test.go
func TestLoginReturnsToken(t *testing.T) {
  // POST /auth/login -> 200 + JWT
}
```

### 3. Live curl tests (in tests/scripts/)
```bash
#!/bin/bash
# tests/scripts/live_test_tier_0.sh
set -e
TOKEN=$(curl -s -X POST localhost:8080/auth/login -d '{"email":"test@test.com","password":"test"}' | jq -r .token)
[ -n "$TOKEN" ] || { echo "FAIL: no token"; exit 1; }
curl -s -H "Authorization: Bearer $TOKEN" localhost:8080/auth/me | grep '"email"'
echo "OK"
```

### 4. Live verifier (full harness)
```bash
# tests/scripts/verify_tier_0.sh
# Runs all tier 0 checks, prints PASS/FAIL summary
```

### 5. TESTING.md (manual steps for user)
```markdown
# How to test Tier 0

## Step 1: Health check
Open: http://localhost:8080/health
Expected: `{"status":"ok"}`

## Step 2: Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test"}'
Expected: 200 with `{"token":"...","user":{"id":"..."}}`

## Step 3: Get profile
curl -H "Authorization: Bearer <token>" http://localhost:8080/auth/me
Expected: 200 with user object
```

---

## What goes wrong if I skip verification

User has been burned multiple times. Examples:
- "Tier 5 done" — actually only 4 of 12 endpoints worked
- "Mobile app shipped" — actually never built
- "Auth works" — actually JWT expired on first request
- "DB migration applied" — actually failed silently

** Every claim must be backed by fresh output. **

---

## Verifier scripts (must be re-runnable)

Every verifier must be:
- Re-runnable (deterministic output)
- Self-contained (no external deps beyond the system under test)
- Idempotent (creates test data, doesn't require manual cleanup)
- Fast (< 30 seconds for a single tier)

Pattern saved as: `tests/verify_tier_<N>.sh` (or .py)

---

## Reporting test results

When I report test results, I MUST include:

```
✅ Tier 0 — Authentication (X/Y PASS)
1. POST /auth/login returns 200 + JWT — PASS
2. POST /auth/login with bad password returns 401 — PASS
3. GET /auth/me with valid JWT returns 200 — PASS
4. GET /auth/me without JWT returns 401 — PASS
... (X more)

Verifier: tests/scripts/verify_tier_0.sh
Output: tests/output/2026-08-16_tier-0_auth.txt
```

The number X/Y must be from THIS turn's run, not a previous one.

---

## When I cannot run tests

Sometimes I can't run live tests (e.g., the server is down, network issue). When this happens:

❌ NEVER claim "should work" or "probably fine"
✅ ALWAYS say: "I could not run live tests because [reason]. Here's what I verified instead: [static checks, code review, unit tests in isolation]. Live test will run when [condition]."

---

## User testing protocol

After I deliver a feature with TESTING.md:

1. **You run the steps** in TESTING.md
2. **You report errors** (verbatim with stack traces if any)
3. **I fix ONLY what you reported** (no scope creep)
4. **You re-test** the fix
5. **You say "next"** only when satisfied

If you say "looks good but the dashboard is broken" — I fix the dashboard, not the whole feature.

---

## Daily verification protocol

Every session ends with:
1. `tests/verify_tier_<N>.sh` runs green
2. `tests/output/YYYY-MM-DD_*.txt` files attached
3. CI shows green
4. `journal.md` updated with what was tested

If the daily verification fails, the session is not "done" — even if I built everything.

---

## The "no claiming done" rule

I will never say "done" unless:
- ✅ All tests pass in this turn
- ✅ TESTING.md exists for this feature
- ✅ Live output captured
- ✅ User has tested and confirmed

"Code complete" ≠ "done"
"Builds" ≠ "done"
"Tests pass" ≠ "done" (until user confirms)

---

## Failure mode of the verification

If I say "done" and there are bugs, the user gets frustrated. To prevent this:

1. **Run verifiers BEFORE claiming done** — not after
2. **Re-run verifiers if too much time has passed** — last turn's output is stale
3. **Run verifiers 4+ times for non-trivial features** — Datadog style
4. **Never trust output from a previous turn** — always fresh

---

## Reference

- `CI-LOG.md` — all CI runs
- `tests/output/` — all live test outputs
- `tests/scripts/` — all verifier scripts
- `tests/TESTING.md` — user-facing test instructions
- `journal.md` — what was tested each session
- `errors.md` — test failures + fixes
