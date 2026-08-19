import { Link } from 'react-router-dom';

const features = [
  {
    icon: '⌁',
    title: 'Unified infrastructure view',
    body: 'See Proxmox, TrueNAS, and bare-metal workloads on one canvas — no more flipping between five tabs to find the slow VM.',
  },
  {
    icon: '◫',
    title: 'Self-hosted by default',
    body: 'Your servers, your data, your audit trail. Run on your own hardware with no third-party telemetry leaving the building.',
  },
  {
    icon: '▤',
    title: 'Drill into any layer',
    body: 'Click a node to see its VMs, containers, pools, and services. Click a workload to see live CPU, memory, and disk pressure.',
  },
  {
    icon: '⌘',
    title: 'Browser terminal built in',
    body: 'Open a real shell to any registered server from the dashboard — no SSH client required.',
  },
  {
    icon: '△',
    title: 'Alerts that respect context',
    body: 'See firing alerts with the affected workload right beside the rule that triggered them. Acknowledge or resolve in one click.',
  },
  {
    icon: '◇',
    title: 'Built for small teams',
    body: 'Designed for operators who own five servers and want them all in one place, not enterprise teams with five hundred dashboards.',
  },
];

const steps = [
  { n: '01', title: 'Register a host', body: 'Add the API token for your Proxmox or TrueNAS control plane. StackWatch pulls the topology automatically.' },
  { n: '02', title: 'See it on the canvas', body: 'Your nodes, VMs, containers, and pools appear in seconds. Health and uptime update in real time.' },
  { n: '03', title: 'Move from signal to action', body: 'Open a terminal, restart a workload, or check logs without leaving the dashboard.' },
];

