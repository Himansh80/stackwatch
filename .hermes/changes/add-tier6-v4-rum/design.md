# Design: Tier 6 v4 — Real User Monitoring (M8)

## Approach (minimal viable)

Three endpoints + one JS snippet. The JS snippet:
1. Loads on any page where you include `<script src="/rum.js">`
2. Captures: page load timing, JS errors (window.onerror), long tasks (>50ms), fetch/XHR failures
3. Batches events (every 5s or 50 events, whichever first) and POSTs to /api/v1/rum/events

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Tiny JS snippet (~80 LOC, no deps) | Self-contained, no framework lock-in | Less features than commercial RUM |
| Auth: tenant_id passed as query param from snippet | Avoids CORS preflight for simple token | Less secure than JWT; rate-limited |
| One `rum_events` table with JSONB attrs | Flexible — different kinds have different shapes | Can't index by specific attrs |
| No session replay (out of scope) | Privacy + size concerns | Big differentiator vs Datadog |

## Implementation details

### Migration `011_rum_events.sql`
```sql
CREATE TABLE rum_events (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,    -- 'pageload' | 'jserror' | 'longtask' | 'xhr' | 'fetch'
    url TEXT,
    value DOUBLE PRECISION,  -- metric value (load time ms, task duration, etc.)
    attrs JSONB DEFAULT '{}'::jsonb,  -- kind-specific (stack trace, status code, etc.)
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rum_tenant_ts ON rum_events (tenant_id, ts DESC);
CREATE INDEX idx_rum_kind_ts ON rum_events (kind, ts DESC);
```

### Endpoints (`internal/handler/rum.go`)

- `POST /api/v1/rum/events` — accept array of events, batch insert (auth via tenant_id query param)
- `GET /api/v1/rum/summary` — aggregates: count by kind, avg value by kind (last 24h)
- `GET /api/v1/rum/events` — recent events for debugging

### JS snippet (`web/public/rum.js`)
```js
(function() {
  var tenant = location.pathname.match(/^\/t\/([^\/]+)/);
  tenant = tenant ? tenant[1] : 'demo';
  var queue = [];
  function flush() { if (!queue.length) return; var events = queue; queue = []; fetch('/api/v1/rum/events?tenant_id='+tenant, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({events: events}), keepalive:true}); }
  setInterval(flush, 5000);
  // page load timing
  var nav = performance.getEntriesByType('navigation')[0];
  if (nav) queue.push({kind:'pageload', url: location.href, value: nav.loadEventEnd-nav.startTime, attrs:{ttfb: nav.responseStart-nav.startTime, dom: nav.domContentLoadedEventEnd-nav.startTime}});
  // js errors
  window.addEventListener('error', function(e) { queue.push({kind:'jserror', url: location.href, value: 0, attrs:{msg: e.message, file: e.filename, line: e.lineno}}); });
  // long tasks
  try { var obs = new PerformanceObserver(function(list) { list.getEntries().forEach(function(e){if (e.duration > 50) queue.push({kind:'longtask', url: location.href, value: e.duration}); }); }); obs.observe({entryTypes:['longtask']}); } catch(e){}
  // fetch / xhr failures
  var origFetch = window.fetch;
  window.fetch = function() { var p = origFetch.apply(this, arguments); p.catch(function(e){queue.push({kind:'fetch', url: arguments[0], value: 0, attrs:{msg: String(e)}}); }); return p; };
})();
```

## Risks
- **R1: tenant_id from URL is spoofable** for unauthenticated ingestion. Mitigation: rate-limit + max payload size (10KB) + per-IP throttling.
- **R2: JSONB attrs can be huge** (stack traces). Mitigation: limit attrs to 4KB; truncate with marker.
- **R3: JS errors capture sensitive data** (URL params, etc.). Mitigation: scrub `?token=...` patterns before sending.
