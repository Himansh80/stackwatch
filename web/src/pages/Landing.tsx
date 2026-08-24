import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  motion,
  useReducedMotion,
  type Variants,
} from 'framer-motion';

// ---------- Demo data (unchanged) --------------------------------------
const NODE_NAMES = ['edge-01', 'core-a', 'core-b', 'mgmt-01', 'gpu-02', 'storage-1', 'hypervisor-eu', 'worker-3', 'bastion', 'observability-1', 'db-primary', 'cache-01', 'ci-runner-2'];
const WORKLOAD_NAMES = [
  'web-api', 'auth-svc', 'payments', 'image-worker', 'search-indexer',
  'notification-fanout', 'realtime-gateway', 'job-runner', 'metrics-pipeline',
  'log-shipper', 'backup-agent', 'metrics-api', 'cdn-orchestrator',
  'analytics-etl', 'queue-worker', 'session-store', 'feature-flags',
  'rate-limiter', 'audit-writer', 'webhook-router',
];
const TYPES = ['qemu', 'lxc', 'container', 'vm'];
const STATUSES_GOOD = ['running', 'online', 'healthy'];
const STATUSES_WARN = ['degraded', 'high-cpu', 'high-mem'];
const STATUSES_BAD = ['stopped', 'offline', 'crashed'];

function pick<T>(arr: T[], seed: () => number): T {
  return arr[Math.floor(seed() * arr.length)]!;
}

function makeSeed(): () => number {
  let state = Math.floor(Math.random() * 0x7fffffff);
  if (state === 0) state = 1;
  return () => {
    state = (state * 48271) % 0x7fffffff;
    return state / 0x7fffffff;
  };
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

function generatePreview() {
  const rand = makeSeed();
  const hosts = 1 + Math.floor(rand() * 6);
  const nodes = hosts * (1 + Math.floor(rand() * 3));
  const totalWorkloads = 4 + Math.floor(rand() * 28);
  const runningCount = Math.max(1, totalWorkloads - Math.floor(rand() * 4));
  const rows = Array.from({ length: 4 }).map((_, idx) => {
    const statusBucket = rand();
    let tone: 'good' | 'warn' | 'bad';
    let status: string;
    if (statusBucket < 0.7) {
      tone = 'good';
      status = pick(STATUSES_GOOD, rand);
    } else if (statusBucket < 0.9) {
      tone = 'warn';
      status = pick(STATUSES_WARN, rand);
    } else {
      tone = 'bad';
      status = pick(STATUSES_BAD, rand);
    }
    const cpu = Math.floor(rand() * 100);
    const mem = Math.floor(50 + rand() * 900);
    const name = idx === 0 ? pick(NODE_NAMES, rand) : pick(WORKLOAD_NAMES, rand);
    const type = pick(TYPES, rand);
    return {
      name,
      type,
      tone,
      status,
      detail: tone === 'good'
        ? `${status} · ${cpu}% cpu`
        : tone === 'warn'
          ? `${status} · ${cpu}% cpu`
          : `${status}`,
      mem: tone === 'good' ? `${mem} MB` : null,
    };
  });
  const stamp = `${pad(new Date().getHours())}:${pad(new Date().getMinutes())}:${pad(new Date().getSeconds())}`;
  return {
    hosts,
    nodes,
    totalWorkloads,
    runningCount,
    rows,
    stamp,
    apiHealth: rand() < 0.92 ? 'ok' : 'degraded',
  };
}

// ---------- Content (unchanged) ----------------------------------------
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

// ---------- Motion tokens (single source of truth) --------------------
// Durations follow agent-design-intelligence: 320ms for UI, 420-520ms for
// bigger moments. Easings: ease-out for enter, ease-in for exit. Stagger 50-80ms.
// Reduced motion is honored globally via useReducedMotion() below.
const EASE_OUT: [number, number, number, number] = [0.16, 1, 0.3, 1];

const fadeUp: Variants = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.42, ease: EASE_OUT } },
};

const stagger = (delayChildren = 0.08, staggerChildren = 0.06): Variants => ({
  hidden: {},
  show: { transition: { delayChildren, staggerChildren } },
});

const cardHover: MotionHoverProps = { y: -3, transition: { type: 'spring', stiffness: 380, damping: 26 } };