export default function Landing() {
  return (
    <div className="landing-app">
      <header className="landing-topbar">
        <Link className="landing-brand" to="/">
          <span className="landing-brand-mark">S</span>
          <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
        </Link>
        <nav className="landing-nav">
          <a href="#features">Features</a>
          <a href="#how">How it works</a>
          <a href="#pricing">Pricing</a>
          <a href="https://github.com/" target="_blank" rel="noreferrer">GitHub</a>
        </nav>
        <div className="landing-top-actions">
          <Link className="landing-button landing-button-ghost" to="/login">Sign in</Link>
          <Link className="landing-button landing-button-primary" to="/signup">Sign up free</Link>
        </div>
      </header>

      <main>
        <section className="landing-hero">
          <div className="landing-hero-text">
            <span className="landing-eyebrow">Self-hosted · Open · Yours</span>
            <h1>Run your infrastructure from <span className="landing-gradient">one calm screen</span>.</h1>
            <p className="landing-lede">
              StackWatch is the dashboard for operators who would rather own their monitoring than rent it.
              Register a Proxmox or TrueNAS control plane and see your entire fleet — nodes, workloads, alerts,
              terminals — in a single place, on hardware you control.
            </p>
            <div className="landing-hero-actions">
              <Link className="landing-button landing-button-primary landing-button-large" to="/signup">Create a free account</Link>
              <Link className="landing-button landing-button-ghost landing-button-large" to="/login">I already have one</Link>
            </div>
            <div className="landing-hero-meta">
              <span><span className="landing-meta-dot" />No credit card</span>
              <span><span className="landing-meta-dot" />Free tier for 1 host</span>
              <span><span className="landing-meta-dot" />Open source</span>
            </div>
          </div>
          <aside className="landing-hero-card" aria-label="Dashboard preview">
            <header className="landing-hero-card-head">
              <span className="landing-eyebrow">Command center</span>
              <span className="landing-hero-status"><span className="landing-live-dot" />Live</span>
            </header>
            <div className="landing-hero-kpis">
              <div><small>Connected hosts</small><strong>1</strong></div>
              <div><small>Compute nodes</small><strong>1</strong></div>
              <div><small>Running workloads</small><strong>15</strong></div>
              <div><small>API health</small><strong className="landing-good">ok</strong></div>
            </div>
            <div className="landing-hero-rows">
              <div><span className="landing-row-dot landing-row-good" />router <em>online</em></div>
              <div><span className="landing-row-dot landing-row-good" />pf-sense <em>running · 4% cpu</em></div>
              <div><span className="landing-row-dot landing-row-good" />adguard <em>running · 0% cpu</em></div>
              <div><span className="landing-row-dot landing-row-warn" />portfolio <em>running · 62% cpu</em></div>
            </div>
            <div className="landing-hero-foot">Live snapshot · updated seconds ago</div>
          </aside>
        </section>

        <section className="landing-section" id="features">
          <span className="landing-eyebrow">What you get</span>
          <h2>Everything a small infra team actually uses.</h2>
          <p className="landing-section-lede">
            No 200-feature checklist you&apos;ll never open. The dashboard gives you the things you reach for every day,
            built to feel fast on a five-server homelab and a fifty-server production cluster alike.
          </p>
          <div className="landing-feature-grid">
            {features.map((feature) => (
              <article className="landing-feature" key={feature.title}>
                <span className="landing-feature-icon">{feature.icon}</span>
                <strong>{feature.title}</strong>
                <p>{feature.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section landing-section-alt" id="how">
          <span className="landing-eyebrow">How it works</span>
          <h2>Three steps from signup to live infra.</h2>
          <ol className="landing-steps">
            {steps.map((step) => (
              <li key={step.n}>
                <span className="landing-step-num">{step.n}</span>
                <div>
                  <strong>{step.title}</strong>
                  <p>{step.body}</p>
                </div>
              </li>
            ))}
          </ol>
        </section>

        <section className="landing-section" id="pricing">
          <span className="landing-eyebrow">Pricing</span>
          <h2>Free for personal use. Honest pricing for teams.</h2>
          <p className="landing-section-lede">
            The full product is free to run on your own hardware. Cloud billing covers the cost of running it for you.
          </p>
          <div className="landing-pricing">
            <article className="landing-plan">
              <header><strong>Self-hosted</strong><span className="landing-price">Free</span><small>Forever, on your own hardware</small></header>
              <ul>
                <li>Unlimited hosts and workloads</li>
                <li>Full command center + terminal</li>
                <li>All alert rules and integrations</li>
                <li>Source-available, audit your own code</li>
              </ul>
              <Link className="landing-button landing-button-ghost" to="/signup">Get the binary</Link>
            </article>
            <article className="landing-plan landing-plan-featured">
              <span className="landing-plan-badge">Most popular</span>
              <header><strong>Cloud free</strong><span className="landing-price">$0</span><small>Up to 1 host, forever</small></header>
              <ul>
                <li>1 connected host, unlimited workloads</li>
                <li>Hosted control plane at stackwatch.smarthomelab.fun</li>
                <li>All dashboards, alerts, terminals</li>
                <li>Community support</li>
              </ul>
              <Link className="landing-button landing-button-primary" to="/signup">Start free</Link>
            </article>
            <article className="landing-plan">
              <header><strong>Cloud team</strong><span className="landing-price">$8<span>/host/mo</span></span><small>For teams running more than one host</small></header>
              <ul>
                <li>Unlimited hosts and workloads</li>
                <li>SAML SSO, role-based access</li>
                <li>Audit log retention, priority support</li>
                <li>Email and chat onboarding</li>
              </ul>
              <Link className="landing-button landing-button-ghost" to="/signup">Start 14-day trial</Link>
            </article>
          </div>
        </section>

        <section className="landing-cta">
          <div>
            <h2>Ready to see your fleet in one place?</h2>
            <p>Create a free account in under a minute. No credit card, no trial countdown.</p>
          </div>
          <div className="landing-cta-actions">
            <Link className="landing-button landing-button-primary landing-button-large" to="/signup">Create free account</Link>
            <Link className="landing-button landing-button-ghost landing-button-large" to="/login">Sign in</Link>
          </div>
        </section>
      </main>

      <footer className="landing-footer">
        <div className="landing-footer-grid">
          <div>
            <Link className="landing-brand" to="/">
              <span className="landing-brand-mark">S</span>
              <span><strong>StackWatch</strong><small>Self-hosted infrastructure platform</small></span>
            </Link>
            <p className="landing-footer-tag">Own your monitoring. StackWatch is source-available and runs on your own hardware.</p>
          </div>
          <div>
            <strong>Product</strong>
            <a href="#features">Features</a>
            <a href="#how">How it works</a>
            <a href="#pricing">Pricing</a>
          </div>
          <div>
            <strong>Company</strong>
            <a href="#">About</a>
            <a href="#">Blog</a>
            <a href="https://github.com/" target="_blank" rel="noreferrer">GitHub</a>
          </div>
          <div>
            <strong>Get started</strong>
            <Link to="/signup">Create account</Link>
            <Link to="/login">Sign in</Link>
            <a href="#">Contact</a>
          </div>
        </div>
        <div className="landing-footer-bottom">
          <span>© 2026 SmartHomeLab. Built in Bareilly, India.</span>
          <span>Self-hosted · No tracking · No vendor lock-in</span>
        </div>
      </footer>
    </div>
  );
}
