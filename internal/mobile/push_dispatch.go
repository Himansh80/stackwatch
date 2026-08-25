// Tier 13 Phase 1 — Mobile push dispatch worker.
//
// PushDispatcher polls alerts + push_devices every 30s and queues
// pushes via FCM (Android) or APNs (iOS, Phase 2 stub). One tick =
// one full pass; per-device errors are logged but never crash.
//
// Phase 1 behavior:
//   - Log that the worker started
//   - On each tick, log a single line indicating FCM key state
//   - Phase 2 will implement the actual query + dispatch via
//     push_stack.go's dispatchToFCM / dispatchToAPNs
//
// Why Phase 1 is intentionally a stub: the dispatch worker needs
// FCM_SERVER_KEY (Firebase project setup) and at least one registered
// device. The system can boot + serve Phase 2's REST API without
// either; the worker just logs "FCM_SERVER_KEY unset" until the
// operator wires FCM.
package mobile

import (
	"context"
	"log"
	"time"

	"github.com/stackwatch/platform/internal/db"
)

// PushDispatcher polls push targets every 30s. Phase 1 is a stub that
// just logs each tick; Phase 2 fills in the actual query + dispatch.
type PushDispatcher struct {
	pool *db.Pool
	tick time.Duration
}

// New returns a dispatcher. Call .Start on api-gateway boot.
func New(p *db.Pool) *PushDispatcher {
	return &PushDispatcher{
		pool: p,
		tick: 30 * time.Second,
	}
}

// Start launches the polling goroutine. The goroutine runs one tick
// immediately, then ticks on the configured interval. Exits cleanly on
// context cancellation.
func (d *PushDispatcher) Start(ctx context.Context) {
	if d.pool == nil {
		log.Printf("[push_dispatcher] pool is nil, not starting")
		return
	}
	t := time.NewTicker(d.tick)
	go func() {
		// Run once immediately, then on each tick
		d.tickOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				d.tickOnce(ctx)
			}
		}
	}()
	log.Printf("[push_dispatcher] started (tick=%s)", d.tick)
}

// tickOnce runs one dispatch pass. Phase 1 stub: just log state.
func (d *PushDispatcher) tickOnce(ctx context.Context) {
	// Phase 2 will:
	//   1. SELECT alerts WHERE status='open' AND NOT EXISTS push_log.alert_id
	//   2. For each alert, JOIN push_devices WHERE tenant_id=? AND user_id=?
	//   3. Build fcmMessage per device, dispatchToFCM
	//   4. INSERT push_log.status='sent' or 'skipped' or 'failed'
	//
	// For now, just emit a single log line per tick so the operator can
	// verify the worker is alive via journalctl.
	log.Printf("[push_dispatcher] tick: Phase 1 stub — no pushes dispatched (FCM wiring deferred to Phase 2)")
	_ = ctx
}