// ---------- KPI counter (animates from 0 -> n over ~700ms) ------------
type MotionHoverProps = { y: number; transition: { type: 'spring'; stiffness: number; damping: number } };

function CountUp({ value, durationMs = 700 }: { value: number; durationMs?: number }) {
  const reduce = useReducedMotion();
  const [display, setDisplay] = useState(reduce ? value : 0);
  const raf = useRef<number | null>(null);
  useEffect(() => {
    if (reduce) {
      setDisplay(value);
      return;
    }
    const start = performance.now();
    const ease = (t: number) => 1 - Math.pow(1 - t, 3);
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / durationMs);
      setDisplay(Math.round(value * ease(t)));
      if (t < 1) raf.current = requestAnimationFrame(tick);
    };
    raf.current = requestAnimationFrame(tick);
    return () => {
      if (raf.current != null) cancelAnimationFrame(raf.current);
    };
  }, [value, durationMs, reduce]);
  return <>{display}</>;
}

// ---------- Live dot (real elapsed-since-mount, no fake shimmer) -------
function LiveDot() {
  const [, force] = useState(0);
  useEffect(() => {
    const id = window.setInterval(() => force((n) => n + 1), 1000);
    return () => window.clearInterval(id);
  }, []);
  return <span className="landing-live-dot" aria-hidden="true" />;
}

// ---------- Topbar (motion-aware Sign in / Sign up CTAs only) ----------
function LandingTopbar() {
  const reduce = useReducedMotion();
  return (
    <motion.header
      className="landing-topbar"
      initial={reduce ? false : { opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.32, ease: EASE_OUT }}
    >
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
        <motion.div whileHover={cardHover} className="landing-button-wrap">
          <Link className="landing-button landing-button-ghost" to="/login">Sign in</Link>
        </motion.div>
        <motion.div whileHover={cardHover} className="landing-button-wrap">
          <Link className="landing-button landing-button-primary" to="/signup">Sign up free</Link>
        </motion.div>
      </div>
    </motion.header>
  );
}

// ---------- Hero --------------------------------------------------------
function LandingHero({ preview }: { preview: ReturnType<typeof generatePreview> }) {
  const reduce = useReducedMotion();
  const heroItemVariants: Variants = {
    hidden: { opacity: 0, y: 16 },
    show: { opacity: 1, y: 0, transition: { duration: 0.5, ease: EASE_OUT } },
  };

  return (
    <motion.section
      className="landing-hero"
      initial={reduce ? false : "hidden"}
      animate="show"
      variants={stagger(0, 0.08)}
    >
      <motion.div className="landing-hero-text" variants={heroItemVariants}>
        <motion.span className="landing-eyebrow" variants={heroItemVariants}>
          Self-hosted · Open · Yours
        </motion.span>
        <motion.h1 variants={heroItemVariants}>
          Run your infrastructure from <span className="landing-gradient">one calm screen</span>.
        </motion.h1>
        <motion.p className="landing-lede" variants={heroItemVariants}>
          StackWatch is the dashboard for operators who would rather own their monitoring than rent it.
          Register a Proxmox or TrueNAS control plane and see your entire fleet — nodes, workloads, alerts,
          terminals — in a single place, on hardware you control.
        </motion.p>
        <motion.div className="landing-hero-actions" variants={heroItemVariants}>
          <motion.div whileHover={cardHover} whileTap={{ y: 0, scale: 0.98 }} className="landing-button-wrap">
            <Link className="landing-button landing-button-primary landing-button-large" to="/signup">
              Create a free account
            </Link>
          </motion.div>
          <motion.div whileHover={cardHover} className="landing-button-wrap">
            <Link className="landing-button landing-button-ghost landing-button-large" to="/login">
              I already have one
            </Link>
          </motion.div>
        </motion.div>
        <motion.div className="landing-hero-meta" variants={heroItemVariants}>
          <span><span className="landing-meta-dot" />No credit card</span>
          <span><span className="landing-meta-dot" />Free tier for 1 host</span>
          <span><span className="landing-meta-dot" />Open source</span>
        </motion.div>
      </motion.div>

      <motion.aside
        className="landing-hero-card"
        aria-label="Dashboard preview"
        variants={heroItemVariants}
        whileHover={reduce ? undefined : { y: -2, transition: { duration: 0.25, ease: EASE_OUT } }}
      >
        <header className="landing-hero-card-head">
          <span className="landing-eyebrow">Command center</span>
          <span className="landing-hero-status">
            <LiveDot />
            <span>Live</span>
          </span>
        </header>
        <motion.div className="landing-hero-kpis" variants={stagger(0.1, 0.05)} initial={reduce ? false : "hidden"} animate="show">
          <motion.div variants={heroItemVariants}>
            <small>Connected hosts</small>
            <strong className="landing-tabular"><CountUp value={preview.hosts} /></strong>
          </motion.div>
          <motion.div variants={heroItemVariants}>
            <small>Compute nodes</small>
            <strong className="landing-tabular"><CountUp value={preview.nodes} /></strong>
          </motion.div>
          <motion.div variants={heroItemVariants}>
            <small>Running workloads</small>
            <strong className="landing-tabular"><CountUp value={preview.runningCount} /></strong>
            <small className="landing-subline">of {preview.totalWorkloads} total</small>
          </motion.div>
          <motion.div variants={heroItemVariants}>
            <small>API health</small>
            <strong className={preview.apiHealth === 'ok' ? 'landing-good landing-tabular' : 'landing-warn landing-tabular'}>
              {preview.apiHealth}
            </strong>
          </motion.div>
        </motion.div>
        <motion.div className="landing-hero-rows" variants={stagger(0.25, 0.06)} initial={reduce ? false : "hidden"} animate="show">
          {preview.rows.map((row, idx) => (
            <motion.div key={`${row.name}-${idx}`} variants={heroItemVariants}>
              <span className={`landing-row-dot landing-row-${row.tone}`} />
              {row.name} <em>{row.detail}</em>
            </motion.div>
          ))}
        </motion.div>
        <div className="landing-hero-foot">
          Example snapshot · regenerated on every page load · <span className="landing-tabular">{preview.stamp}</span>
        </div>
      </motion.aside>
    </motion.section>
  );
}

