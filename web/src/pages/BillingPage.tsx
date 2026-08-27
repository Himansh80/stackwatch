import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError, api, clearToken, getToken } from '../lib/api';
import { motion, pageEnter } from '../lib/motion';
import StatusPill from '../components/shared/StatusPill';

type Tenant = {
  id: string;
  name: string;
  slug: string;
  plan: string;
  status: string;
  created_at: string;
  updated_at: string;
};

const PLANS = [
  {
    id: 'free',
    name: 'Free',
    price: '$0',
    cadence: 'forever',
    tagline: 'For solo operators and homelabs.',
    perks: [
      'Up to 3 connected servers',
      '1 user per workspace',
      '7-day metric retention',
      'Community support',
    ],
  },
  {
    id: 'pro',
    name: 'Pro',
    price: '$8',
    cadence: 'per host per month',
    tagline: 'For production infrastructure.',
    perks: [
      'Unlimited connected servers',
      '30-day metric retention',
      'Email + chat support',
      'Custom alert rules',
    ],
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    price: 'Custom',
    cadence: 'self-hosted or cloud',
    tagline: 'For multi-tenant operators and SRE teams.',
    perks: [
      'Single sign-on (SAML / OIDC)',
      'Audit log retention',
      'Priority support',
      'Custom integrations',
    ],
  },
];

export default function BillingPage() {
  const nav = useNavigate();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!getToken()) {
        setError('You are not signed in.');
        setLoading(false);
        return;
      }
      try {
        const me = await api<{ tenant?: Tenant }>('GET', '/api/v1/auth/me');
        if (cancelled) return;
        setTenant(me.tenant ?? null);
      } catch (cause) {
        if (cancelled) return;
        if (cause instanceof ApiError) {
          setError(cause.friendlyMessage);
        } else {
          setError(cause instanceof Error ? cause.message : 'Unable to load your billing info.');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  function logout() {
    clearToken();
    nav('/login');
  }

  const currentPlanId = (tenant?.plan || 'free').toLowerCase();

  return (
        <motion.div
          className="dash-content"
          initial="hidden"
          animate="show"
          variants={pageEnter}
        >
          {error && (
            <div className="dash-error">
              <strong>Could not load your billing info</strong>
              <span>{error}</span>
              <button onClick={() => window.location.reload()}>Retry</button>
            </div>
          )}
          {loading ? (
            <section className="dash-panel"><div className="dash-panel-empty"><strong>Loading billing…</strong></div></section>
          ) : (
            <>
              <section className="dash-panel">
                <div className="dash-panel-head">
                  <div><span className="dash-eyebrow">Current plan</span><h3>Workspace subscription</h3></div>
                  <StatusPill status={(tenant?.status === 'active' || tenant?.status === 'suspended' || tenant?.status === 'trialing') ? tenant.status : 'unknown'} label={tenant?.status || 'active'} size="sm" />
                </div>
                <div className="dash-resource-grid">
                  <div className="dash-resource-card">
                    <div className="dash-resource-head"><span className="dash-resource-type">Plan</span></div>
                    <strong style={{ textTransform: 'capitalize' }}>{tenant?.plan || 'free'}</strong>
                    <div className="dash-resource-meta"><span>workspace <b>{tenant?.name || '—'}</b></span></div>
                  </div>
                  <div className="dash-resource-card">
                    <div className="dash-resource-head"><span className="dash-resource-type">Tenant id</span></div>
                    <strong>{tenant?.id ? tenant.id.slice(0, 8) + '…' : '—'}</strong>
                    <div className="dash-resource-meta"><span>slug <b>{tenant?.slug || '—'}</b></span></div>
                  </div>
                  <div className="dash-resource-card">
                    <div className="dash-resource-head"><span className="dash-resource-type">Member since</span></div>
                    <strong>{tenant?.created_at ? new Date(tenant.created_at).toLocaleDateString() : '—'}</strong>
                    <div className="dash-resource-meta"><span>updated <b>{tenant?.updated_at ? new Date(tenant.updated_at).toLocaleDateString() : '—'}</b></span></div>
                  </div>
                </div>
              </section>

              <section className="dash-section">
                <span className="dash-eyebrow">Available plans</span>
                <h2 className="dash-section-title">Choose how StackWatch runs for you.</h2>
                <p className="dash-section-lede">
                  All plans include the full dashboard, alert engine, and terminal. Higher tiers unlock longer metric retention, more connected servers, and SSO.
                </p>
                <div className="dash-pricing">
                  {PLANS.map((plan) => (
                    <article key={plan.id} className={`dash-plan ${currentPlanId === plan.id ? 'dash-plan-current' : ''}`}>
                      {currentPlanId === plan.id && <span className="dash-plan-badge">Current plan</span>}
                      <header className="dash-plan-head">
                        <strong>{plan.name}</strong>
                        <span className="dash-plan-price">{plan.price}</span>
                        <small>{plan.cadence}</small>
                      </header>
                      <p className="dash-plan-tag">{plan.tagline}</p>
                      <ul className="dash-plan-perks">
                        {plan.perks.map((perk) => <li key={perk}>{perk}</li>)}
                      </ul>
                      <button
                        type="button"
                        className={`sw-button ${currentPlanId === plan.id ? 'sw-button-quiet' : 'sw-button-primary'}`}
                        disabled={currentPlanId === plan.id}
                        title="Payment provider coming soon"
                      >
                        {currentPlanId === plan.id ? 'You are on this plan' : 'Switch to this plan'}
                      </button>
                    </article>
                  ))}
                </div>
              </section>

              <p className="dash-foot-note">Payment provider integration (Stripe / Razorpay) is on the roadmap. The plan shown above is your workspace&apos;s current tier.</p>
            </>
          )}
        </motion.div>
  );
}
