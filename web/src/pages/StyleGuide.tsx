/**
 * StyleGuide.tsx — visual showcase of every Tier 20 shared component.
 *
 * Purpose: every variant + state of every shared component, rendered
 * in one place. Unlisted from sidebar nav (accessible only via direct
 * /style-guide URL).
 *
 * Sections (11 total, per plan §5):
 *   1. Color tokens — visual swatches of every var(--color-*)
 *   2. Typography — every var(--text-*) size rendered
 *   3. Spacing — visual bars of every var(--space-*)
 *   4. Buttons — — every variant × size × state
 *   5. Modals — one open, one closed example
 *   6. DataTables — sample table
 *   7. Inputs — every form input type + error state
 *   8. StatusPills — every status × size
 *   9. KpiCards — every accent × with/without sparkline
 *   10. EmptyStates — example with title + subhead
 *   11. Loading — SkeletonRow + SkeletonCard examples
 */
import { useState } from 'react';
import Button from '../components/shared/Button';
import Modal from '../components/shared/Modal';
import DataTable, { type Column } from '../components/shared/DataTable';
import Input from '../components/shared/Input';
import Select from '../components/shared/Select';
import Textarea from '../components/shared/Textarea';
import StatusPill from '../components/shared/StatusPill';
import KpiCard from '../components/shared/KpiCard';
import EmptyState from '../components/shared/EmptyState';
import SkeletonRow from '../components/shared/SkeletonRow';
import SkeletonCard from '../components/shared/SkeletonCard';
import BrandLogo from '../components/shared/BrandLogo';
import { HeartIcon, ServerIcon, ActivityIcon, DatabaseIcon } from '../components/icons';

interface SectionProps {
  title: string;
  number: number;
  children: React.ReactNode;
}

function Section({ title, number, children }: SectionProps) {
  return (
    <section className="sg-section">
      <header className="sg-section-header">
        <span className="sg-section-number">{String(number).padStart(2, '0')}</span>
        <h2 className="sg-section-title">{title}</h2>
      </header>
      <div className="sg-section-body">{children}</div>
    </section>
  );
}

interface ColorSwatch {
  name: string;
  token: string;
  category: 'surface' | 'text' | 'brand' | 'semantic' | 'border';
}

const COLOR_SWATCHES: ColorSwatch[] = [
  { name: 'bg', token: '--color-bg', category: 'surface' },
  { name: 'bg-elevated', token: '--color-bg-elevated', category: 'surface' },
  { name: 'surface', token: '--color-surface', category: 'surface' },
  { name: 'surface-elevated', token: '--color-surface-elevated', category: 'surface' },
  { name: 'surface-hover', token: '--color-surface-hover', category: 'surface' },
  { name: 'text', token: '--color-text', category: 'text' },
  { name: 'text-muted', token: '--color-text-muted', category: 'text' },
  { name: 'text-subtle', token: '--color-text-subtle', category: 'text' },
  { name: 'text-disabled', token: '--color-text-disabled', category: 'text' },
  { name: 'primary', token: '--color-primary', category: 'brand' },
  { name: 'primary-hover', token: '--color-primary-hover', category: 'brand' },
  { name: 'primary-active', token: '--color-primary-active', category: 'brand' },
  { name: 'success', token: '--color-success', category: 'semantic' },
  { name: 'warning', token: '--color-warning', category: 'semantic' },
  { name: 'error', token: '--color-error', category: 'semantic' },
  { name: 'info', token: '--color-info', category: 'semantic' },
  { name: 'border', token: '--color-border', category: 'border' },
  { name: 'border-strong', token: '--color-border-strong', category: 'border' },
];

const TEXT_SAMPLES = [
  { token: 'text-xs', size: 11, label: 'Section header (uppercase, letter-spaced)' },
  { token: 'text-sm', size: 12, label: 'Small labels, captions' },
  { token: 'text-base', size: 14, label: 'Body text (default)' },
  { token: 'text-md', size: 16, label: 'Large body, card titles' },
  { token: 'text-lg', size: 18, label: 'Section titles' },
  { token: 'text-xl', size: 20, label: 'Page titles' },
  { token: 'text-2xl', size: 24, label: 'Big page titles' },
  { token: 'text-3xl', size: 30, label: 'KPI values' },
  { token: 'text-4xl', size: 36, label: 'Hero KPI' },
];

const SPACING_TOKENS = [4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128];

const STATUS_VALUES = ['up', 'ok', 'stale', 'warn', 'down', 'crit', 'unknown'] as const;
const STATUS_SIZES = ['sm', 'md', 'lg'] as const;
const KPI_ACCENTS = ['cyan', 'indigo', 'green', 'amber', 'red', 'violet'] as const;

interface DemoRow {
  id: string;
  name: string;
  count: number;
}
const demoColumns: Column<DemoRow>[] = [
  { key: 'name', header: 'Name', sortable: true },
  { key: 'count', header: 'Count', sortable: true, align: 'right' as const },
];
const demoRows: DemoRow[] = [
  { id: '1', name: 'apple', count: 5 },
  { id: '2', name: 'banana', count: 3 },
  { id: '3', name: 'cherry', count: 8 },
];