// ---------- Features grid (stagger 6-up) -------------------------------
function FeatureGrid() {
  const reduce = useReducedMotion();
  return (
    <motion.section
      className="landing-section"
      id="features"
      initial={reduce ? false : "hidden"}
      whileInView="show"
      viewport={{ once: true, margin: '-80px' }}
      variants={stagger(0, 0.06)}
    >
      <motion.span className="landing-eyebrow" variants={fadeUp}>What you get</motion.span>
      <motion.h2 variants={fadeUp}>Everything a small infra team actually uses.</motion.h2>
      <motion.p className="landing-section-lede" variants={fadeUp}>
        No 200-feature checklist you&apos;ll never open. The dashboard gives you the things you reach for every day,
        built to feel fast on a five-server homelab and a fifty-server production cluster alike.
      </motion.p>
      <motion.div
        className="landing-feature-grid"
        variants={stagger(0.08, 0.05)}
      >
        {features.map((feature) => (
          <motion.article
            className="landing-feature"
            key={feature.title}
            variants={fadeUp}
            whileHover={reduce ? undefined : { y: -4, transition: { type: 'spring', stiffness: 360, damping: 24 } }}
          >
            <span className="landing-feature-icon">{feature.icon}</span>
            <strong>{feature.title}</strong>
            <p>{feature.body}</p>
          </motion.article>
        ))}
      </motion.div>
    </motion.section>
  );
}

// ---------- Steps (numbered, stagger 3-up) -----------------------------
function HowSteps() {
  const reduce = useReducedMotion();
  return (
    <motion.section
      className="landing-section landing-section-alt"
      id="how"
      initial={reduce ? false : "hidden"}
      whileInView="show"
      viewport={{ once: true, margin: '-80px' }}
      variants={stagger(0, 0.08)}
    >
      <motion.span className="landing-eyebrow" variants={fadeUp}>How it works</motion.span>
      <motion.h2 variants={fadeUp}>Three steps from signup to live infra.</motion.h2>
      <motion.ol className="landing-steps" variants={stagger(0.08, 0.07)}>
        {steps.map((step) => (
          <motion.li
            key={step.n}
            variants={fadeUp}
            whileHover={reduce ? undefined : { x: 4, transition: { duration: 0.2, ease: EASE_OUT } }}
          >
            <span className="landing-step-num">{step.n}</span>
            <div>
              <strong>{step.title}</strong>
              <p>{step.body}</p>
            </div>
          </motion.li>
        ))}
      </motion.ol>
    </motion.section>
  );
}

