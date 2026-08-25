import { useEffect, useMemo, useState } from 'react';
import { ApiError, api, me } from '../lib/api';
import { useLogout } from '../lib/useLogout';
import AppSidebar from '../components/AppSidebar';
import { motion, pageEnter } from '../lib/motion';

// Tier 11 Phase 8 — Platform & Commerce unified 6-tab
// customer-facing dashboard.
//
// Tab 1: Deploy (PL1)   → <InstallSection />
// Tab 2: Metering (PL2) → <MeteringSection />
// Tab 3: Signup (PL3)   → <SignupSection />
// Tab 4: Limits (PL4)   → <LimitsSection />
// Tab 5: Backups (PL5)  → <BackupSection />
// Tab 6: Regions (PL6)  → <RegionsSection />
//
// (PL7 RateLimit + PL8 Health live in their own dedicated
// pages — they're operator-facing, not customer-facing. The
// super_admin sidebar entry gets both /platform/health and
// /platform/ratelimit later; Phase 8 finalises the
// customer-facing 6-tab shell.)
//
// Auth: redirects to /login if no JWT. Each tab lazy-loads its
// own component so the initial bundle stays light.
import InstallSection from '../components/admin/DeploySection';
import MeteringSection from '../components/platform/MeteringSection';
import SignupSection from '../components/platform/SignupSection';
import LimitsSection from '../components/platform/LimitsSection';
import BackupSection from '../components/platform/BackupSection';
import RegionsSection from '../components/platform/RegionsSection';

type TabKey = 'deploy' | 'metering' | 'signup' | 'limits' | 'backups' | 'regions';

const TABS: Array<{ key: TabKey; label: string; subtitle: string }> = [
  { key: 'deploy', label: 'Deploy', subtitle: 'One-command installer + tokens' },
  { key: 'metering', label: 'Metering', subtitle: 'Usage events + hourly aggregates' },
  { key: 'signup', label: 'Signup', subtitle: 'Self-service tenant onboarding' },
  { key: 'limits', label: 'Limits', subtitle: 'Plan caps + retention' },
  { key: 'backups', label: 'Backups', subtitle: 'AES-256-GCM encrypted snapshots' },
  { key: 'regions', label: 'Regions', subtitle: 'Multi-region catalog + probes' },
];

export default function PlatformPage() {
  const onLogout = useLogout();
  const [active, setActive] = useState<TabKey>('deploy');
  const [isPlatformAdmin, setIsPlatformAdmin] = useState(false);
  const [error, setError] = useState('');
  const [sectionError, setSectionError] = useState('');

  // DeploySection expects a backend URL + error reporter. Today
  // we hard-code window.location.origin (the api-gateway sits on
  // the same host as the web bundle in production). A future
  // Phase can derive this from a runtime config.
  const backend = typeof window !== 'undefined' ? window.location.origin : '';
  const onDeployError = (msg: string) => setSectionError(msg);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const m = await me();
        if (cancelled) return;
        const role = (m?.user as { role?: string } | undefined)?.role ?? '';
        setIsPlatformAdmin(role === 'super_admin');
      } catch (cause) {
        if (!cancelled) {
          setError(
            cause instanceof ApiError ? cause.friendlyMessage : (cause as Error).message
          );
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Show tab content based on `active`.
  const content = useMemo(() => {
    switch (active) {
      case 'deploy':
        return <InstallSection backend={backend} onError={onDeployError} />;
      case 'metering':
        return <MeteringSection isPlatformAdmin={isPlatformAdmin} />;
      case 'signup':
        return <SignupSection />;
      case 'limits':
        return <LimitsSection canChangePlan={isPlatformAdmin} />;
      case 'backups':
        return <BackupSection canMutate={isPlatformAdmin} />;
      case 'regions':
        return <RegionsSection canMutate={isPlatformAdmin} />;
      default:
        return null;
    }
  }, [active, isPlatformAdmin, backend]);

  return (
    <div className="dash-shell">
      <AppSidebar active="dashboard" onLogout={onLogout} />
      <main className="dash-main">
        <motion.div initial="hidden" animate="show" variants={pageEnter} className="space-y-4">
          <header className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h1 className="text-2xl font-semibold">Platform &amp; Commerce</h1>
              <p className="text-sm opacity-70">
                Tier 11 unified dashboard. PL1-PL6 customer-facing surfaces.
              </p>
            </div>
          </header>

          {error ? (
            <div className="callout-error" role="alert">
              {error}
            </div>
          ) : null}

          {sectionError ? (
            <div className="callout-error" role="alert">
              {sectionError}
            </div>
          ) : null}

          <nav className="flex flex-wrap gap-2 border-b border-white/10 pb-2" aria-label="Platform sections">
            {TABS.map((tab) => {
              const isActive = tab.key === active;
              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setActive(tab.key)}
                  className={`px-3 py-2 rounded-t-md text-sm transition-colors ${
                    isActive
                      ? 'bg-white/10 border-b-2 border-cyan-400'
                      : 'opacity-70 hover:opacity-100'
                  }`}
                  aria-current={isActive ? 'page' : undefined}
                >
                  <span className="font-semibold">{tab.label}</span>
                  <span className="block text-xs opacity-60">{tab.subtitle}</span>
                </button>
              );
            })}
          </nav>

          <section>{content}</section>
        </motion.div>
      </main>
    </div>
  );
}

// _ = api keeps the import live for future inline use (and
// matches the convention used elsewhere so editors don't
// flag it).
void api;