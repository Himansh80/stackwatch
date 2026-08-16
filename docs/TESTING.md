# StackWatch — Tier 0 Testing Guide

**For:** Himan (the user)
**Date:** 2026-08-16
**Status:** Tier 0 foundation shipped. 18/18 live tests pass on http://192.168.0.115:8080

---

## What you can test RIGHT NOW

### 1. Health check (open in browser)

```
http://192.168.0.115:8080/health
```

Expected:
```json
{"db":"ok","status":"ok","version":"0.1.0-tier0"}
```

### 2. Sign up a new account

```bash
curl -X POST http://192.168.0.115:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"YOUR@EMAIL.COM","password":"YOUR_STRONG_PASSWORD","full_name":"Your Name","tenant_name":"Your Org"}'
```

Expected: 201 + JSON with `token`, `user`, `tenant`.

### 3. Log in

```bash
curl -X POST http://192.168.0.115:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"YOUR@EMAIL.COM","password":"YOUR_STRONG_PASSWORD"}'
```

Expected: 200 + token.

### 4. Get your profile (token required)

```bash
TOKEN="paste-token-here"
curl -H "Authorization: Bearer $TOKEN" http://192.168.0.115:8080/api/v1/auth/me
```

Expected: 200 + `{user: {...}, tenant: {...}}`.

### 5. Change your password

```bash
curl -X POST http://192.168.0.115:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"YOUR_OLD","new_password":"YOUR_NEW_STRONG"}'
```

Expected: 200 + `{ok: true}`. Then login with the new password works, old password fails.

### 6. Log out

```bash
curl -X POST http://192.168.0.115:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

Expected: 200. (JWT is stateless — server just returns ok. Client discards the token.)

### 7. Test failure paths

| Test | Expected |
|------|----------|
| Bad email format | 400 |
| Short password (<8) | 400 |
| Wrong password | 401 |
| Non-existent email | 401 |
| No Authorization header | 401 |
| Bad/expired JWT | 401 |
| Duplicate email signup | 409 |
| Unknown route | 404 |
| OPTIONS preflight | 204 |

### 8. Test security headers

```bash
curl -I http://192.168.0.115:8080/health
```

Expected headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `X-Request-ID: <uuid>`

### 9. Verify the database

```bash
ssh root@192.168.0.116 "PGPASSWORD=3d5cb43fba1f82283a2ba02c79e116cf psql -h 127.0.0.1 -U ios -d ios -c '\dt'"
```

Expected: 4 tables — `api_keys`, `audit_log`, `tenants`, `users`.

### 10. View live logs

```bash
ssh root@192.168.0.115 "tail -f /opt/stackwatch/logs/api-gateway.log"
```

Every request logged as structured JSON with request_id, method, path, status, duration.

---

## Automated verifier

Run from Windows:
```bash
python "C:/Users/himan/AppData/Local/Temp/hermes-verify-tier0-2026-08-16.py"
```

**Latest result:** 18/18 PASS (verified 4 consecutive runs).

---

## What I need from you

After you've tested, tell me one of:

1. **"next"** — move to Tier 1 (Proxmox full replacement)
2. **"fix X"** — describe the issue
3. **"change Y"** — describe the change

I'll fix or move on. No claim of "done" until you say it works.

---

## Known gaps (Tier 0 only — all addressed in later tiers)

- **No password reset email** — `/forgot-password` returns ok but sends nothing (Tier 1.6 will wire Resend)
- **No 2FA** — Tier 9.6
- **No SAML/SSO** — Tier 9.2
- **Frontend is minimal** — Dashboard + Login only; 17 more pages ship with their tiers
- **No agents / metrics / alerts** — Tier 1+