// ---------- Pricing (3-up, featured lifts) ------------------------------
function Pricing() {
  const reduce = useReducedMotion();
  return (
    <motion.section
      className="landing-section"
      id="pricing"
      initial={reduce ? false : "hidden"}
      whileInView="show"
      viewport={{ once: true, margin: '-80px' }}
      variants={stagger(0, 0.07)}
    >
      <motion.span className="landing-eyebrow" variants={fadeUp}>Pricing</motion.span>
      <motion.h2 variants={fadeUp}>Free for personal use. Honest pricing for teams.</motion.h2>
      <motion.p className="landing-section-lede" variants={fadeUp}>
        The full product is free to run on your own hardware. Cloud billing covers the cost of running it for you.
      </motion.p>
      <motion.div
        className="landing-pricing"
        variants={stagger(0.08, 0.06)}
      >
        {[
          {
            planClass: 'landing-plan',
            key: 'self',
            header: (
              <header>
                <strong>Self-hosted</strong>
                <span className="landing-price">Free</span>
                <small>Forever, on your own hardware</small>
              </header>
            ),
            items: ['Unlimited hosts and workloads', 'Full command center + terminal', 'All alert rules and integrations', 'Source-available, audit your own code'],
            cta: <Link className="landing-button landing-button-ghost" to="/signup">Get the binary</Link>,
          },
          {
            planClass: 'landing-plan landing-plan-featured',
            key: 'free',
            header: (
              <>
                <span className="landing-plan-badge">Most popular</span>
                <header><strong>Cloud free</strong><span className="landing-price">$0</span><small>Up to 1 host, forever</small></header>
              </>
            ),
            items: ['1 connected host, unlimited workloads', 'Hosted control plane at stackwatch.smarthomelab.fun', 'All dashboards, alerts, terminals', 'Community support'],
            cta: <Link className="landing-button landing-button-primary" to="/signup">Start free</Link>,
          },
          {
            planClass: 'landing-plan',
            key: 'team',
            header: (
              <header>
                <strong>Cloud team</strong>
                <span className="landing-price">$8<span>/host/mo</span></span>
                <small>For teams running more than one host</small>
              </header>
            ),
            items: ['Unlimited hosts and workloads', 'SAML SSO, role-based access', 'Audit log retention, priority support', 'Email and chat onboarding'],
            cta: <Link className="landing-button landing-button-ghost" to="/signup">Start 14-day trial</Link>,
          },
        ].map((plan) => (
          <motion.article
            className={plan.planClass}
            key={plan.key}
            variants={fadeUp}
            whileHover={reduce ? undefined : { y: plan.planClass.includes('featured') ? -6 : -4, transition: { type: 'spring', stiffness: 360, damping: 24 } }}
          >
            {plan.header}
            <ul>
              {plan.items.map((it) => <li key={it}>{it}</li>)}
            </ul>
            {plan.cta}
          </motion.article>
        ))}
      </motion.div>
    </motion.section>
  );
}

// ---------- Bottom CTA + Footer (unchanged structure) ------------------
function BottomCta() {
  const reduce = useReducedMotion();
  return (
    <motion.section
      className="landing-cta"
      initial={reduce ? false : { opacity: 0, y: 16 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.5, ease: EASE_OUT }}
    >
      <div>
        <h2>Ready to see your fleet in one place?</h2>
        <p>Create a free account in under a minute. No credit card, no trial countdown.</p>
      </div>
      <div className="landing-cta-actions">
        <motion.div whileHover={cardHover} whileTap={{ y: 0, scale: 0.98 }} className="landing-button-wrap">
          <Link className="landing-button landing-button-primary landing-button-large" to="/signup">Create free account</Link>
        </motion.div>
        <motion.div whileHover={cardHover} className="landing-button-wrap">
          <Link className="landing-button landing-button-ghost landing-button-large" to="/login">Sign in</Link>
        </motion.div>
      </div>
    </motion.section>
  );
}

// ---------- Main -------------------------------------------------------
export default function Landing() {
  const preview = useMemo(() => generatePreview(), []);

  return (
    <div className="landing-app">
      <LandingTopbar />
      <main>
        <LandingHero preview={preview} />
        <FeatureGrid />
        <HowSteps />
        <Pricing />
        <BottomCta />
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