export default function StyleGuide() {
  const [modalOpen, setModalOpen] = useState(false);

  return (
    <div className="style-guide">
      <header className="sg-page-head">
        <div>
          <span className="px-eyebrow">Tier 20 — UI Consistency Polish</span>
          <h1 className="sg-page-title">Style Guide</h1>
          <p className="sg-page-sub">
            Every shared component, every variant, every state. The single
            source of truth for what every page should look like.
          </p>
        </div>
        <BrandLogo variant="full" size={48} />
      </header>

      <Section title="Color tokens" number={1}>
        <div className="sg-color-grid">
          {COLOR_SWATCHES.map((swatch) => (
            <div key={swatch.token} className="sg-swatch">
              <div
                className={`sg-swatch-chip sg-swatch-${swatch.category}`}
                style={{ background: `var(${swatch.token})` }}
              />
              <div className="sg-swatch-meta">
                <code className="sg-swatch-token">{swatch.token}</code>
                <span className="sg-swatch-name">{swatch.name}</span>
              </div>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Typography" number={2}>
        <div className="sg-text-samples">
          {TEXT_SAMPLES.map((s) => (
            <div key={s.token} className="sg-text-sample">
              <span className="sg-text-label">
                {s.token} — {s.size}px — {s.label}
              </span>
              <div style={{ fontSize: `${s.size}px` }}>The quick brown fox</div>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Spacing" number={3}>
        <div className="sg-spacing-samples">
          {SPACING_TOKENS.map((n) => (
            <div key={n} className="sg-spacing-sample">
              <span className="sg-spacing-label">--space-{n === 4 ? '1' : n === 8 ? '2' : n === 12 ? '3' : n === 16 ? '4' : n === 20 ? '5' : n === 24 ? '6' : n === 32 ? '8' : n === 40 ? '10' : n === 48 ? '12' : n === 64 ? '16' : n === 80 ? '20' : n === 96 ? '24' : '32'} — {n}px</span>
              <div
                className="sg-spacing-bar"
                style={{ width: `${n}px`, height: '12px', background: 'var(--color-primary)' }}
              />
            </div>
          ))}
        </div>
      </Section>

      <Section title="Buttons" number={4}>
        <div className="sg-button-row">
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="danger">Danger</Button>
        </div>
        <div className="sg-button-row">
          <Button variant="primary" size="sm">Small</Button>
          <Button variant="primary" size="md">Medium</Button>
          <Button variant="primary" size="lg">Large</Button>
        </div>
        <div className="sg-button-row">
          <Button variant="primary" disabled>Disabled</Button>
          <Button variant="primary" loading>Loading</Button>
          <Button variant="primary" icon={<HeartIcon size={16} />}>With icon</Button>
          <Button variant="primary" fullWidth>Full width</Button>
        </div>
      </Section>

      <Section title="Modals" number={5}>
        <Button onClick={() => setModalOpen(true)}>Open modal</Button>
        <Modal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          title="Confirm deletion"
          description="This cannot be undone"
          footer={
            <>
              <Button variant="ghost" onClick={() => setModalOpen(false)}>
                Cancel
              </Button>
              <Button variant="danger" onClick={() => setModalOpen(false)}>
                Delete
              </Button>
            </>
          }
        >
          <p>Are you sure you want to delete this server? All metrics will be lost.</p>
        </Modal>
      </Section>

      <Section title="DataTables" number={6}>
        <DataTable columns={demoColumns} rows={demoRows} rowKey={(r) => r.id} />
      </Section>

      <Section title="Form inputs" number={7}>
        <div className="sg-input-grid">
          <Input label="Email" placeholder="you@example.com" description="We'll never spam you" />
          <Input label="Password" type="password" required />
          <Input label="With error" error="This field is required" defaultValue="bad value" />
          <Input label="Disabled" disabled defaultValue="Can't edit" />
          <Select
            label="Plan"
            options={[
              { value: 'free', label: 'Free' },
              { value: 'pro', label: 'Pro' },
              { value: 'enterprise', label: 'Enterprise' },
            ]}
            placeholder="Choose a plan"
          />
          <Textarea label="Notes" placeholder="Optional notes…" rows={4} />
        </div>
      </Section>

      <Section title="Status pills" number={8}>
        <div className="sg-status-grid">
          {STATUS_VALUES.map((status) => (
            <div key={status} className="sg-status-row">
              <span className="sg-status-label">{status}</span>
              <StatusPill status={status} size="sm" />
              <StatusPill status={status} size="md" />
              <StatusPill status={status} size="lg" />
            </div>
          ))}
        </div>
      </Section>

      <Section title="KPI cards" number={9}>
        <div className="sg-kpi-grid">
          {KPI_ACCENTS.map((accent) => (
            <KpiCard
              key={accent}
              label={accent.toUpperCase()}
              value="42.5%"
              delta="+2.1%"
              accent={accent as 'cyan' | 'indigo' | 'green' | 'amber' | 'red' | 'violet'}
              icon={
                accent === 'cyan' ? <ActivityIcon size={16} /> :
                accent === 'indigo' ? <DatabaseIcon size={16} /> :
                accent === 'green' ? <HeartIcon size={16} /> :
                accent === 'amber' ? <ServerIcon size={16} /> :
                <ServerIcon size={16} />
              }
              sparkline={[3, 5, 4, 7, 6, 8, 9]}
            />
          ))}
        </div>
      </Section>

      <Section title="Empty states" number={10}>
        <EmptyState
          illustration={<ServerIcon size={48} />}
          headline="No servers yet"
          subhead="Add your first server to start monitoring."
          cta={{ label: 'Add server', onClick: () => {} }}
        />
      </Section>

      <Section title="Loading states" number={11}>
        <div className="sg-loading-grid">
          <div>
            <span className="sg-loading-label">SkeletonRow (4 cols)</span>
            <SkeletonRow columns={4} />
          </div>
          <div>
            <span className="sg-loading-label">SkeletonCard (kpi)</span>
            <SkeletonCard variant="kpi" />
          </div>
          <div>
            <span className="sg-loading-label">SkeletonCard (panel)</span>
            <SkeletonCard variant="panel" />
          </div>
          <div>
            <span className="sg-loading-label">SkeletonCard (list)</span>
            <SkeletonCard variant="list" />
          </div>
        </div>
      </Section>
    </div>
  );
}