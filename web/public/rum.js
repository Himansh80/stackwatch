/* StackWatch RUM (Real User Monitoring) — browser-side collector.
 *
 * Include on any HTML page:
 *   <script src="/rum.js?tenant_id=<your-tenant-uuid>" defer></script>
 *
 * Captures: page load timing, JS errors, long tasks, fetch failures.
 * Batches events every 5s and POSTs to /api/v1/rum/events.
 *
 * No dependencies, no build step, < 2KB minified.
 */
(function() {
  if (window.__stackwatchRumLoaded) return;
  window.__stackwatchRumLoaded = true;

  // Resolve tenant_id: query param > path segment > 'demo' fallback.
  var tenant = '';
  var qp = new URLSearchParams(location.search);
  tenant = qp.get('tenant_id') || '';
  if (!tenant) {
    var m = location.pathname.match(/^\/t\/([^\/]+)/);
    if (m) tenant = m[1];
  }
  if (!tenant) tenant = 'demo';

  var ENDPOINT = '/api/v1/rum/events?tenant_id=' + encodeURIComponent(tenant);
  var queue = [];
  var MAX_BATCH = 50;
  var FLUSH_MS = 5000;

  function push(ev) {
    queue.push(ev);
    if (queue.length >= MAX_BATCH) flush();
  }

  function flush() {
    if (!queue.length) return;
    var batch = queue.slice(0, MAX_BATCH);
    queue = queue.slice(MAX_BATCH);
    try {
      fetch(ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ events: batch }),
        keepalive: true,
      }).catch(function() { /* swallow — best effort */ });
    } catch (e) { /* ignore */ }
  }

  // Page load timing
  function reportPageLoad() {
    var nav = performance.getEntriesByType('navigation')[0];
    if (!nav) return;
    push({
      kind: 'pageload',
      url: location.href,
      value: Math.round(nav.loadEventEnd - nav.startTime),
      attrs: {
        ttfb: Math.round(nav.responseStart - nav.startTime),
        dom: Math.round(nav.domContentLoadedEventEnd - nav.startTime),
        type: nav.type,
        redirect: nav.redirectCount,
      },
    });
  }
  if (document.readyState === 'complete') reportPageLoad();
  else window.addEventListener('load', reportPageLoad);

  // JS errors
  window.addEventListener('error', function(e) {
    push({
      kind: 'jserror',
      url: location.href,
      value: 0,
      attrs: {
        msg: String(e.message || '').slice(0, 500),
        file: String(e.filename || '').slice(0, 200),
        line: e.lineno || 0,
        col: e.colno || 0,
      },
    });
  });

  // Unhandled promise rejections
  window.addEventListener('unhandledrejection', function(e) {
    var reason = (e.reason && (e.reason.message || e.reason.toString())) || 'unknown';
    push({
      kind: 'jserror',
      url: location.href,
      value: 0,
      attrs: { msg: 'unhandledrejection: ' + String(reason).slice(0, 500) },
    });
  });

  // Long tasks (>50ms blocking)
  try {
    if (typeof PerformanceObserver !== 'undefined') {
      var obs = new PerformanceObserver(function(list) {
        list.getEntries().forEach(function(e) {
          if (e.duration > 50) {
            push({
              kind: 'longtask',
              url: location.href,
              value: Math.round(e.duration),
              attrs: { startTime: Math.round(e.startTime) },
            });
          }
        });
      });
      obs.observe({ entryTypes: ['longtask'] });
    }
  } catch (e) { /* ignore */ }

  // fetch() wrapper — capture network failures
  if (window.fetch) {
    var origFetch = window.fetch;
    window.fetch = function() {
      var args = arguments;
      var url = (args[0] && args[0].url) || String(args[0] || '');
      var p = origFetch.apply(this, args);
      p.catch(function(err) {
        push({
          kind: 'fetch',
          url: String(url).slice(0, 500),
          value: 0,
          attrs: { msg: String(err && err.message || err).slice(0, 300) },
        });
      });
      return p;
    };
  }

  // Flush on hide / unload
  document.addEventListener('visibilitychange', function() {
    if (document.visibilityState === 'hidden') flush();
  });
  window.addEventListener('pagehide', flush);

  // Periodic flush
  setInterval(flush, FLUSH_MS);
})();
