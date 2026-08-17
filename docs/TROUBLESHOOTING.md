# StackWatch — Troubleshooting

**Last updated:** 2026-08-17

Issues and fixes pulled from real runbooks on .115.

---

## Quick triage

```bash
# 1. Service status
ssh root@<gateway-host> 'systemctl status api-gateway truenas-connector'
# 2. Logs
ssh root@<gateway-host> 'journalctl -u api-gateway -n 200 --no-pager'
# 3. Database
PGPASSWORD=<pw> psql -h <db-host> -U ios -d ios -c "\d users"
```

---

## "Connection refused" on api-gateway port 8080

**Symptom:** `curl http://<host>:8080/health` returns nothing.

**Likely cause:**
1. Service not running
2. Listening on a different interface (loopback vs 0.0.0.0)
3. Firewall blocking the port

**Fix:**
```bash
# Is it running?
systemctl status api-gateway
# Is it listening on 0.0.0.0?
ss -tlnp | grep api-gateway
# Is the port allowed?
sudo iptables -L INPUT -n | grep 8080
```

---

## "invalid credentials" on /auth/login even with correct password

**Likely cause:**
- DB user/role has been deactivated (`is_active=false`)
- Password was changed without your knowledge (rotated)
- Edge case: bcrypt cost mismatch (we use 12)

**Fix:**
```sql
PGPASSWORD=<pw> psql -h <db> -U ios -d ios \
  -c "UPDATE users SET is_active=true, locked_until=NULL WHERE email='you@example.com';"
```

---

## "JWT signature is invalid" (401 on every protected endpoint)

**Likely cause:**
- `JWT_SECRET` env var differs between deploys, or wasn't set on first run
- Clock skew on multi-replica setup (JWT `exp` is in seconds)

**Fix:**
```bash
# Confirm JWT_SECRET is set
ssh root@<gateway-host> 'env | grep JWT_SECRET'
# Re-login to get fresh token
curl -X POST http://<host>:8080/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"...","password":"..."}'
```

---

## Agent shows "stale" / no heartbeat

**Likely cause:**
- Agent doesn't have the right ingest URL
- Network blocked (firewall, routing)
- Ingest key revoked

**Fix:**
```bash
# On the agent host
sudo systemctl status ios-agent
sudo journalctl -u ios-agent -n 50
sudo cat /opt/ios-agent/.env  # check BACKEND and INGEST_KEY

# Reinstall from dashboard if needed
```

---

## TrueNAS sidecar returns "EBUSY"

**Likely cause:**
- TrueNAS middleware rate limit (too many logins)
- JSON-RPC parse error from filter shape mismatch

**Fix:**
- Wait 30s for the rate-limit cooldown (see `internal/client/truenas/truenas.go`)
- If persistent, restart `truenas-connector` to reset state cache

---

## Database connection refused

**Symptom:**
```
api-gateway.log: pq: password authentication failed for user "ios"
```

**Fix:**
```bash
# Verify password
PGPASSWORD=<pw> psql -h <db> -U ios -d ios -c "SELECT 1"
# Reset if needed (on db host)
sudo -u postgres psql -c "ALTER USER ios WITH PASSWORD '<new-pw>';"
# Update DATABASE_URL env var
sudo systemctl restart api-gateway
```

---

## "Too many open files" (terminal sessions)

**Likely cause:**
- WebSocket sessions leak (PTY not closed)

**Fix:**
```bash
# Increase ulimit
echo "fs.file-max = 100000" | sudo tee -a /etc/sysctl.conf
ulimit -n 65536
# Restart service
sudo systemctl restart web-terminal
```

---

## Verify scripts fail with HTTP 401

**Likely cause:**
- Old token (24h TTL) — get a fresh one
- Wrong password in the script

**Fix:**
```bash
# Most verifier scripts accept login env vars:
LOGIN_EMAIL=you@co LOGIN_PW=TestPass!2026Stack python verifier.py
```

---

## Frontend shows stale data after deploy

**Symptom:**
- Cache-buster `?v=N` in `app.html` < latest version

**Fix:**
- Hard refresh: Ctrl+Shift+R
- Or accept warning on self-signed TLS cert (browser remembers)

---

## Common CLI commands

```bash
# Logs (last 100 lines, no pager)
journalctl -u api-gateway -n 100 --no-pager -o cat

# Config
ssh root@<host> 'env | grep -E "DATABASE_URL|JWT_SECRET|ALLOWED_ORIGINS"'

# Restart cleanly
sudo systemctl restart api-gateway && journalctl -u api-gateway -n 50 --no-pager -f
```
