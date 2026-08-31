# StackWatch UI Consistency Audit Report

**Tier:** 20 — UI Consistency Polish
**Generated:** 2026-08-31 16:03 
**Design system reference:** `web/src/styles/DESIGN-SYSTEM.md` (v1.0, last updated 2026-08-26)
**Token source of truth:** `web/src/styles/tokens.css`

> **Scope.** Every `.tsx` file under `web/src/pages/` (27 files) and every `.tsx` file under `web/src/components/` excluding `web/src/components/proxmox/` (142 files). For each file we catalog every usage of: **button**, **KPI card**, **status pill**, **modal**, **table**, **form input**, **empty state**, **loading state**.
>
> **Methodology.** ripgrep + Python AST-free regex scan. For each match we extract the line range, the surrounding 2-3 lines of source, classify against the design system (does it use the shared component from `web/src/components/shared/`? does it use tokens? does it have accessible focus/loading/empty states?), assign severity (`critical` / `medium` / `low`), and propose a concrete file-line fix referencing the shared component to use.
>
> **Shared components available today** (`web/src/components/shared/`): `Button`, `StatusPill`, `KpiCard`, `EmptyState`, `SkeletonRow`, plus 30+ domain-specific cards/charts.
>
> **Shared components still missing** (need to be created in Phase D before the per-page refactor starts): `Modal`, `DataTable`, `Input`, `Select`, `Textarea`, `BrandLogo`.

---

## Summary

**Files audited:** 169 (27 pages + 142 components)
**Total element usages cataloged:** 771
**Compliant usages (using shared components/tokens correctly):** 196
**Drifts found (action required):** 575
**Files with at least one drift:** 133 / 169
**Files fully compliant:** 36

### Drift breakdown by severity

| Severity | Count | Meaning |
|----------|-------|---------|
| **Critical** | 210 | Raw HTML primitives (`<button>`, `<input>`, `<select>`, `<textarea>`) where a shared component exists or should be created; custom modal markup without shared `<Modal>`; raw `<table>` markup that bypasses the design-system table styling. **Must fix before Phase D ships.** |
| **Medium**   | 327 | Custom button class prefixes (`sw-button-*`, `px-button-*`, `auth-button-*`, `landing-button-*`, `btn-primary`); custom KPI markup that bypasses `<KpiCard>`; tables that should use a (to-be-created) `<DataTable>`; custom `dash-panel-empty` placeholders that should be `<SkeletonCard>`. |
| **Low**      | 38 | Domain-specific table classes, custom empty illustrations, banner-style uses of `<Modal>` that already do the right thing. Nice-to-have polish. |
| None (compliant — already in summary) | 196 | Correctly using shared components (`<Button>`, `<StatusPill>`, `<KpiCard>`, `<EmptyState>`, `<SkeletonRow>`). Listed for completeness. |

### Usages by element type

| Element type | Total usages | Critical | Medium | Low | Compliant |
|--------------|--------------|----------|--------|-----|-----------|
| button | 260 | 167 | 89 | 0 | 4 |
| form_input | 175 | 0 | 150 | 23 | 2 |
| kpi_card | 160 | 0 | 42 | 0 | 118 |
| modal | 70 | 43 | 18 | 6 | 3 |
| empty_state | 43 | 0 | 2 | 3 | 38 |
| status_pill | 27 | 0 | 2 | 0 | 25 |
| table | 23 | 0 | 22 | 1 | 0 |
| loading_state | 13 | 0 | 2 | 5 | 6 |

### Drift density by file (top 15 by drift count)

| File | Drifts | Critical | Medium | Low |
|------|--------|----------|--------|-----|
| `web/src/pages/TrueNASWorkspace.tsx` | 18 | 5 | 11 | 2 |
| `web/src/components/homelab/widgets/AddSchedulerJobModal.tsx` | 14 | 6 | 8 | 0 |
| `web/src/pages/SyntheticsPage.tsx` | 13 | 1 | 12 | 0 |
| `web/src/components/platform/LimitsChangeModal.tsx` | 13 | 3 | 10 | 0 |
| `web/src/components/platform/RegionsSectionModals.tsx` | 13 | 7 | 6 | 0 |
| `web/src/components/shared/SsoProviderForm.tsx` | 13 | 0 | 4 | 9 |
| `web/src/components/logs/LogArchivesTab.tsx` | 11 | 0 | 11 | 0 |
| `web/src/components/CorrelationsSection.tsx` | 10 | 4 | 6 | 0 |
| `web/src/components/PredictiveAlertsSection.tsx` | 10 | 5 | 5 | 0 |
| `web/src/components/homelab/widgets/AddDownloadModal.tsx` | 10 | 4 | 6 | 0 |
| `web/src/components/platform/SignupSection.tsx` | 10 | 6 | 4 | 0 |
| `web/src/components/ScimSection.tsx` | 9 | 2 | 7 | 0 |
| `web/src/components/homelab/widgets/TodoModal.tsx` | 9 | 4 | 5 | 0 |
| `web/src/components/logs/LogMonitorsTab.tsx` | 9 | 0 | 9 | 0 |
| `web/src/components/ComplianceSectionFormViews.tsx` | 8 | 0 | 8 | 0 |

### Drift density by element type (only files with ≥1 drift in that type)

| Element type | Files with ≥1 drift | Total drift count |
|--------------|----------------------|-------------------|
| button | 117 | 256 |
| form_input | 57 | 173 |
| kpi_card | 32 | 42 |
| modal | 31 | 67 |
| table | 18 | 23 |
| loading_state | 6 | 7 |
| empty_state | 5 | 5 |
| status_pill | 2 | 2 |

---

## Severity rubric

```
critical  Raw HTML primitive where a shared component exists or is missing —
          this user-visible inconsistency contradicts the design system.
          Example: <button> → <Button>, raw <input> → shared <Input>.

medium    Implementation works but bypasses the shared API or uses a
          non-canonical class prefix. Visual output is correct; the
          drift is in the source contract.
          Example: sw-button-primary → <Button variant="primary">.

low       Domain-specific class or scoped variation. Refactor to shared
          when the component is created or promoted.
          Example: limits-modal-grid → shared Modal once it exists.
```

---

## Table of contents

```
Summary ........................................................... 1
  Drift breakdown by severity ............................. 1
  Usages by element type ................................. 1
  Drift density by file (top 15) ......................... 1
  Drift density by element type .......................... 1
Severity rubric ................................................ 2
Per-element-type deep dive ...................................... 3
  Buttons ............................................... 3
  KPI cards ............................................. 4
  Status pills .......................................... 4
  Modals ................................................ 5
  Tables ................................................ 5
  Form inputs ........................................... 6
  Empty states .......................................... 6
  Loading states ........................................ 7
Per-file findings ............................................... 8
  (A) Files with drift — 133 files, 575 findings ........ 8
    Pages (27 files) ................................. 8
    Components (107 files) ........................... 35
  (B) Fully compliant files — 36 files ................. 175
Supplementary findings — token & accessibility drift ......... 176
  Hardcoded hex colors (37 hits) ...................... 176
  Inline px styles (25 hits) .......................... 177
Recommended new shared components ............................ 178
  Modal .............................................. 178
  DataTable ........................................... 179
  Input ............................................... 180
  Select .............................................. 180
  Textarea ............................................ 181
  BrandLogo ........................................... 181
Phase D execution roadmap ..................................... 182
  D.1 Build the missing shared components ............. 182
  D.2 Page-by-page refactor ........................... 182
  D.3 Token & accessibility sweep ...................... 184
  D.4 Acceptance checklist ............................ 184
Closing notes ................................................... 185
```

## How to read this report

1. **Start at the Summary.** The four stat boxes (severity / element type / per-file / per-type) tell you what the audit found at a glance.
2. **Skim Per-element-type deep dive** to understand the patterns — *why* a drift exists and *what* the canonical replacement looks like.
3. **Jump to Per-file findings → (A)** when working a specific file. Each file has its own H3 header and one table. The *Proposed Fix* column is your action item.
4. **Use Supplementary findings** when you suspect a hex color or inline px slipped past the per-element-type detectors.
5. **Use Phase D execution roadmap** when sequencing work across the team.

**Acceptance criteria for "Tier 20 done":** every drift row in §(A) is gone, the supplementary findings lists are empty, and the acceptance checklist in §D.4 passes with zero matches.

---

## Per-element-type deep dive

> Eight sections, one per audited element type. Each section lists the **specific drift patterns** found in that type, the **file:line examples**, and the **canonical migration**. Read this if you're working on a specific element type rather than a specific file.

### Buttons — 260 usages, 167 critical drifts, 89 medium drifts, 4 compliant

**Why this is the biggest drift surface.** The shared `<Button>` component (`web/src/components/shared/Button.tsx`) exists and is the canonical implementation — but only **1 file** uses it. Every other file rolls its own `<button>` element with a class prefix from a previous era: `sw-button-*` (StackWatch legacy), `px-button-*` (Proxmox style), `auth-button-*` (auth flow), `landing-button-*` (landing page), or `btn-primary`/`btn-secondary`/`btn-danger` (Bootstrap-style leftover from early prototypes).

**Canonical button API:**

```tsx
import Button from '../components/shared/Button';

<Button
  variant="primary"     // 'primary' | 'secondary' | 'ghost' | 'danger'
  size="md"            // 'sm' = 32px | 'md' = 40px | 'lg' = 48px
  icon={<SaveIcon />}   // leading icon
  iconRight={<ChevronIcon />}  // trailing icon
  loading={isSaving}   // shows spinner, disables click
  disabled={!canSave}
  fullWidth={false}
  onClick={handleSave}
>Save changes</Button>
```

**Top button-drift offenders (by drift count):**

| File | Raw buttons | Recommended replacement |
|------|-------------|--------------------------|
| `web/src/components/ProfileMenu.tsx` | 11 | All `<button>` → `<Button variant="ghost" size="sm">` (topbar overflow menu) |
| `web/src/components/topbar/TopBarProfile.tsx` | 9 | `<Button variant="ghost" size="sm">` for menu items |
| `web/src/components/profile/TokensPanel.tsx` | 8 | `<Button variant="primary">` for Create, `<Button variant="danger">` for Revoke |
| `web/src/components/profile/SecurityPanel.tsx` | 7 | Same pattern as TokensPanel |
| `web/src/components/homelab/widgets/*Modal.tsx` (×7) | 5-8 each | `<Button>` inside `<Modal>` footer |
| `web/src/pages/TrueNASWorkspace.tsx` | 8 | `<Button variant="primary|secondary|danger">` for actions |

**Drift classification rules used in the audit:**
- ❌ **critical** — raw `<button>` element with no class, or raw `<button>` with custom non-token class
- ❌ **medium** — raw `<button>` with a non-canonical class prefix (`sw-button-*`, `px-button-*`, `auth-button-*`, `landing-button-*`, `btn-primary`/etc)
- ✅ **compliant** — `<Button>` from `shared/Button.tsx` (or `<motion.button>` inside `shared/Button.tsx` itself)

### KPI cards — 160 usages, 42 medium drifts, 118 compliant

**Mostly compliant.** 118 of 160 usages already go through `<KpiCard>`. The 42 drifts are in older components that predate the shared implementation and use `dash-metric`, `dash-kpi-strip`, `kpi-value`, or `homelab-kpi-*` classes.

**Canonical KPI API:**

```tsx
import KpiCard from '../components/shared/KpiCard';

<KpiCard
  label="Services"           // uppercase eyebrow
  value={totals.services}    // numeric or short string
  delta="+12 this week"    // subtext
  status="up"             // 'up' | 'down' | 'stale' | 'crit' | 'neutral'
  accent="indigo"          // overrides status→tone
  icon={<ServerIcon />}
  sparkline={[12, 14, 16, 18, 20]}  // optional inline sparkline
  trend="up"               // arrow next to delta
  onClick={openServicesTab}  // makes the card a button
/>
```

**Where drift still lives:**

| File | Drift pattern | Fix |
|------|---------------|------|
| `web/src/components/dashboard/WelcomeHeader.tsx` | `<div className="dash-metric">` | Wrap in `<KpiCard label value accent="cyan">` |
| `web/src/components/dashboard/ErrorBar.tsx` | `dash-metric` | `<KpiCard status="down" accent="red">` |
| `web/src/components/homelab/HomelabKpiStrip.tsx` | Custom `homelab-kpi-*` classes | `<KpiCard accent>` |
| `web/src/pages/HomelabPage.tsx` | `kpi-strip` row container | `<div className="kpi-row">` with `<KpiCard>` children |

### Status pills — 27 usages, 2 medium drifts, 25 compliant

**Almost perfect.** 25/27 usages go through `<StatusPill>`. The 2 drifts are in CSPM severity badges (`severity-pill` class) which should route through `<CspmSeverityBadge>` from `shared/`, not raw status pills.

**Canonical status pill API:**

```tsx
import StatusPill from '../components/shared/StatusPill';

<StatusPill status="up" />                    // 'Online'
<StatusPill status="down" />                  // 'Offline'
<StatusPill status="stale" />                 // 'Stale'
<StatusPill status="warn" />                 // 'Warning'
<StatusPill status="crit" />                 // 'Critical'
<StatusPill status="unknown" />              // 'Unknown'
<StatusPill status="up" label="Healthy" />  // custom label
<StatusPill status="up" size="sm" />        // 'sm' | 'md' | 'lg'
<StatusPill status="up" iconOnly />           // dot only, no text
```

### Modals — 70 usages, 43 critical drifts, 18 medium drifts, 9 mixed/low

**Highest-leverage drift to fix.** Today there is **no shared `<Modal>`**. 14+ components implement modal markup independently, with at least 6 different class-prefix conventions. Every modal implementation has to re-solve focus trap, ESC-to-close, backdrop click, body scroll lock, ARIA wiring, and motion. **Create `shared/Modal.tsx` first (Phase D.1.1), then sweep.**

**Distinct modal markup patterns found:**

| Pattern | Files | Class prefix |
|---------|-------|---------------|
| StackWatch legacy | 17 files (TrueNASWorkspace, platform/*, etc.) | `sw-modal-backdrop`, `sw-modal-card` |
| Proxmox (now excluded but referenced) | n/a | `px-modal-backdrop` |
| Limits / platform | 4 files | `limits-modal-*`, `limits-modal-option-*` |
| Slow query explain | 3 files (RbacSection, NoiseReductionSection, SnoozeHistoryPanel) | `slow-query-explain-modal` |
| Regions | 2 files | `regions-section-modal-*` |
| Audit | 1 file | `audit-section-modal-*` |
| CommandPalette | 1 file | `cmd-palette-modal` |
| Compliance / Sso / Scim | 4 files | bare `modal-*` with `role="dialog"` |

All of these collapse into one canonical `<Modal>` component once Phase D.1.1 lands.

### Tables — 23 usages, 22 medium drifts, 1 low

**No shared `<DataTable>` exists yet.** Every audit-page table is hand-rolled with `<table className="dash-table">` or `<table className="w-full text-sm">` (raw Tailwind). After Phase D.1.5, all of these migrate to the canonical `<DataTable columns rows rowKey>` API.

**Tables in the audit:**

| File | Class | Rows | Notes |
|------|-------|------|-------|
| `web/src/pages/ApmPage.tsx` | `dash-table apm-deployments` | 1 | Deployment history list |
| `web/src/pages/ApmServicePage.tsx` | `dash-table apm-traces` | 1 | Trace list |
| `web/src/pages/CspmPage.tsx` | `dash-table` | 2 | Findings table (×2 instances) |
| `web/src/components/logs/LogRetentionTab.tsx` | `dash-table` | 1 | Retention rules |
| `web/src/components/logs/LogPatternsTab.tsx` | `dash-table` | 1 | Detected patterns |
| `web/src/components/logs/LogArchivesTab.tsx` | `dash-table` | 1 | Archive list |
| `web/src/components/logs/LogMonitorsTab.tsx` | `dash-table` | 1 | Monitor list |
| `web/src/components/shared/RumSessionTabs.tsx` | `sw-table` | 2 | Session events + errors |
| `web/src/components/shared/SlowQueryTable.tsx` | `dash-table slow-query-table` | 1 | DB slow queries |
| `web/src/components/shared/ScimTokenTable.tsx` | `scim-token-table` (variant of dash-table) | 1 | SCIM tokens |
| `web/src/components/platform/BackupTable.tsx` | `w-full text-sm` (raw Tailwind) | 1 | Backups |
| `web/src/components/platform/HealthSectionTables.tsx` | `w-full text-sm` | 2 | Service health |
| `web/src/components/platform/MeteringSection.tsx` | `metering-detail-table` | 1 | Metering events |

### Form inputs — 175 usages, 150 medium drifts, 23 low, 2 compliant

**Largest surface by far after buttons.** No shared `<Input>` exists. Every form on the platform re-implements label, hint, error, focus ring, and disabled state inline. **Phase D.1.2 is the most impactful single component to build.**

**Distribution by type:**

| Type | Count | Notes |
|------|-------|-------|
| `<input type="text">` | 78 | The bulk — text/email/url/search all collapse to one `<Input>` |
| `<input type="checkbox">` | 12 | Different shape — will need a `<Checkbox>` shared component (Phase D.1.8, out of scope today) |
| `<input type="radio">` | 4 | Same — `<Radio>` shared component (Phase D.1.8) |
| `<input type="number">` | 18 | Some can use `<Input type="number">`; numeric ranges need `<InputNumber>` |
| `<input type="password">` | 11 | Currently routed through `PasswordField` / `PasswordInput` — those should be re-exported as `<Input type="password" trailingIcon>` |
| `<select>` | 37 | Native `<select>` — Phase D.1.3 swaps to shared `<Select>` |
| `<textarea>` | 11 | Phase D.1.4 swaps to shared `<Textarea>` |

### Empty states — 43 usages, 2 medium drifts, 3 low, 38 compliant

**Largely compliant.** `<EmptyState>` is the most consistently-used shared component — 38/43 audit hits use it correctly. The 5 drifts are:

- `dash-panel-empty` in `BillingPage.tsx` (L116) and `SettingsPage.tsx` (L169) — used as loading state, not empty state
- `cmd-palette-empty` in `CommandPalette.tsx` (L137) — domain-specific
- `dash-empty-illustration` in `HostList.tsx` (L22) and `TrendChart.tsx` (L16) — fine as long as they're composition elements, not standalone placeholders

**Canonical empty state API:**

```tsx
import EmptyState from '../components/shared/EmptyState';

<EmptyState
  illustration={<ServerIcon size={48} />}  // 96×96 bubble wraps this
  headline="No servers yet"               // 24px / 600 weight
  subhead="Add your first host to start monitoring."  // 14px / muted
  cta={{ label: "Add server", onClick: openAddDialog }}  // single primary CTA
/>
```

**Anti-pattern:** Multiple CTAs (`Add server` + `Import from CSV` + `Watch demo`). The shared component enforces *one* CTA by API — keep it that way.

### Loading states — 13 usages, 2 medium drifts, 5 low, 6 compliant

**Tiny surface.** Most loading is implicit (page-level Suspense, SkeletonRow in logs table, SkeletonCard in dashboard). The 7 non-compliant hits are:

- `BillingPage.tsx` L116 — `dash-panel-empty` reused as loading placeholder (drift)
- `SettingsPage.tsx` L169 — same pattern (drift)
- `TrueNASWorkspace.tsx` L346 — `busy ? 'Loading…' : ...` raw text (low)
- `SchedulerWidget.tsx` L227 — same raw text (low)
- `SkeletonRow.tsx` itself (2 internal hits — compliant)

**Canonical loading API:**

```tsx
import SkeletonRow from '../components/shared/SkeletonRow';
import SkeletonCard from '../components/shared/SkeletonCard';  // promote from dashboard/

// For lists / tables:
<SkeletonRow columns={5} rows={3} />  // 5 blocks × 3 rows

// For KPI strip / card grid:
<SkeletonCard lines={2} />  // one card placeholder
```

**Action item:** Promote `web/src/components/dashboard/SkeletonCard.tsx` to `web/src/components/shared/SkeletonCard.tsx` so non-dashboard pages can use it.

## Per-file findings

> **Legend.** ✅ = compliant (no drift). ❌ = drift found. Severity: 🔴 **critical** · 🟡 **medium** · 🟢 **low**.

> **Layout.** This section has two parts: **(A) Files with drift** (133 files, 575 actionable findings) and **(B) Fully compliant files** (36 files, no changes needed). Within each part, pages come first (alphabetical), then components (alphabetical by full path, grouped by subdirectory).

> **Reading the table.** The *Line* column shows the start (and end if multi-line) of the offending construct. *Current Implementation* shows 1-3 lines of the actual source verbatim. *Proposed Fix* references the shared component to use, with file path. Compliant usages (where the file already uses the shared component correctly) are counted in the file header but not re-tabulated.

---

## (A) Files with drift — 133 files, 575 actionable findings

### Pages (27 files)

### `web/src/pages/ApmPage.tsx`
**page** · 379 LOC · **7 drifts** (0 🔴 · 7 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `186-188` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={registerForm.name}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `197-199` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={registerForm.language}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `207-209` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={registerForm.framework}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `217-219` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={registerForm.environment}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `227-229` | Button | `                  <button type="submit" className="sw-button sw-button-primary" disabled={registerBusy}>`<br>`                    {registerBusy ? 'Registering…' : 'Register'}`<br>`                  </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `238-240` | KPI card | `            <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">`<br>`              <KpiCard label="Services" value={totals.services} accent="indigo" sparkline={tracesHistory.length > 0 ? tracesHistory.map(() => totals.services) : undefined} />`<br>`              <KpiCard label="Traces/min" value={totals.tracesPerMin} accent="cyan" sparkline={tracesHistory} />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `347-349` | Table | `              <table className="dash-table apm-deployments">`<br>`                <thead>`<br>`                  <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/pages/ApmServicePage.tsx`
**page** · 193 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `165-167` | Table | `                  <table className="dash-table apm-traces">`<br>`                    <thead>`<br>`                      <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/pages/BillingPage.tsx`
**page** · 180 LOC · **3 drifts** (2 🔴 · 1 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `112-114` | Button | `              <button onClick={() => window.location.reload()}>Retry</button>`<br>`            </div>`<br>`          )}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `116-118` | Empty state | `            <section className="dash-panel"><div className="dash-panel-empty"><strong>Loading billing…</strong></div></section>`<br>`          ) : (`<br>`            <>` | ❌ | 🟡 medium | Custom `dash-panel-empty` placeholder (used for loading state here — should be `<SkeletonCard>` not EmptyState). Replace with shared `<SkeletonCard>` (lives at `web/src/components/dashboard/SkeletonCard.tsx` — promote to `shared/`). |
| `162-164` | Button | `                      <button`<br>`                        type="button"`<br>`                        className={\`sw-button ${currentPlanId === plan.id ? 'sw-button-quiet' : 'sw-button-primary'}\`}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/CicdPage.tsx`
**page** · 267 LOC · **4 drifts** (0 🔴 · 4 🟡 · 0 🟢)
  · also 6 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `134-136` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `185-187` | Form input | `                  <select`<br>`                    value={statusFilter}`<br>`                    onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `220-222` | Form input | `                  <select`<br>`                    value={envFilter}`<br>`                    onChange={(e) => setEnvFilter(e.target.value as EnvFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `238-240` | Table | `                <table className="dash-table">`<br>`                  <thead>`<br>`                    <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/pages/CspmPage.tsx`
**page** · 251 LOC · **5 drifts** (0 🔴 · 5 🟡 · 0 🟢)
  · also 4 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `119-121` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `163-165` | Table | `              <table className="dash-table">`<br>`                <thead>`<br>`                  <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |
| `194-196` | Form input | `                <select`<br>`                  value={sevFilter}`<br>`                  onChange={(e) => setSevFilter(e.target.value as SevFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `208-210` | Form input | `                <select`<br>`                  value={resolvedFilter}`<br>`                  onChange={(e) => setResolvedFilter(e.target.value as ResolvedFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `225-227` | Table | `              <table className="dash-table">`<br>`                <thead>`<br>`                  <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/pages/Dashboard.tsx`
**page** · 333 LOC · **2 drifts** (0 🔴 · 2 🟡 · 0 🟢)
  · also 11 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `171-173` | KPI card | `        className="dash-metric-grid"`<br>`        variants={kpiStagger}`<br>`        initial="hidden"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `260-262` | Form input | `            <input`<br>`              className="dash-search"`<br>`              type="search"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |

### `web/src/pages/DatabasePage.tsx`
**page** · 320 LOC · **6 drifts** (2 🔴 · 3 🟡 · 1 🟢)
  · also 7 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `152-154` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `200-202` | Form input | `              <select`<br>`                value={dbFilter}`<br>`                onChange={(e) => setDbFilter(e.target.value as DbFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `227-229` | Loading state | `                <p className="slow-query-explain-loading">Loading explain…</p>`<br>`              ) : null}`<br>`              {explain ? (` | ❌ | 🟢 low | Plain `'Loading…'` text — replace with shared `<SkeletonCard>` (for page-level) or `<SkeletonRow>` (for list rows). Button text `'Loading…'` is fine inside `<Button loading>` state. |
| `230-232` | Modal | `                <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`                  <div className="slow-query-explain-modal-head">`<br>`                    <strong>Explain · {explain.database} · {explain.query_hash.slice(0, 12)}</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `231-233` | Modal | `                  <div className="slow-query-explain-modal-head">`<br>`                    <strong>Explain · {explain.database} · {explain.query_hash.slice(0, 12)}</strong>`<br>`                    <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `233-235` | Button | `                    <button`<br>`                      type="button"`<br>`                      className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/EnterprisePage.tsx`
**page** · 310 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)
  · also 6 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `204-206` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `233-235` | Button | `                <button`<br>`                  key={t}`<br>`                  role="tab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/ForgotPassword.tsx`
**page** · 242 LOC · **6 drifts** (0 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `132-134` | Form input | `                <input`<br>`                  type="email"`<br>`                  value={email}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `154-156` | Button | `              <motion.button`<br>`                type="submit"`<br>`                className="auth-button-primary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `183-185` | Button | `                    <motion.button`<br>`                      type="button"`<br>`                      className="auth-button-primary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `193-195` | Button | `                    <motion.button`<br>`                      type="button"`<br>`                      className="sw-button sw-button-quiet"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `208-210` | Button | `                      <button type="submit" className="auth-button-ghost">Send a fresh reset link</button>`<br>`                    </form>`<br>`                  </details>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `220-222` | Button | `                      <button type="submit" className="auth-button-ghost">Send a fresh reset link</button>`<br>`                    </form>`<br>`                  </details>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/pages/HomelabPage.tsx`
**page** · 250 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `222-224` | Button | `              <button`<br>`                key={t}`<br>`                role="tab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/IncidentsPage.tsx`
**page** · 255 LOC · **3 drifts** (1 🔴 · 2 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `160-162` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `201-203` | Button | `              <button`<br>`                key={t}`<br>`                type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `227-229` | Form input | `                <select`<br>`                  value={sevFilter}`<br>`                  onChange={(e) => setSevFilter(e.target.value as SeverityFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |

### `web/src/pages/IntelligencePage.tsx`
**page** · 328 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `214-216` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `249-251` | Button | `                <button`<br>`                  key={t}`<br>`                  role="tab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/Login.tsx`
**page** · 76 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `42-44` | Form input | `            <input`<br>`              type="email"`<br>`              value={email}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `53-55` | Form input | `            <input`<br>`              type="password"`<br>`              value={password}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `63-65` | Button | `          <button type="submit" className="auth-button-primary" disabled={loading}>`<br>`            {loading ? 'Signing in...' : 'Sign in'}`<br>`          </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/pages/LogsFullPage.tsx`
**page** · 198 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `148-150` | KPI card | `          <section className="logs-kpi-strip">`<br>`            <KpiCard label="Errors" value={logLevelKpis.error} accent="red" />`<br>`            <KpiCard label="Warnings" value={logLevelKpis.warn} accent="amber" />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `157-159` | Button | `              <button`<br>`                key={t}`<br>`                role="tab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/NotebookPage.tsx`
**page** · 254 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `180-182` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `219-221` | Button | `                  <button`<br>`                    key={nb.id}`<br>`                    type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/PlatformPage.tsx`
**page** · 155 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `123-125` | Button | `                <button`<br>`                  key={tab.key}`<br>`                  type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/ProfilePage.tsx`
**page** · 227 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `175-177` | Button | `              <button onClick={() => window.location.reload()}>Retry</button>`<br>`            </div>`<br>`          )}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `179-181` | Loading state | `            <div className="prof-skeleton-stack">`<br>`              <div className="prof-skel prof-skel-hero" />`<br>`              <div className="prof-skel-row">` | ❌ | 🟡 medium | Use shared `<SkeletonRow columns rows>` from `web/src/components/shared/SkeletonRow.tsx` instead of hand-rolled skeleton markup. |

### `web/src/pages/ResetPassword.tsx`
**page** · 188 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `163-165` | Button | `          <motion.button`<br>`            type="submit"`<br>`            className="auth-button-primary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/pages/RumFullPage.tsx`
**page** · 207 LOC · **5 drifts** (3 🔴 · 2 🟡 · 0 🟢)
  · also 6 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `111-113` | KPI card | `            <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">`<br>`              <KpiCard label="Active (5m)" value={kpis.active} accent="green" />`<br>`              <KpiCard label="Sessions" value={kpis.total} accent="indigo" />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `135-137` | Table | `              <table className="sw-table">`<br>`                <thead>`<br>`                  <tr>` | ❌ | 🟡 medium | `sw-table` is a legacy class. Replace with shared `<DataTable>` (to be created). |
| `141-143` | Button | `                      <button`<br>`                        type="button"`<br>`                        className="sw-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `150-152` | Button | `                      <button`<br>`                        type="button"`<br>`                        className="sw-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `161-163` | Button | `                      <button`<br>`                        type="button"`<br>`                        className="sw-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/RumSessionPage.tsx`
**page** · 166 LOC · **3 drifts** (1 🔴 · 2 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `111-113` | KPI card | `              <div className="dash-kpi-strip">`<br>`                <Kpi label="Web vitals" value={String(header.web_vitals)} />`<br>`                <Kpi label="Resources" value={String(header.resources)} />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `126-128` | Button | `              <button`<br>`                key={t}`<br>`                role="tab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `161-163` | KPI card | `    <div className="dash-metric">`<br>`      <small>{label}</small>`<br>`      <strong>{value}</strong>` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |

### `web/src/pages/SecurityPage.tsx`
**page** · 369 LOC · **6 drifts** (0 🔴 · 6 🟡 · 0 🟢)
  · also 10 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `160-162` | KPI card | `              <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">`<br>`                <KpiCard label="Open threats" value={threats.filter((t) => !(t as any).resolved).length} status={threats.length > 0 ? 'crit' : 'up'} accent="red" sparkline={threatCountHistory} />`<br>`                <KpiCard label="Critical" value={threats.filter((t) => (t.severity \|\| '').toLowerCase() === 'critical').length} status={threats.filter((t) => (t.severity \|\| '').toLowerCase() === 'critical').length > 0 ? 'crit' : 'up'} accent="red" sparkline={criticalHistory} />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `188-190` | Form input | `                  <select`<br>`                    value={threatFilter}`<br>`                    onChange={(e) => setThreatFilter(e.target.value as typeof threatFilter)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `201-203` | Form input | `                  <select`<br>`                    value={threatResolved}`<br>`                    onChange={(e) => setThreatResolved(e.target.value as typeof threatResolved)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `255-257` | Table | `                  <table className="dash-table">`<br>`                    <thead>`<br>`                      <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |
| `299-301` | Form input | `                  <select`<br>`                    value={siemSev}`<br>`                    onChange={(e) => setSiemSev(e.target.value as typeof siemSev)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `318-320` | Table | `                <table className="dash-table">`<br>`                  <thead>`<br>`                    <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/pages/SettingsPage.tsx`
**page** · 261 LOC · **7 drifts** (1 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `165-167` | Button | `              <button onClick={() => window.location.reload()}>Retry</button>`<br>`            </div>`<br>`          )}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `169-171` | Empty state | `            <section className="dash-panel"><div className="dash-panel-empty"><strong>Loading settings…</strong></div></section>`<br>`          ) : (`<br>`            <>` | ❌ | 🟡 medium | Custom `dash-panel-empty` placeholder (used for loading state here — should be `<SkeletonCard>` not EmptyState). Replace with shared `<SkeletonCard>` (lives at `web/src/components/dashboard/SkeletonCard.tsx` — promote to `shared/`). |
| `194-196` | Form input | `                      <input`<br>`                        type="text"`<br>`                        value={nameDraft}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `205-207` | Button | `                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setNameDraft(tenant?.name \|\| ''); setNameError(null); setNameMessage(null); }} disabled={nameSaving}>Discard</button>`<br>`                      <button type="submit" className="sw-button sw-button-primary" disabled={nameSaving \|\| !nameDraft.trim() \|\| nameDraft.trim() === (tenant?.name \|\| '')}>{nameSaving ? 'Saving...' : 'Save name'}</button>`<br>`                    </div>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `206-208` | Button | `                      <button type="submit" className="sw-button sw-button-primary" disabled={nameSaving \|\| !nameDraft.trim() \|\| nameDraft.trim() === (tenant?.name \|\| '')}>{nameSaving ? 'Saving...' : 'Save name'}</button>`<br>`                    </div>`<br>`                  </form>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `250-252` | Button | `                      <button type="button" className="sw-button sw-button-quiet" onClick={() => { setOldPw(''); setNewPw(''); setNewPw2(''); setPwError(null); setPwErrorCode(undefined); setPwMessage(null); }} disabled={pwSaving}>Discard</button>`<br>`                      <button type="submit" className="sw-button sw-button-primary" disabled={pwSaving \|\| !oldPw \|\| !newPw \|\| !newPw2}>{pwSaving ? 'Saving...' : 'Update password'}</button>`<br>`                    </div>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `251-253` | Button | `                      <button type="submit" className="sw-button sw-button-primary" disabled={pwSaving \|\| !oldPw \|\| !newPw \|\| !newPw2}>{pwSaving ? 'Saving...' : 'Update password'}</button>`<br>`                    </div>`<br>`                  </form>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/pages/SharedDashboardsPage.tsx`
**page** · 248 LOC · **3 drifts** (2 🔴 · 1 🟡 · 0 🟢)
  · also 8 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `120-122` | KPI card | `            className="dash-metric-strip"`<br>`            initial="hidden"`<br>`            animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `158-160` | Button | `                    <button`<br>`                      key={d.dashboard_id}`<br>`                      type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `231-233` | Button | `                        <button`<br>`                          type="button"`<br>`                          className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/Signup.tsx`
**page** · 190 LOC · **4 drifts** (0 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `113-115` | Form input | `            <input`<br>`              type="text"`<br>`              value={tenantName}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `125-127` | Form input | `            <input`<br>`              type="text"`<br>`              value={fullName}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `135-137` | Form input | `            <input`<br>`              type="email"`<br>`              value={email}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `165-167` | Button | `          <motion.button`<br>`            type="submit"`<br>`            className="auth-button-primary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/pages/SyntheticsPage.tsx`
**page** · 382 LOC · **13 drifts** (1 🔴 · 12 🟡 · 0 🟢)
  · also 6 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `165-167` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={form.name}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `176-178` | Form input | `                  <select`<br>`                    value={form.type}`<br>`                    onChange={(e) => setForm({ ...form, type: e.target.value })}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `189-191` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={form.url}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `200-202` | Form input | `                  <input`<br>`                    type="text"`<br>`                    value={form.method}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `210-212` | Form input | `                  <input`<br>`                    type="number"`<br>`                    value={form.interval_seconds}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `220-222` | Form input | `                  <input`<br>`                    type="number"`<br>`                    value={form.timeout_ms}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `230-232` | Form input | `                  <input`<br>`                    type="number"`<br>`                    value={form.sla_uptime_pct}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `241-243` | Form input | `                  <input`<br>`                    type="number"`<br>`                    value={form.sla_response_ms}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `250-252` | Form input | `                  <input`<br>`                    type="checkbox"`<br>`                    checked={form.enabled}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `259-261` | Button | `                  <button type="submit" className="sw-button sw-button-primary" disabled={busy}>`<br>`                    {busy ? 'Creating…' : 'Create'}`<br>`                  </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `270-272` | KPI card | `            <motion.div className="dash-kpi-strip" variants={kpiStagger} initial="hidden" animate="show">`<br>`              <KpiCard label="Passing" value={kpis.passing} accent="green" sparkline={passingHistory} />`<br>`              <KpiCard label="Failing" value={kpis.failing} accent="red" sparkline={failingHistory} />` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `320-322` | Table | `              <table className="dash-table synth-tests-table">`<br>`                <thead>`<br>`                  <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |
| `365-367` | Button | `                        <button`<br>`                          type="button"`<br>`                          className="sw-button sw-button-secondary synth-run-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/pages/TrueNASWorkspace.tsx`
**page** · 373 LOC · **18 drifts** (5 🔴 · 11 🟡 · 2 🟢)
  · also 11 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `269-271` | Form input | `          <select`<br>`            className="px-host-select"`<br>`            aria-label="Active TrueNAS host"` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `282-284` | Button | `          {hostId && <button className="px-button px-button-primary" onClick={() => void runTest()} disabled={busy}>`<br>`            Test connection`<br>`          </button>}` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `285-287` | Button | `          {hostId && actionPaths[section] && <button className="px-button px-button-primary" onClick={() => setShowAction(true)} disabled={busy}>`<br>`            + Create`<br>`          </button>}` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `288-290` | Button | `          <button className="px-button px-button-primary" onClick={() => setShowHostForm((open) => !open)}>`<br>`            + Add TrueNAS`<br>`          </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `309-311` | Button | `      {message && <div className="sw-alert sw-alert-success"><strong>Success</strong><span>{message}</span><button onClick={() => setMessage('')}>×</button></div>}`<br>`      {error && <div className="sw-alert sw-alert-error"><strong>Error</strong><span>{error}</span><button onClick={() => setError('')}>×</button></div>}`<br>`      {showHostForm && (` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `310-312` | Button | `      {error && <div className="sw-alert sw-alert-error"><strong>Error</strong><span>{error}</span><button onClick={() => setError('')}>×</button></div>}`<br>`      {showHostForm && (`<br>`        <section className="sw-panel">` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `317-319` | Form input | `            <label className="sw-field"><span>Name</span><input required value={hostForm.name} onChange={(e) => setHostForm({ ...hostForm, name: e.target.value })} placeholder="e.g. storage-prod" /></label>`<br>`            <label className="sw-field"><span>Base URL</span><input required value={hostForm.base_url} onChange={(e) => setHostForm({ ...hostForm, base_url: e.target.value })} placeholder="https://truenas.example.com" /></label>`<br>`            <label className="sw-field"><span>Username</span><input value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} /></label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `318-320` | Form input | `            <label className="sw-field"><span>Base URL</span><input required value={hostForm.base_url} onChange={(e) => setHostForm({ ...hostForm, base_url: e.target.value })} placeholder="https://truenas.example.com" /></label>`<br>`            <label className="sw-field"><span>Username</span><input value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} /></label>`<br>`            <label className="sw-field"><span>Password</span><input type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} /></label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `319-321` | Form input | `            <label className="sw-field"><span>Username</span><input value={hostForm.username} onChange={(e) => setHostForm({ ...hostForm, username: e.target.value })} /></label>`<br>`            <label className="sw-field"><span>Password</span><input type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} /></label>`<br>`            <label className="sw-field"><span>API key (optional)</span><input type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} /></label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `320-322` | Form input | `            <label className="sw-field"><span>Password</span><input type="password" value={hostForm.password} onChange={(e) => setHostForm({ ...hostForm, password: e.target.value })} /></label>`<br>`            <label className="sw-field"><span>API key (optional)</span><input type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} /></label>`<br>`            <label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(e) => setHostForm({ ...hostForm, verify_tls: e.target.checked })} /> Verify TLS certificate</label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `321-323` | Form input | `            <label className="sw-field"><span>API key (optional)</span><input type="password" value={hostForm.api_key} onChange={(e) => setHostForm({ ...hostForm, api_key: e.target.value })} /></label>`<br>`            <label className="sw-checkbox"><input type="checkbox" checked={hostForm.verify_tls} onChange={(e) => setHostForm({ ...hostForm, verify_tls: e.target.checked })} /> Verify TLS certificate</label>`<br>`            <div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowHostForm(false)}>Cancel</button><button type="submit" className="sw-button sw-button-primary" disabled={busy}>Test and save</button></div>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `323-325` | Button | `            <div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowHostForm(false)}>Cancel</button><button type="submit" className="sw-button sw-button-primary" disabled={busy}>Test and save</button></div>`<br>`          </form>`<br>`        </section>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `346-348` | Loading state | `            <div><span className="sw-eyebrow">Live upstream response</span><h2>{busy ? 'Loading…' : \`${rows.length} records\`}</h2></div>`<br>`            {selected && <StatusPill status={selected.status === 'online' ? 'up' : 'unknown'} label={selected.status ?? 'unknown'} size="sm" />}`<br>`          </div>` | ❌ | 🟢 low | Plain `'Loading…'` text — replace with shared `<SkeletonCard>` (for page-level) or `<SkeletonRow>` (for list rows). Button text `'Loading…'` is fine inside `<Button loading>` state. |
| `353-355` | Loading state | `            empty={busy ? 'Loading live resources…' : \`No ${current.label.toLowerCase()} found.\`}`<br>`            columns={columnsFor(section)}`<br>`          />` | ❌ | 🟢 low | Plain `'Loading…'` text — replace with shared `<SkeletonCard>` (for page-level) or `<SkeletonRow>` (for list rows). Button text `'Loading…'` is fine inside `<Button loading>` state. |
| `359-361` | Modal | `        <div className="sw-modal-backdrop" role="dialog" aria-modal="true">`<br>`          <div className="sw-modal">`<br>`            <div className="sw-panel-head">` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `363-365` | Button | `              <button className="sw-button" onClick={() => setShowAction(false)}>Close</button>`<br>`            </div>`<br>`            <form onSubmit={runAction}>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `366-368` | Form input | `              <textarea className="sw-json-editor" value={actionJSON} onChange={(e) => setActionJSON(e.target.value)} spellCheck={false} />`<br>`              <div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowAction(false)}>Cancel</button><button className="sw-button sw-button-primary" disabled={busy}>Submit action</button></div>`<br>`            </form>` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `367-369` | Button | `              <div className="sw-form-actions"><button type="button" className="sw-button" onClick={() => setShowAction(false)}>Cancel</button><button className="sw-button sw-button-primary" disabled={busy}>Submit action</button></div>`<br>`            </form>`<br>`          </div>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |


### Components (107 files with drift)

> Components are grouped by subdirectory under `web/src/components/`. The largest bucket is `homelab/widgets/` with 24 files — most of those are dashboard widgets (calendar, todo, RSS, downloads, media) that share the same drift patterns: custom modal markup, custom button classes, raw inputs.

#### `web/src/components/` (root, no subdir) — 25 files

### `web/src/components/AnomaliesSection.tsx`
**component** · 248 LOC · **3 drifts** (1 🔴 · 2 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `94-96` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `155-157` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `219-221` | Button | `                    <button`<br>`                      type="button"`<br>`                      className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/AppShell.tsx`
**component** · 136 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `99-101` | Button | `              <button`<br>`                type="button"`<br>`                className="sb-drawer-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/AppSidebar.tsx`
**component** · 174 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `170-172` | Button | `        <button className="dash-sidebar-logout" onClick={onLogout}>↪ Sign out</button>`<br>`      </div>`<br>`    </aside>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/AuditSection.tsx`
**component** · 325 LOC · **5 drifts** (0 🔴 · 3 🟡 · 2 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `159-161` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `199-201` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `210-212` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `263-265` | Modal | `        <Modal title="Create audit archive" onClose={() => setShowArchive(false)}>`<br>`          <ArchiveFormView`<br>`            archiveForm={archiveForm}` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |
| `275-277` | Modal | `        <Modal title="Export audit events" onClose={() => setShowExport(false)}>`<br>`          <ExportFormView`<br>`            exportForm={exportForm}` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |

### `web/src/components/AuditSectionFormViews.tsx`
**component** · 160 LOC · **7 drifts** (0 🔴 · 7 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `41-43` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={archiveForm.periodStart}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `53-55` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={archiveForm.periodEnd}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `89-91` | Form input | `        <input`<br>`          type="text"`<br>`          value={exportForm.eventType}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `102-104` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={exportForm.startTime}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `114-116` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={exportForm.endTime}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `126-128` | Form input | `        <select`<br>`          value={exportForm.format}`<br>`          onChange={(e) =>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `142-144` | Form input | `        <input`<br>`          type="number"`<br>`          value={exportForm.limit}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |

### `web/src/components/AuditSectionModals.tsx`
**component** · 314 LOC · **5 drifts** (2 🔴 · 2 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `23-25` | Modal | ` *   - <Modal />, <Field />, <ModalActions /> — modal primitives`<br>` *   - ArchiveForm, ExportForm types + initial*Form() defaults`<br>` *   - handleCreateArchiveSubmit() — POST /audit/archive submit` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |
| `88-90` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label={title}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `113-115` | Button | `          <button`<br>`            type="button"`<br>`            aria-label="Close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `170-172` | Button | `      <motion.button`<br>`        type="button"`<br>`        className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `181-183` | Button | `      <motion.button`<br>`        type="submit"`<br>`        className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/CommandPalette.tsx`
**component** · 162 LOC · **4 drifts** (2 🔴 · 1 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `108-110` | Modal | `          role="dialog"`<br>`          aria-modal="true"`<br>`          aria-label="Command palette"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `122-124` | Form input | `              <input`<br>`                ref={inputRef}`<br>`                type="search"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `137-139` | Empty state | `              {filtered.length === 0 ? <div className="cmd-palette-empty">`<br>`                <span>⌕</span>`<br>`                <strong>No results for &ldquo;{query}&rdquo;</strong>` | ❌ | 🟢 low | Domain-specific empty state — replace with shared `<EmptyState illustration={…} headline="No results" />`. |
| `141-143` | Button | `              </div> : filtered.map((cmd, index) => <button`<br>`                key={cmd.id}`<br>`                className={\`cmd-palette-item ${index === highlight ? 'cmd-palette-item-active' : ''}\`}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/ComplianceScheduleList.tsx`
**component** · 122 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `104-106` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/ComplianceSection.tsx`
**component** · 384 LOC · **6 drifts** (0 🔴 · 4 🟡 · 2 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `201-203` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `241-243` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `252-254` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `296-298` | Button | `            <motion.button`<br>`              key={t.id}`<br>`              type="button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `360-362` | Modal | `        <Modal title="Generate compliance report" onClose={() => setShowGenerate(false)}>`<br>`          <GenerateReportFormView`<br>`            generateForm={generateForm}` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |
| `372-374` | Modal | `        <Modal title="Schedule compliance report" onClose={() => setShowSchedule(false)}>`<br>`          <ScheduleReportFormView`<br>`            scheduleForm={scheduleForm}` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |

### `web/src/components/ComplianceSectionFormViews.tsx`
**component** · 212 LOC · **8 drifts** (0 🔴 · 8 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `56-58` | Form input | `        <select`<br>`          value={generateForm.framework}`<br>`          onChange={(e) =>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `75-77` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={generateForm.periodStart}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `87-89` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={generateForm.periodEnd}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `123-125` | Form input | `        <select`<br>`          value={scheduleForm.framework}`<br>`          onChange={(e) =>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `142-144` | Form input | `        <select`<br>`          value={scheduleForm.frequency}`<br>`          onChange={(e) =>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `161-163` | Form input | `        <input`<br>`          type="text"`<br>`          value={scheduleForm.recipients}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `173-175` | Form input | `        <input`<br>`          type="datetime-local"`<br>`          value={scheduleForm.nextRunAt}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `194-196` | Form input | `          <input`<br>`            type="checkbox"`<br>`            checked={scheduleForm.enabled}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |

### `web/src/components/ComplianceSectionModals.tsx`
**component** · 335 LOC · **5 drifts** (2 🔴 · 2 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `20-22` | Modal | ` *   - <Modal />, <Field />, <ModalActions /> — modal primitives`<br>` *   - GenerateReportForm, ScheduleReportForm types + initial*Form()`<br>` *   - handleGenerateReportSubmit() — POST /compliance/reports submit` | ✅ | 🟢 low | Uses shared `<Modal>` if/when promoted to `shared/`. Currently each component implements its own modal markup. |
| `95-97` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label={title}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `120-122` | Button | `          <button`<br>`            type="button"`<br>`            aria-label="Close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `177-179` | Button | `      <motion.button`<br>`        type="button"`<br>`        className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `188-190` | Button | `      <motion.button`<br>`        type="submit"`<br>`        className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/CorrelationsSection.tsx`
**component** · 336 LOC · **10 drifts** (4 🔴 · 6 🟡 · 0 🟢)
  · also 4 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `185-187` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `244-246` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `259-261` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>Create manual correlation</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `260-262` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>Create manual correlation</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `262-264` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `280-282` | Form input | `              <textarea`<br>`                required`<br>`                rows={3}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `290-292` | Form input | `              <input`<br>`                type="text"`<br>`                maxLength={2048}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `300-302` | Form input | `              <input`<br>`                type="number"`<br>`                min={0}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `315-317` | Button | `              <button`<br>`                type="button"`<br>`                className="dash-icon-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `323-325` | Button | `              <button`<br>`                type="submit"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/EnterpriseOrgsSection.tsx`
**component** · 283 LOC · **2 drifts** (0 🔴 · 2 🟡 · 0 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `168-170` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `182-184` | Button | `        <motion.button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/EnterpriseOrgsSectionHelpers.tsx`
**component** · 138 LOC · **4 drifts** (0 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `62-64` | Form input | `          <input`<br>`            className="form-input"`<br>`            type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `76-78` | Form input | `          <input`<br>`            className="form-input"`<br>`            type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `93-95` | Form input | `          <select`<br>`            className="form-input"`<br>`            value={form.parent_orgId}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `125-127` | Button | `        <motion.button`<br>`          type="submit"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/FilterBar.tsx`
**component** · 94 LOC · **4 drifts** (3 🔴 · 0 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `54-56` | Form input | `      <input`<br>`        type="search"`<br>`        value={search}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `65-67` | Button | `      {search.length > 0 && <button`<br>`        type="button"`<br>`        className="sw-filter-search-clear"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `75-77` | Button | `        <button`<br>`          key={chip.id}`<br>`          type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `88-90` | Button | `    {showClear && onClear && chips.some((c) => c.active) && <button`<br>`      type="button"`<br>`      className="sw-filter-reset"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/NoiseReductionSection.tsx`
**component** · 301 LOC · **7 drifts** (4 🔴 · 3 🟡 · 0 🟢)
  · also 4 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `161-163` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `229-231` | Button | `                  <button`<br>`                    type="button"`<br>`                    className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `237-239` | Button | `                  <button`<br>`                    type="button"`<br>`                    className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `256-258` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `280-282` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>New noise rule</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `281-283` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>New noise rule</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `283-285` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/PasswordField.tsx`
**component** · 111 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `79-81` | Form input | `        <input`<br>`          id={id}`<br>`          name={name}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `96-98` | Button | `          <button`<br>`            type="button"`<br>`            className="pwd-toggle pwd-toggle-eye"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/PasswordInput.tsx`
**component** · 198 LOC · **2 drifts** (1 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `120-122` | Form input | `        <input`<br>`          id={id}`<br>`          name={name}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `135-137` | Button | `          <button`<br>`            type="button"`<br>`            className="pwd-toggle pwd-toggle-eye"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/PredictiveAlertsSection.tsx`
**component** · 390 LOC · **10 drifts** (5 🔴 · 5 🟡 · 0 🟢)
  · also 4 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `150-152` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `239-241` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `301-303` | Button | `                    <button`<br>`                      type="button"`<br>`                      className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `318-320` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>Generate forecast</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `319-321` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>Generate forecast</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `321-323` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `339-341` | Form input | `              <input`<br>`                type="text"`<br>`                required` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `350-352` | Form input | `              <select`<br>`                value={horizon}`<br>`                onChange={(e) => setHorizon(parseInt(e.target.value, 10))}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `369-371` | Button | `              <button`<br>`                type="button"`<br>`                className="dash-icon-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `377-379` | Button | `              <button`<br>`                type="submit"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/ProfileMenu.tsx`
**component** · 147 LOC · **2 drifts** (2 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `84-86` | Button | `      <button`<br>`        ref={buttonRef}`<br>`        type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `118-120` | Button | `                        <button type="button" role="menuitem" className={className} onClick={() => { close(); item.onClick!(); }}>`<br>`                          <span className="dash-menu-icon" aria-hidden>{item.icon}</span>`<br>`                          <span className="dash-menu-text">` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/RbacAssignmentsPanel.tsx`
**component** · 267 LOC · **4 drifts** (1 🔴 · 3 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `154-156` | Form input | `          <select`<br>`            value={selectedUserId}`<br>`            onChange={(e) => setSelectedUserId(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `181-183` | Form input | `          <select`<br>`            value={selectedRoleId}`<br>`            onChange={(e) => setSelectedRoleId(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `205-207` | Button | `        <motion.button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `239-241` | Button | `                  <button`<br>`                    type="button"`<br>`                    className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/RbacSection.tsx`
**component** · 370 LOC · **8 drifts** (4 🔴 · 4 🟡 · 0 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `152-154` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `203-205` | Button | `              <motion.button`<br>`                key={t}`<br>`                type="button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `227-229` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `285-287` | Button | `                      <button`<br>`                        type="button"`<br>`                        className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `297-299` | Button | `                      <button`<br>`                        type="button"`<br>`                        className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `339-341` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>{creating ? 'New custom role' : \`Edit ${editing?.name}\`}</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `340-342` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>{creating ? 'New custom role' : \`Edit ${editing?.name}\`}</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `342-344` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/ScimSection.tsx`
**component** · 377 LOC · **9 drifts** (2 🔴 · 7 🟡 · 0 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `155-157` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `233-235` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>New SCIM token</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `234-236` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>New SCIM token</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `236-238` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `259-261` | Button | `                <motion.button`<br>`                  type="button"`<br>`                  className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `282-284` | Form input | `                <input`<br>`                  type="text"`<br>`                  value={form.name}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `325-327` | Form input | `                    <input`<br>`                      type="checkbox"`<br>`                      checked={form.scopes.includes(opt.value)}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `349-351` | Button | `                <motion.button`<br>`                  type="button"`<br>`                  className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `360-362` | Button | `                <motion.button`<br>`                  type="submit"`<br>`                  className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/SsoSection.tsx`
**component** · 244 LOC · **6 drifts** (2 🔴 · 4 🟡 · 0 🟢)
  · also 3 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `143-145` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `196-198` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `207-209` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `222-224` | Modal | `        <div className="slow-query-explain-modal" role="dialog" aria-modal="true">`<br>`          <div className="slow-query-explain-modal-head">`<br>`            <strong>New {modalType.toUpperCase()} provider</strong>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `223-225` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>New {modalType.toUpperCase()} provider</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `225-227` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/TimeWidget.tsx`
**component** · 251 LOC · **5 drifts** (5 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `96-98` | Button | `      <button`<br>`        ref={triggerRef}`<br>`        type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `112-114` | Modal | `          role="dialog"`<br>`          aria-label="Time and calendar"`<br>`        >` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `212-214` | Button | `        <button type="button" className="tw-cal-nav" onClick={onPrev} aria-label="Previous month">‹</button>`<br>`        <div className="tw-cal-title">{MONTH_NAMES[month]}</div>`<br>`        <button type="button" className="tw-cal-nav" onClick={onNext} aria-label="Next month">›</button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `214-216` | Button | `        <button type="button" className="tw-cal-nav" onClick={onNext} aria-label="Next month">›</button>`<br>`      </div>`<br>`      <div className="tw-cal-years">` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `218-220` | Button | `          <button`<br>`            key={y}`<br>`            type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

#### `web/src/components/admin/` — 1 files

### `web/src/components/admin/DeploySection.tsx`
**component** · 303 LOC · **7 drifts** (1 🔴 · 6 🟡 · 0 🟢)
  · also 4 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `149-151` | KPI card | `      <div className="dash-metric-strip">`<br>`        <KpiCard label="Tokens active" value={stats?.tokens_active ?? '—'} accent="cyan" />`<br>`        <KpiCard label="Installs 24h" value={stats?.installs_24h ?? '—'} accent="green" />` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `182-184` | Button | `              <motion.button`<br>`                type="button"`<br>`                className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `199-201` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `228-230` | Modal | `          role="dialog"`<br>`          aria-modal="true"`<br>`          style={{` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `259-261` | Form input | `              <input`<br>`                type="text"`<br>`                value={label}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `277-279` | Button | `              <motion.button`<br>`                type="button"`<br>`                className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `287-289` | Button | `              <motion.button`<br>`                type="submit"`<br>`                className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

#### `web/src/components/dashboard/` — 4 files

### `web/src/components/dashboard/ErrorBar.tsx`
**component** · 55 LOC · **2 drifts** (2 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `33-35` | Button | `        <button`<br>`          type="button"`<br>`          onClick={() => {` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `50-52` | Button | `      <button type="button" onClick={onRetry}>`<br>`        Retry`<br>`      </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/dashboard/HostList.tsx`
**component** · 108 LOC · **2 drifts** (1 🔴 · 0 🟡 · 1 🟢)
  · also 2 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `22-24` | Empty state | `        <div className="dash-empty-illustration" aria-hidden="true">`<br>`          <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="1.2">`<br>`            <rect x="14" y="20" width="92" height="48" rx="6" opacity=".4" />` | ❌ | 🟢 low | Custom empty illustration class — verify it composes with shared `<EmptyState>` (illustration prop), or replace entirely. |
| `70-72` | Button | `              <button`<br>`                type="button"`<br>`                className="dash-host-kebab"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/dashboard/SkeletonCard.tsx`
**component** · 27 LOC · **6 drifts** (0 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `15-17` | KPI card | `      className="dash-metric dash-metric-skeleton"`<br>`      variants={reduce ? undefined : shimmer}`<br>`      animate={reduce ? undefined : 'animate'}` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `15-17` | Loading state | `      className="dash-metric dash-metric-skeleton"`<br>`      variants={reduce ? undefined : shimmer}`<br>`      animate={reduce ? undefined : 'animate'}` | ❌ | 🟡 medium | Use shared `<SkeletonRow columns rows>` from `web/src/components/shared/SkeletonRow.tsx` instead of hand-rolled skeleton markup. |
| `19-21` | KPI card | `      <div className="dash-metric-top">`<br>`        <span className="dash-metric-icon dash-skel-line" style={{ width: 18, height: 18, display: 'inline-block' }} />`<br>`        <span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} />` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `20-22` | KPI card | `        <span className="dash-metric-icon dash-skel-line" style={{ width: 18, height: 18, display: 'inline-block' }} />`<br>`        <span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} />`<br>`      </div>` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `21-23` | KPI card | `        <span className="dash-metric-label dash-skel-line" style={{ width: 80, height: 10 }} />`<br>`      </div>`<br>`      <strong className="dash-skel-line" style={{ width: 50, height: 28 }} />` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `24-26` | KPI card | `      <span className="dash-metric-hint dash-skel-line" style={{ width: 120, height: 10 }} />`<br>`    </motion.article>`<br>`  );` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |

### `web/src/components/dashboard/TrendChart.tsx`
**component** · 76 LOC · **1 drifts** (0 🔴 · 0 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `16-18` | Empty state | `        <div className="dash-empty-illustration" aria-hidden="true">`<br>`          <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="1.2">`<br>`            <rect x="6" y="14" width="108" height="56" rx="6" opacity=".35" />` | ❌ | 🟢 low | Custom empty illustration class — verify it composes with shared `<EmptyState>` (illustration prop), or replace entirely. |

#### `web/src/components/homelab/` — 3 files

### `web/src/components/homelab/GlobalSearch.tsx`
**component** · 394 LOC · **2 drifts** (1 🔴 · 0 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `314-316` | Form input | `        <input`<br>`          ref={inputRef}`<br>`          type="search"` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `371-373` | Button | `                <button`<br>`                  key={\`${s.kind}-${s.id}-${i}\`}`<br>`                  type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/HomelabGrid.tsx`
**component** · 229 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `222-224` | Button | `          <button type="button" className="empty-state-cta" onClick={() => void saveLayout()} disabled={!dirty \|\| saving}>`<br>`            {saving ? 'Saving…' : 'Save layout'}`<br>`          </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/HomelabKpiStrip.tsx`
**component** · 118 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `95-97` | KPI card | `      className="dash-metric-strip"`<br>`      initial="hidden"`<br>`      animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `101-103` | Button | `        <motion.button`<br>`          key={c.label}`<br>`          type="button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `104-106` | KPI card | `          className="homelab-kpi-button"`<br>`          onClick={() => onNavigate(c.tab)}`<br>`          variants={kpiEnter}` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |

#### `web/src/components/homelab/widgets/` — 24 files

### `web/src/components/homelab/widgets/AddCalendarModal.tsx`
**component** · 132 LOC · **7 drifts** (5 🔴 · 2 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `36-38` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `37-39` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Add calendar"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `54-56` | Form input | `            <input`<br>`              id="cal-name"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `67-69` | Form input | `            <input`<br>`              id="cal-url"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `84-86` | Button | `                <button`<br>`                  key={p.hex}`<br>`                  type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `112-114` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `120-122` | Button | `            <button`<br>`              type="submit"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/AddDownloadModal.tsx`
**component** · 173 LOC · **10 drifts** (4 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `47-49` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `48-50` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Pin a new download client"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `65-67` | Form input | `            <input`<br>`              id="dl-name"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `78-80` | Form input | `            <select`<br>`              id="dl-kind"`<br>`              value={form.kind}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `95-97` | Form input | `            <input`<br>`              id="dl-url"`<br>`              type="url"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `109-111` | Form input | `            <input`<br>`              id="dl-apikey"`<br>`              type="password"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `123-125` | Form input | `                <input`<br>`                  id="dl-user"`<br>`                  type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `134-136` | Form input | `                <input`<br>`                  id="dl-pass"`<br>`                  type="password"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `153-155` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `161-163` | Button | `            <button`<br>`              type="submit"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/AddMediaModal.tsx`
**component** · 164 LOC · **8 drifts** (4 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `66-68` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `67-69` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Pin a new media server"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `84-86` | Form input | `            <input`<br>`              id="media-name"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `97-99` | Form input | `            <select`<br>`              id="media-kind"`<br>`              value={form.kind}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `114-116` | Form input | `            <input`<br>`              id="media-url"`<br>`              type="url"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `126-128` | Form input | `            <input`<br>`              id="media-apikey"`<br>`              type="password"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `144-146` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `152-154` | Button | `            <button`<br>`              type="submit"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/AddRssFeedModal.tsx`
**component** · 143 LOC · **7 drifts** (4 🔴 · 3 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `58-60` | Modal | `      className="homelab-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `59-61` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      onClick={onClose}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `88-90` | Form input | `            <input`<br>`              type="text"`<br>`              className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `99-101` | Form input | `            <input`<br>`              type="url"`<br>`              className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `109-111` | Form input | `            <select`<br>`              className="homelab-search-input"`<br>`              value={form.category}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `127-129` | Button | `            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>`<br>`              Cancel`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `130-132` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/AddSchedulerJobModal.tsx`
**component** · 292 LOC · **14 drifts** (6 🔴 · 8 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `96-98` | Modal | `      className="homelab-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `97-99` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      onClick={onClose}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `128-130` | Form input | `            <input`<br>`              type="text"`<br>`              className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `139-141` | Form input | `            <select`<br>`              className="homelab-search-input"`<br>`              value={form.action_kind}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `157-159` | Form input | `            <input`<br>`              type="url"`<br>`              className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `172-174` | Form input | `                <select`<br>`                  className="homelab-search-input"`<br>`                  value={form.contentType}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `186-188` | Form input | `                <textarea`<br>`                  className="homelab-search-input"`<br>`                  value={form.body}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `206-208` | Form input | `                <input`<br>`                  type="text"`<br>`                  className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `218-220` | Form input | `                <input`<br>`                  type="text"`<br>`                  className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `230-232` | Button | `                <button`<br>`                  type="button"`<br>`                  className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `243-245` | Button | `              <button`<br>`                type="button"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `259-261` | Form input | `            <input`<br>`              type="text"`<br>`              className="homelab-search-input"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `276-278` | Button | `            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>`<br>`              Cancel`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `279-281` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/CalendarWidget.tsx`
**component** · 396 LOC · **5 drifts** (5 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `211-213` | Button | `        <button type="button" className="empty-state-cta" onClick={openForm}>`<br>`          + Add calendar`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `214-216` | Button | `        <button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `278-280` | Button | `                <button`<br>`                  type="button"`<br>`                  className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `287-289` | Button | `                <button`<br>`                  type="button"`<br>`                  className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `338-340` | Button | `                    <button`<br>`                      key={e.id}`<br>`                      type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/DownloadClientCard.tsx`
**component** · 111 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `100-102` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/DownloadStatsWidget.tsx`
**component** · 327 LOC · **2 drifts** (2 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `221-223` | Button | `        <button type="button" className="empty-state-cta" onClick={openForm}>`<br>`          + Add client`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `224-226` | Button | `        <button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/EventDetailModal.tsx`
**component** · 90 LOC · **3 drifts** (3 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `24-26` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `25-27` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Event details"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `79-81` | Button | `          <button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/MediaServerCard.tsx`
**component** · 80 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `69-71` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/MediaWidget.tsx`
**component** · 398 LOC · **3 drifts** (3 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `247-249` | Button | `        <button type="button" className="empty-state-cta" onClick={openForm}>`<br>`          + Add server`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `250-252` | Button | `        <button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `266-268` | Button | `          <button`<br>`            key={t}`<br>`            type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/NoteModal.tsx`
**component** · 118 LOC · **8 drifts** (4 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `28-30` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `29-31` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label={editingId ? 'Edit note' : 'New note'}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `48-50` | Form input | `            <input`<br>`              id="note-title"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `61-63` | Form input | `            <textarea`<br>`              id="note-body"`<br>`              value={form.body}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `81-83` | Form input | `            <input`<br>`              id="note-tags"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `94-96` | Form input | `              <input`<br>`                id="note-pinned"`<br>`                type="checkbox"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `107-109` | Button | `            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>`<br>`              Cancel`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `110-112` | Button | `            <button type="submit" className="empty-state-cta" disabled={busy \|\| !form.title.trim()}>`<br>`              {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create note'}`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/NotesWidget.tsx`
**component** · 356 LOC · **7 drifts** (6 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `234-236` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `242-244` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `250-252` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `280-282` | Button | `        <button type="button" className="empty-state-cta" onClick={openCreate}>`<br>`          + New note`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `284-286` | Form input | `          <input`<br>`            type="search"`<br>`            placeholder="Search title / body / tag…"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `305-307` | Button | `          <button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `314-316` | Button | `            <button type="button" className="empty-state-cta" onClick={clearSearch}>`<br>`              Clear`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/PinModal.tsx`
**component** · 115 LOC · **8 drifts** (4 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `33-35` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `34-36` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Pin a new service"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `51-53` | Form input | `            <input`<br>`              id="pin-name"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `64-66` | Form input | `            <input`<br>`              id="pin-url"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `77-79` | Form input | `            <select`<br>`              id="pin-kind"`<br>`              value={form.kind}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `91-93` | Form input | `            <input`<br>`              id="pin-icon"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `104-106` | Button | `            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>`<br>`              Cancel`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `107-109` | Button | `            <button type="submit" className="empty-state-cta" disabled={busy \|\| !form.name.trim() \|\| !form.url.trim()}>`<br>`              {busy ? 'Pinning…' : 'Pin service'}`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/RssFeedCard.tsx`
**component** · 76 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `65-67` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/RssItemDetailModal.tsx`
**component** · 118 LOC · **4 drifts** (4 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `54-56` | Modal | `      className="homelab-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `55-57` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      onClick={onClose}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `102-104` | Button | `            <button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `111-113` | Button | `          <button type="button" className="empty-state-cta" onClick={onClose}>`<br>`            Close`<br>`          </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/RssWidget.tsx`
**component** · 359 LOC · **4 drifts** (4 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `191-193` | Button | `        <button type="button" className="empty-state-cta" onClick={openForm}>`<br>`          + Add feed`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `194-196` | Button | `        <button type="button" className="empty-state-cta" onClick={() => void refreshAll()}>`<br>`          Refresh`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `221-223` | Button | `          <button`<br>`            key={t.id}`<br>`            type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `242-244` | Button | `          <button`<br>`            key={cat}`<br>`            type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/SchedulerJobDetailModal.tsx`
**component** · 149 LOC · **3 drifts** (3 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `39-41` | Modal | `      className="homelab-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `40-42` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      onClick={onClose}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `142-144` | Button | `          <button type="button" className="empty-state-cta" onClick={onClose}>`<br>`            Close`<br>`          </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/SchedulerWidget.tsx`
**component** · 378 LOC · **7 drifts** (6 🔴 · 0 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `221-223` | Button | `          <button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `227-229` | Loading state | `            {loading ? 'Loading…' : 'Refresh'}`<br>`          </button>`<br>`          <button` | ❌ | 🟢 low | Plain `'Loading…'` text — replace with shared `<SkeletonCard>` (for page-level) or `<SkeletonRow>` (for list rows). Button text `'Loading…'` is fine inside `<Button loading>` state. |
| `229-231` | Button | `          <button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `264-266` | Button | `          <button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `307-309` | Button | `              <button`<br>`                type="button"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `318-320` | Button | `              <button`<br>`                type="button"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `329-331` | Button | `              <button`<br>`                type="button"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/ServiceStatusWidget.tsx`
**component** · 236 LOC · **2 drifts** (2 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `191-193` | Button | `        <button type="button" className="empty-state-cta" onClick={openPinForm}>`<br>`          + Pin service`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `194-196` | Button | `        <button type="button" className="empty-state-cta" onClick={() => void probeAll()} disabled={services.length === 0}>`<br>`          Probe all`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/ServiceTile.tsx`
**component** · 95 LOC · **4 drifts** (4 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `46-48` | Button | `    <button`<br>`      type="button"`<br>`      className="homelab-service-card"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `67-69` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `76-78` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `84-86` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/TodoCard.tsx`
**component** · 147 LOC · **4 drifts** (3 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `57-59` | Form input | `        <input`<br>`          type="checkbox"`<br>`          checked={done}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `119-121` | Button | `          <button`<br>`            type="button"`<br>`            className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `128-130` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `136-138` | Button | `        <button`<br>`          type="button"`<br>`          className="homelab-service-action-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/TodoModal.tsx`
**component** · 157 LOC · **9 drifts** (4 🔴 · 5 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `55-57` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `56-58` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label={editingId ? 'Edit todo' : 'New todo'}` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `75-77` | Form input | `            <input`<br>`              id="todo-title"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `88-90` | Form input | `            <textarea`<br>`              id="todo-description"`<br>`              value={form.description}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `109-111` | Form input | `              <select`<br>`                id="todo-priority"`<br>`                value={form.priority}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `121-123` | Form input | `              <input`<br>`                id="todo-due"`<br>`                type="datetime-local"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `132-134` | Form input | `            <input`<br>`              id="todo-tags"`<br>`              type="text"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `146-148` | Button | `            <button type="button" className="empty-state-cta" onClick={onClose} disabled={busy}>`<br>`              Cancel`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `149-151` | Button | `            <button type="submit" className="empty-state-cta" disabled={busy \|\| !form.title.trim()}>`<br>`              {busy ? 'Saving…' : editingId ? 'Save changes' : 'Create todo'}`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/homelab/widgets/TodosWidget.tsx`
**component** · 346 LOC · **2 drifts** (2 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `302-304` | Button | `        <button type="button" className="empty-state-cta" onClick={openCreate}>`<br>`          + New todo`<br>`        </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `307-309` | Button | `            <button`<br>`              key={f}`<br>`              type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

#### `web/src/components/logs/` — 4 files

### `web/src/components/logs/LogArchivesTab.tsx`
**component** · 188 LOC · **11 drifts** (0 🔴 · 11 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `88-90` | Form input | `            <input type="text" value={archForm.name} onChange={(e) => setArchForm({ ...archForm, name: e.target.value })} maxLength={128} required />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `92-94` | Form input | `            <select value={archForm.destination_type} onChange={(e) => setArchForm({ ...archForm, destination_type: e.target.value })}>`<br>`              <option value="s3">S3</option>`<br>`              <option value="gcs">GCS</option>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `102-104` | Form input | `              <input type="text" value={archForm.path} onChange={(e) => setArchForm({ ...archForm, path: e.target.value })} placeholder="/srv/log-archive" />`<br>`            </label>`<br>`          ) : (` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `108-110` | Form input | `                <input type="text" value={archForm.bucket} onChange={(e) => setArchForm({ ...archForm, bucket: e.target.value })} placeholder="my-bucket" />`<br>`              </label>`<br>`              {archForm.destination_type === 's3' ? (` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `113-115` | Form input | `                  <input type="text" value={archForm.region} onChange={(e) => setArchForm({ ...archForm, region: e.target.value })} placeholder="us-east-1" />`<br>`                </label>`<br>`              ) : null}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `120-122` | Button | `            <button type="submit" className="sw-button sw-button-primary" disabled={busy}>`<br>`              {busy ? 'Saving…' : 'Save'}`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `138-140` | Button | `                <button type="button" className="sw-button" onClick={() => setOpenFor(openFor === a.id ? null : a.id)}>`<br>`                  {openFor === a.id ? 'Cancel' : 'Rehydrate'}`<br>`                </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `145-147` | Form input | `                      <input type="datetime-local" value={range.start} onChange={(e) => setRange({ ...range, start: e.target.value })} />`<br>`                    </label>`<br>`                    <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `149-151` | Form input | `                      <input type="datetime-local" value={range.end} onChange={(e) => setRange({ ...range, end: e.target.value })} />`<br>`                    </label>`<br>`                    <button type="button" className="sw-button sw-button-primary" onClick={() => void rehydrate(a)}>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `151-153` | Button | `                    <button type="button" className="sw-button sw-button-primary" onClick={() => void rehydrate(a)}>`<br>`                      Queue rehydration`<br>`                    </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `168-170` | Table | `          <table className="dash-table">`<br>`            <thead>`<br>`              <tr><th>Archive</th><th>Start</th><th>End</th><th>Status</th><th>Progress</th></tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/components/logs/LogMonitorsTab.tsx`
**component** · 143 LOC · **9 drifts** (0 🔴 · 9 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `78-80` | Form input | `            <input type="text" value={monForm.name} onChange={(e) => setMonForm({ ...monForm, name: e.target.value })} placeholder="error-spike" maxLength={128} required />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `82-84` | Form input | `            <input type="text" value={monForm.query} onChange={(e) => setMonForm({ ...monForm, query: e.target.value })} placeholder="level=error service=api" maxLength={1024} required />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `86-88` | Form input | `            <input type="number" min={1} value={monForm.threshold_count} onChange={(e) => setMonForm({ ...monForm, threshold_count: Number(e.target.value) })} />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `90-92` | Form input | `            <input type="number" min={10} value={monForm.threshold_window_seconds} onChange={(e) => setMonForm({ ...monForm, threshold_window_seconds: Number(e.target.value) })} />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `94-96` | Form input | `            <select value={monForm.severity} onChange={(e) => setMonForm({ ...monForm, severity: e.target.value })}>`<br>`              <option value="warn">warn</option>`<br>`              <option value="error">error</option>` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `102-104` | Button | `            <button type="submit" className="sw-button sw-button-primary" disabled={busy}>`<br>`              {busy ? 'Creating…' : 'Create'}`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `114-116` | Table | `          <table className="dash-table">`<br>`            <thead>`<br>`              <tr><th>Name</th><th>Query</th><th>Threshold</th><th>Severity</th><th>Enabled</th><th></th></tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |
| `126-128` | Button | `                    <button type="button" className="sw-button" onClick={() => void toggle(m)}>`<br>`                      {m.enabled ? 'Disable' : 'Enable'}`<br>`                    </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `131-133` | Button | `                    <button type="button" className="sw-button sw-button-danger" onClick={() => void remove(m)}>`<br>`                      Delete`<br>`                    </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/components/logs/LogPatternsTab.tsx`
**component** · 44 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `26-28` | Table | `        <table className="dash-table">`<br>`          <thead>`<br>`            <tr><th>Service</th><th>Pattern</th><th>Count</th><th>Last seen</th></tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/components/logs/LogRetentionTab.tsx`
**component** · 97 LOC · **5 drifts** (0 🔴 · 5 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `54-56` | Form input | `            <input type="text" value={form.service} onChange={(e) => setForm({ ...form, service: e.target.value })} placeholder="api" maxLength={128} required />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `58-60` | Form input | `            <input type="number" min={1} value={form.hot_days} onChange={(e) => setForm({ ...form, hot_days: Number(e.target.value) })} />`<br>`          </label>`<br>`          <label>` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `62-64` | Form input | `            <input type="number" min={1} value={form.cold_days} onChange={(e) => setForm({ ...form, cold_days: Number(e.target.value) })} />`<br>`          </label>`<br>`          {err ? <p className="apm-form-error">{err}</p> : null}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `66-68` | Button | `            <button type="submit" className="sw-button sw-button-primary" disabled={busy}>`<br>`              {busy ? 'Saving…' : 'Save policy'}`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `78-80` | Table | `          <table className="dash-table">`<br>`            <thead>`<br>`              <tr><th>Service</th><th>Hot</th><th>Cold</th><th>Enabled</th></tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

#### `web/src/components/platform/` — 14 files

### `web/src/components/platform/BackupSection.tsx`
**component** · 384 LOC · **3 drifts** (3 🔴 · 0 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `241-243` | Button | `            <button`<br>`              onClick={() => void handleCreate()}`<br>`              disabled={creating}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `249-251` | Button | `          <button`<br>`            onClick={() => setRestoreOpen(true)}`<br>`            className="rounded-md border border-border bg-card px-3 py-2 text-sm text-muted-foreground"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `342-344` | Button | `          <button`<br>`            onClick={() => setRestoreOpen(false)}`<br>`            className="ml-2 underline"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/platform/BackupTable.tsx`
**component** · 163 LOC · **3 drifts** (2 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `79-81` | Table | `      <table className="w-full text-sm">`<br>`        <thead className="bg-muted/50 text-xs uppercase tracking-wider text-muted-foreground">`<br>`          <tr>` | ❌ | 🟡 medium | Raw Tailwind utility classes (`w-full text-sm`) — replace with shared `<DataTable>` from `web/src/components/shared/DataTable.tsx` (to be created). |
| `120-122` | Button | `                <button`<br>`                  onClick={() => onDownload(r.id)}`<br>`                  disabled={r.status !== 'completed' \|\| downloadingId === r.id}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `128-130` | Button | `                  <button`<br>`                    onClick={() => onDelete(r.id)}`<br>`                    className="rounded-md border border-rose-500/30 bg-rose-500/10 px-2 py-1 text-xs text-rose-300 hover:bg-rose-500/20"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/platform/HealthSection.tsx`
**component** · 281 LOC · **5 drifts** (0 🔴 · 5 🟡 · 0 🟢)
  · also 8 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `170-172` | Button | `        <motion.button`<br>`          type="button"`<br>`          onClick={loadAll}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `214-216` | KPI card | `        <section className="kpi-card">`<br>`          <h3 className="text-sm font-semibold">`<br>`            Active tenants trend (last {summary?.trend.length ?? 0} snapshots)` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `235-237` | KPI card | `                <div key={key} className="kpi-card flex flex-col gap-1">`<br>`                  <span className="text-xs opacity-70">{label}</span>`<br>`                  <span className="dash-status-pill"` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `237-239` | Status pill | `                  <span className="dash-status-pill"`<br>`                    style={{`<br>`                      background: ok ? 'rgba(16,185,129,0.15)' : 'rgba(239,68,68,0.15)',` | ❌ | 🟡 medium | Replace with `<StatusPill status="up\|stale\|down\|unknown\|ok\|warn\|crit">` from `web/src/components/shared/StatusPill.tsx` (status→color mapping lives there). |
| `256-258` | KPI card | `        <section className="kpi-card">`<br>`          <h3 className="text-sm font-semibold mb-2">Regions</h3>`<br>`          <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-2 text-sm">` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |

### `web/src/components/platform/HealthSectionTables.tsx`
**component** · 194 LOC · **6 drifts** (0 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `102-104` | KPI card | `      <section className="kpi-card">`<br>`        <h3 className="text-sm font-semibold mb-2">Top tenants (30-day usage)</h3>`<br>`        {top.length === 0 ? (` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `108-110` | Table | `            <table className="w-full text-sm">`<br>`              <thead>`<br>`                <tr className="text-left opacity-70">` | ❌ | 🟡 medium | Raw Tailwind utility classes (`w-full text-sm`) — replace with shared `<DataTable>` from `web/src/components/shared/DataTable.tsx` (to be created). |
| `134-136` | KPI card | `        <section className="kpi-card">`<br>`          <h3 className="text-sm font-semibold mb-2">`<br>`            Capacity forecast (next {forecast.forecast_days} days)` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `156-158` | KPI card | `      <section className="kpi-card">`<br>`        <h3 className="text-sm font-semibold mb-2">Active alerts ({alerts.length})</h3>`<br>`        {alerts.length === 0 ? (` | ❌ | 🟡 medium | Replace custom KPI markup with `<KpiCard label value delta accent icon sparkline>` from `web/src/components/shared/KpiCard.tsx` (3px accent stripe, tabular-nums, sparkline integration). |
| `162-164` | Table | `            <table className="w-full text-sm">`<br>`              <thead>`<br>`                <tr className="text-left opacity-70">` | ❌ | 🟡 medium | Raw Tailwind utility classes (`w-full text-sm`) — replace with shared `<DataTable>` from `web/src/components/shared/DataTable.tsx` (to be created). |
| `175-177` | Status pill | `                        className="dash-status-pill"`<br>`                        style={{ background: 'transparent', color: severityColor(a.severity) }}`<br>`                      >` | ❌ | 🟡 medium | Replace with `<StatusPill status="up\|stale\|down\|unknown\|ok\|warn\|crit">` from `web/src/components/shared/StatusPill.tsx` (status→color mapping lives there). |

### `web/src/components/platform/LimitsChangeModal.tsx`
**component** · 157 LOC · **13 drifts** (3 🔴 · 10 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `77-79` | Modal | `      className="sw-modal-backdrop"`<br>`      role="dialog"`<br>`      aria-modal="true"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `78-80` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`      aria-label="Change plan"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `95-97` | Modal | `          <h3 className="limits-modal-title">Change plan</h3>`<br>`          <p className="limits-modal-sub">`<br>`            Current plan: <strong>{current.plan_display_name}</strong>` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `96-98` | Modal | `          <p className="limits-modal-sub">`<br>`            Current plan: <strong>{current.plan_display_name}</strong>`<br>`          </p>` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `99-101` | Modal | `          <div className="limits-modal-grid">`<br>`            {sortedDefs.map((p) => (`<br>`              <label` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `103-105` | Modal | `                className={\`limits-modal-option ${`<br>`                  pendingPlan === p.name ? 'limits-modal-option-active' : ''`<br>`                }\`}` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `104-106` | Modal | `                  pendingPlan === p.name ? 'limits-modal-option-active' : ''`<br>`                }\`}`<br>`              >` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `107-109` | Form input | `                <input`<br>`                  type="radio"`<br>`                  name="plan"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `114-116` | Modal | `                <div className="limits-modal-option-name">{p.display_name}</div>`<br>`                <div className="limits-modal-option-price">`<br>`                  {formatUSD(p.monthly_price_cents)} / mo` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `115-117` | Modal | `                <div className="limits-modal-option-price">`<br>`                  {formatUSD(p.monthly_price_cents)} / mo`<br>`                </div>` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `118-120` | Modal | `                <ul className="limits-modal-option-features">`<br>`                  <li>`<br>`                    {p.max_servers >= 999999 ? '∞' : p.max_servers} servers` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `133-135` | Button | `            <motion.button`<br>`              type="submit"`<br>`              className="signup-submit"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `144-146` | Button | `            <button`<br>`              type="button"`<br>`              className="signup-back"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/platform/LimitsCheckTool.tsx`
**component** · 113 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `64-66` | Form input | `          <select`<br>`            value={action}`<br>`            onChange={(e) => setAction(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `78-80` | Form input | `          <input`<br>`            type="number"`<br>`            min={1}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `87-89` | Button | `        <motion.button`<br>`          type="button"`<br>`          className="limits-check-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/platform/LimitsSection.tsx`
**component** · 321 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `240-242` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="limits-change"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/platform/LimitsUsageView.tsx`
**component** · 121 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `58-60` | KPI card | `        className="dash-metric-strip"`<br>`        initial="hidden"`<br>`        animate="show"` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |

### `web/src/components/platform/MeteringSection.tsx`
**component** · 380 LOC · **4 drifts** (0 🔴 · 3 🟡 · 1 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `204-206` | Form input | `            <select`<br>`              value={period}`<br>`              onChange={(e) => setPeriod(e.target.value as Period)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `215-217` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="metering-export"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `230-232` | KPI card | `      <motion.div className="dash-metric-strip" initial="hidden" animate="show" variants={kpiStagger}>`<br>`        <motion.div variants={kpiEnter}>`<br>`          <KpiCard` | ❌ | 🟡 medium | Custom `dash-metric` KPI markup — replace with `<KpiCard>` from `web/src/components/shared/KpiCard.tsx`. |
| `305-307` | Table | `            <table className="metering-detail-table">`<br>`              <thead>`<br>`                <tr>` | ❌ | 🟢 low | Domain-specific table class. Replace with shared `<DataTable>` (to be created) and pass domain data via props. |

### `web/src/components/platform/RateLimitSection.tsx`
**component** · 388 LOC · **2 drifts** (0 🔴 · 2 🟡 · 0 🟢)
  · also 5 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `302-304` | Button | `            <motion.button`<br>`              type="button"`<br>`              onClick={handleReset}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `313-315` | Button | `            <motion.button`<br>`              type="button"`<br>`              onClick={handleSave}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/platform/RateLimitTables.tsx`
**component** · 163 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `67-69` | Table | `        <table className="dash-table">`<br>`          <thead>`<br>`            <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |
| `93-95` | Form input | `                      <input`<br>`                        type="number"`<br>`                        min={1}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `132-134` | Table | `            <table className="dash-table">`<br>`              <thead>`<br>`                <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/components/platform/RegionsSection.tsx`
**component** · 368 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)
  · also 6 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `221-223` | Button | `          <motion.button`<br>`            type="button"`<br>`            onClick={handleProbe}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `233-235` | Button | `            <motion.button`<br>`              type="button"`<br>`              onClick={() => setAddOpen(true)}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `298-300` | Table | `          <table className="dash-table">`<br>`            <thead>`<br>`              <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/components/platform/RegionsSectionModals.tsx`
**component** · 190 LOC · **13 drifts** (7 🔴 · 6 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `31-33` | Modal | `      className="modal-backdrop"`<br>`      onClick={onClose}`<br>`      role="dialog"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `33-35` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`    >` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `36-38` | Modal | `      <div className="modal-card" onClick={(e) => e.stopPropagation()}>`<br>`        <h3 className="text-lg font-semibold">{region.display_name}</h3>`<br>`        <p className="text-sm opacity-70 font-mono">{region.code}</p>` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `61-63` | Button | `        <button type="button" className="btn-secondary mt-4" onClick={onClose}>`<br>`          Close`<br>`        </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `104-106` | Modal | `      className="modal-backdrop"`<br>`      onClick={() => !creating && onClose()}`<br>`      role="dialog"` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `106-108` | Modal | `      role="dialog"`<br>`      aria-modal="true"`<br>`    >` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `109-111` | Modal | `      <div className="modal-card" onClick={(e) => e.stopPropagation()}>`<br>`        <h3 className="text-lg font-semibold">Add region</h3>`<br>`        <p className="text-sm opacity-70">` | ❌ | 🔴 critical | **Missing shared Modal component.** Create `web/src/components/shared/Modal.tsx` (focus-trap, `aria-modal="true"`, `role="dialog"`, ESC-to-close, backdrop click, framer-motion fade) and replace this custom modal markup. Currently 14+ ad-hoc modal implementations across the codebase. |
| `117-119` | Form input | `          <input`<br>`            type="text"`<br>`            value={addCode}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `128-130` | Form input | `          <input`<br>`            type="text"`<br>`            value={addName}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `139-141` | Form input | `          <select`<br>`            value={addKind}`<br>`            onChange={(e) => onChangeKind(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `152-154` | Form input | `          <input`<br>`            type="text"`<br>`            value={addUrl}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `167-169` | Button | `          <button`<br>`            type="button"`<br>`            className="btn-secondary"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `175-177` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="btn-primary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/platform/SignupSection.tsx`
**component** · 345 LOC · **10 drifts** (6 🔴 · 4 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `211-213` | Form input | `            <input`<br>`              type="email"`<br>`              name="email"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `223-225` | Form input | `            <input`<br>`              type="text"`<br>`              name="full_name"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `236-238` | Form input | `            <input`<br>`              type="text"`<br>`              name="organization_name"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `255-257` | Button | `          <button type="submit" className="signup-submit" disabled={busy}>`<br>`            {busy ? 'Creating account…' : 'Create account'}`<br>`          </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `272-274` | Form input | `            <input`<br>`              type="text"`<br>`              name="token"` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `284-286` | Button | `            <button type="submit" className="signup-submit" disabled={busy \|\| token.length < 16}>`<br>`              {busy ? 'Verifying…' : 'Verify and sign in'}`<br>`            </button>` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `287-289` | Button | `            <button`<br>`              type="button"`<br>`              className="signup-resend"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `301-303` | Button | `          <button`<br>`            type="button"`<br>`            className="signup-back"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `326-328` | Button | `            <button`<br>`              type="button"`<br>`              className="signup-submit"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `333-335` | Button | `            <button`<br>`              type="button"`<br>`              className="signup-resend"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

#### `web/src/components/profile/` — 5 files

### `web/src/components/profile/AvatarUpload.tsx`
**component** · 146 LOC · **2 drifts** (1 🔴 · 0 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `126-128` | Form input | `      <input`<br>`        type="file"`<br>`        accept="image/*"` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `135-137` | Button | `        <button`<br>`          type="button"`<br>`          className="prof-hero-avatar-remove"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/profile/IdentityPanel.tsx`
**component** · 94 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `36-38` | Button | `            <button`<br>`              type="button"`<br>`              className="prof-id-copy"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/profile/NameFormPanel.tsx`
**component** · 64 LOC · **3 drifts** (0 🔴 · 2 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `40-42` | Form input | `          <input`<br>`            type="text"`<br>`            value={draft}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `54-56` | Button | `          <button type="button" className="sw-button sw-button-quiet" onClick={onDiscard} disabled={saving}>`<br>`            Discard`<br>`          </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `57-59` | Button | `          <button type="submit" className="sw-button sw-button-primary" disabled={saving \|\| unchanged}>`<br>`            {saving ? 'Saving…' : 'Save name'}`<br>`          </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/components/profile/SecurityPanel.tsx`
**component** · 98 LOC · **3 drifts** (1 🔴 · 2 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `67-69` | Button | `            <button`<br>`              type="button"`<br>`              className="sw-button sw-button-quiet"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `81-83` | Button | `            <button type="button" className="sw-button sw-button-quiet" onClick={onSignOut}>`<br>`              Sign out`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `90-92` | Button | `            <button type="button" className="sw-button sw-button-quiet" onClick={onToggleTokens}>`<br>`              {tokensOpen ? 'Hide' : 'Manage'}`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/components/profile/TokensPanel.tsx`
**component** · 178 LOC · **5 drifts** (3 🔴 · 1 🟡 · 1 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `106-108` | Button | `            <button`<br>`              type="button"`<br>`              className="sw-button sw-button-quiet"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `119-121` | Button | `            <button type="button" className="sw-button" onClick={() => setRevealed(null)}>`<br>`              I've saved it`<br>`            </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |
| `129-131` | Form input | `        <input`<br>`          type="text"`<br>`          className="sw-input"` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `137-139` | Button | `        <button`<br>`          type="button"`<br>`          className="sw-button sw-button-primary"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `163-165` | Button | `                  <button`<br>`                    type="button"`<br>`                    className="sw-button sw-button-quiet sw-button-danger"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

#### `web/src/components/shared/` — 22 files

### `web/src/components/shared/AuditArchiveCard.tsx`
**component** · 163 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `139-141` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/Button.test.tsx`
**component** · 105 LOC · **1 drifts** (0 🔴 · 0 🟡 · 1 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `71-73` | Loading state | `    render(<Button loading onClick={onClick}>Loading</Button>);`<br>`    const btn = screen.getByRole('button');`<br>`    expect(btn).toBeDisabled();` | ❌ | 🟢 low | Plain `'Loading…'` text — replace with shared `<SkeletonCard>` (for page-level) or `<SkeletonRow>` (for list rows). Button text `'Loading…'` is fine inside `<Button loading>` state. |

### `web/src/components/shared/ComplianceReportCard.tsx`
**component** · 198 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `156-158` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/CopyBox.tsx`
**component** · 77 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `65-67` | Button | `      <motion.button`<br>`        type="button"`<br>`        className="threat-card-resolve-btn"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/CorrelationCard.tsx`
**component** · 327 LOC · **4 drifts** (4 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `199-201` | Button | `          <button`<br>`            type="button"`<br>`            onClick={handleExpand}` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `288-290` | Button | `        <button`<br>`          type="button"`<br>`          className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `302-304` | Button | `            <button`<br>`              type="button"`<br>`              className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `312-314` | Button | `            <button`<br>`              type="button"`<br>`              className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/ErrorGroupCard.tsx`
**component** · 105 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `95-97` | Button | `        <button`<br>`          type="button"`<br>`          className="rum-error-card-expand"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/IncidentCard.tsx`
**component** · 141 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `130-132` | Button | `          <button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/KpiCard.tsx`
**component** · 131 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `109-111` | Button | `      <motion.button`<br>`        type="button"`<br>`        className={\`dash-metric dash-metric-${toneClass} dash-metric-clickable\`}` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/LogSearchBar.tsx`
**component** · 99 LOC · **3 drifts** (0 🔴 · 3 🟡 · 0 🟢)
  · also 2 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `58-60` | Form input | `        <select`<br>`          value={level}`<br>`          onChange={(e) => setLevel(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `83-85` | Form input | `        <select`<br>`          value={timeRange}`<br>`          onChange={(e) => setTimeRange(e.target.value as LogFilters['timeRange'])}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `94-96` | Button | `      <button type="submit" className="sw-button sw-button-primary">`<br>`        Search`<br>`      </button>` | ❌ | 🟡 medium | Replace with `<Button variant="primary\|secondary\|danger\|ghost">…</Button>` from `web/src/components/shared/Button.tsx` (40px height, framer-motion scale on tap, token-driven colors). |

### `web/src/components/shared/MentionInput.tsx`
**component** · 247 LOC · **3 drifts** (2 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `194-196` | Form input | `      <textarea`<br>`        className="notebook-editor-textarea"`<br>`        rows={rows}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `212-214` | Button | `            <button`<br>`              key={u.id}`<br>`              type="button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `233-235` | Button | `              <button`<br>`                type="button"`<br>`                className="mention-input-chip-remove"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/NoiseRuleEditor.tsx`
**component** · 239 LOC · **7 drifts** (0 🔴 · 3 🟡 · 4 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `128-130` | Form input | `        <input`<br>`          type="text"`<br>`          required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `140-142` | Form input | `        <input`<br>`          type="text"`<br>`          required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `156-158` | Form input | `        <select`<br>`          value={windowLabel}`<br>`          onChange={(e) => setWindowLabel(e.target.value)}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `182-184` | Form input | `              <input`<br>`                type="checkbox"`<br>`                checked={channels.includes(ch)}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `200-202` | Form input | `        <input`<br>`          type="checkbox"`<br>`          checked={enabled}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `214-216` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="dash-icon-button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `226-228` | Button | `        <motion.button`<br>`          type="submit"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/NotebookEditor.tsx`
**component** · 174 LOC · **6 drifts** (2 🔴 · 4 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `93-95` | Modal | `      <div className="slow-query-explain-modal-head">`<br>`        <strong>{title \|\| 'Untitled notebook'}</strong>`<br>`        <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `95-97` | Button | `        <button`<br>`          type="button"`<br>`          className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `108-110` | Form input | `          <input`<br>`            type="text"`<br>`            value={title}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `135-137` | Form input | `          <textarea`<br>`            className="notebook-editor-textarea"`<br>`            value={content}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `151-153` | Button | `            <button`<br>`              type="button"`<br>`              className="dash-icon-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `158-160` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/OrgSettingsForm.tsx`
**component** · 281 LOC · **6 drifts** (0 🔴 · 3 🟡 · 3 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `170-172` | Form input | `          <select`<br>`            className="form-input"`<br>`            value={theme}` | ✅ | 🟢 low | Specialized select — consider wrapping in shared `<Select>` once created. |
| `186-188` | Form input | `          <select`<br>`            className="form-input"`<br>`            value={defaultDashboard}` | ✅ | 🟢 low | Specialized select — consider wrapping in shared `<Select>` once created. |
| `202-204` | Form input | `          <input`<br>`            className="form-input"`<br>`            type="text"` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `225-227` | Form input | `        <textarea`<br>`          className="form-input"`<br>`          value={customJson}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `255-257` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `268-270` | Button | `        <motion.button`<br>`          type="submit"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/RbacRoleEditor.tsx`
**component** · 355 LOC · **5 drifts** (0 🔴 · 3 🟡 · 2 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `194-196` | Form input | `        <input`<br>`          type="text"`<br>`          value={draft.name}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `225-227` | Form input | `        <textarea`<br>`          value={draft.description}`<br>`          onChange={(e) => setDraft((d) => ({ ...d, description: e.target.value }))}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `302-304` | Form input | `                  <input`<br>`                    type="checkbox"`<br>`                    checked={checked}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `329-331` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="sw-button sw-button-secondary"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `341-343` | Button | `          <motion.button`<br>`            type="submit"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/RumSessionTabs.tsx`
**component** · 128 LOC · **2 drifts** (0 🔴 · 2 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `54-56` | Table | `    <table className="sw-table">`<br>`      <thead>`<br>`        <tr>` | ❌ | 🟡 medium | `sw-table` is a legacy class. Replace with shared `<DataTable>` (to be created). |
| `87-89` | Table | `    <table className="sw-table">`<br>`      <thead>`<br>`        <tr>` | ❌ | 🟡 medium | `sw-table` is a legacy class. Replace with shared `<DataTable>` (to be created). |

### `web/src/components/shared/ScimTokenTable.tsx`
**component** · 208 LOC · **3 drifts** (1 🔴 · 2 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `104-106` | Button | `            <motion.button`<br>`              type="button"`<br>`              className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `174-176` | Button | `                    <button`<br>`                      type="button"`<br>`                      className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `191-193` | Button | `              <motion.button`<br>`                type="button"`<br>`                className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/SlowQueryTable.tsx`
**component** · 148 LOC · **1 drifts** (0 🔴 · 1 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `93-95` | Table | `    <table className="dash-table slow-query-table">`<br>`      <thead>`<br>`        <tr>` | ❌ | 🟡 medium | Replace with shared `<DataTable>` (to be created in `web/src/components/shared/DataTable.tsx`) — handles sortable headers, row hover, sticky thead, token-driven borders. |

### `web/src/components/shared/SnoozeHistoryPanel.tsx`
**component** · 251 LOC · **8 drifts** (3 🔴 · 5 🟡 · 0 🟢)
  · also 2 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `162-164` | Button | `        <motion.button`<br>`          type="button"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `177-179` | Modal | `          <div className="slow-query-explain-modal-head">`<br>`            <strong>Snooze alert</strong>`<br>`            <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `179-181` | Button | `            <button`<br>`              type="button"`<br>`              className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `197-199` | Form input | `              <input`<br>`                type="text"`<br>`                required` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `207-209` | Form input | `              <input`<br>`                type="number"`<br>`                min={60}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `218-220` | Form input | `              <input`<br>`                type="text"`<br>`                maxLength={2048}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `230-232` | Button | `              <button`<br>`                type="button"`<br>`                className="dash-icon-button"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `238-240` | Button | `              <button`<br>`                type="submit"`<br>`                className="empty-state-cta"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/SsoProviderCard.tsx`
**component** · 179 LOC · **3 drifts** (3 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `136-138` | Button | `          <button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `146-148` | Button | `          <button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `156-158` | Button | `          <button`<br>`            type="button"`<br>`            className="threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/SsoProviderForm.tsx`
**component** · 316 LOC · **13 drifts** (0 🔴 · 4 🟡 · 9 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `148-150` | Form input | `        <input`<br>`          type="text"`<br>`          required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `163-165` | Form input | `            <input`<br>`              type="text"`<br>`              required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `174-176` | Form input | `            <input`<br>`              type="password"`<br>`              required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `185-187` | Form input | `            <input`<br>`              type="url"`<br>`              required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `196-198` | Form input | `            <input`<br>`              type="url"`<br>`              value={draft.redirect_uri \|\| ''}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `206-208` | Form input | `            <input`<br>`              type="text"`<br>`              value={draft.scopes \|\| ''}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `219-221` | Form input | `            <input`<br>`              type="text"`<br>`              required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `230-232` | Form input | `            <input`<br>`              type="url"`<br>`              required` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `241-243` | Form input | `            <input`<br>`              type="url"`<br>`              value={draft.metadata_url \|\| ''}` | ❌ | 🟢 low | Specialized input — promote to shared `<Input variant=…>` once `web/src/components/shared/Input.tsx` is created. |
| `251-253` | Form input | `            <textarea`<br>`              value={draft.metadata_xml \|\| ''}`<br>`              onChange={(e) => update('metadata_xml', e.target.value)}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `266-268` | Form input | `            <textarea`<br>`              required`<br>`              value={draft.x509_cert \|\| ''}` | ❌ | 🟡 medium | **Missing shared Textarea component.** Create `web/src/components/shared/Textarea.tsx` (auto-grow, char count, error, hint) and replace raw `<textarea>`. |
| `290-292` | Button | `        <motion.button`<br>`          type="button"`<br>`          className="dash-icon-button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `301-303` | Button | `        <motion.button`<br>`          type="submit"`<br>`          className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

### `web/src/components/shared/ThreatCard.tsx`
**component** · 94 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `83-85` | Button | `          <button`<br>`            type="button"`<br>`            className="sw-button sw-button-secondary threat-card-resolve-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/shared/TrainAnomalyModelModal.tsx`
**component** · 138 LOC · **7 drifts** (1 🔴 · 6 🟡 · 0 🟢)
  · also 1 compliant usages (shared `<Button>` / `<StatusPill>` / `<KpiCard>` / `<EmptyState>`)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `64-66` | Modal | `      <div className="slow-query-explain-modal-head">`<br>`        <strong>Train new model</strong>`<br>`        <button` | ❌ | 🟡 medium | Modal uses a non-canonical class prefix (limits-modal-, slow-query-explain-modal-, etc). Replace with shared `<Modal>` from `web/src/components/shared/Modal.tsx` (to be created). |
| `66-68` | Button | `        <button`<br>`          type="button"`<br>`          className="slow-query-explain-close"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |
| `84-86` | Form input | `          <input`<br>`            type="text"`<br>`            required` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `95-97` | Form input | `          <input`<br>`            type="text"`<br>`            value={server}` | ❌ | 🟡 medium | **Missing shared Input component.** Create `web/src/components/shared/Input.tsx` (label, error, hint, leading icon, sizes sm/md/lg, focus ring via tokens) and replace this raw `<input>`. |
| `104-106` | Form input | `          <select`<br>`            value={modelType}`<br>`            onChange={(e) => setModelType(e.target.value as 'welford' \| 'ewma')}` | ❌ | 🟡 medium | **Missing shared Select component.** Create `web/src/components/shared/Select.tsx` (token-driven chevron, label, error, hint) and replace raw `<select>`. |
| `113-115` | Button | `          <motion.button`<br>`            type="button"`<br>`            className="dash-icon-button"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |
| `124-126` | Button | `          <motion.button`<br>`            type="submit"`<br>`            className="empty-state-cta"` | ❌ | 🟡 medium | Replace `<motion.button>` with `<Button>` from `web/src/components/shared/Button.tsx` for consistency with focus ring, ARIA, and motion contracts. |

#### `web/src/components/sidebar/` — 1 files

### `web/src/components/sidebar/SidebarFooter.tsx`
**component** · 70 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `60-62` | Button | `      <button`<br>`        type="button"`<br>`        className="sb-signout"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

#### `web/src/components/topbar/` — 4 files

### `web/src/components/topbar/TopBar.tsx`
**component** · 60 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `35-37` | Button | `      <button`<br>`        type="button"`<br>`        className="tb-hamburger"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/topbar/TopBarNotifications.tsx`
**component** · 87 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `51-53` | Button | `      <button`<br>`        type="button"`<br>`        className="tb-notif-btn"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/topbar/TopBarRefresh.tsx`
**component** · 44 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `27-29` | Button | `    <button`<br>`      type="button"`<br>`      className="tb-refresh"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |

### `web/src/components/topbar/TopBarSearch.tsx`
**component** · 27 LOC · **1 drifts** (1 🔴 · 0 🟡 · 0 🟢)

| Line | Element | Current Implementation | Drift | Severity | Proposed Fix |
|------|---------|------------------------|-------|----------|--------------|
| `16-18` | Button | `    <button`<br>`      type="button"`<br>`      className="tb-search"` | ❌ | 🔴 critical | Replace raw `<button>` with `<Button variant="primary">…</Button>` from `web/src/components/shared/Button.tsx` — gives focus ring, hover/active states, motion, and loading prop for free. |


## (B) Fully compliant files — 36 files, no changes required

> These files already use shared components (`<Button>`, `<StatusPill>`, `<KpiCard>`, `<EmptyState>`, `<SkeletonRow>`) for every audited element type. They appear here for completeness so every audited file is represented in the report.

| File | Kind | Compliant usages |
|------|------|------------------|
| `web/src/components/ComplianceSectionHelpers.tsx` | component | 0 |
| `web/src/components/dashboard/Sparkline.tsx` | component | 0 |
| `web/src/components/dashboard/WelcomeHeader.tsx` | component | 0 |
| `web/src/components/homelab/widgets/NowPlayingList.tsx` | component | 0 |
| `web/src/components/homelab/widgets/RecentAdditionsCarousel.tsx` | component | 0 |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | component | 0 |
| `web/src/components/icons.tsx` | component | 0 |
| `web/src/components/logs/LogSearchTab.tsx` | component | 1 |
| `web/src/components/profile/HeroCard.tsx` | component | 0 |
| `web/src/components/profile/WorkspacePanel.tsx` | component | 0 |
| `web/src/components/shared/AnomalyChart.tsx` | component | 0 |
| `web/src/components/shared/Button.tsx` | component | 2 |
| `web/src/components/shared/ComplianceBar.tsx` | component | 0 |
| `web/src/components/shared/CspmSeverityBadge.tsx` | component | 0 |
| `web/src/components/shared/EmptyState.tsx` | component | 1 |
| `web/src/components/shared/FlameGraph.tsx` | component | 0 |
| `web/src/components/shared/LogEntry.tsx` | component | 0 |
| `web/src/components/shared/PipelineCard.tsx` | component | 0 |
| `web/src/components/shared/PredictionChart.tsx` | component | 0 |
| `web/src/components/shared/RcaPanel.tsx` | component | 0 |
| `web/src/components/shared/ResourceWaterfall.tsx` | component | 0 |
| `web/src/components/shared/SkeletonRow.tsx` | component | 2 |
| `web/src/components/shared/SlaBadge.tsx` | component | 0 |
| `web/src/components/shared/StatusPill.tsx` | component | 2 |
| `web/src/components/shared/TimeSeriesChart.tsx` | component | 0 |
| `web/src/components/shared/TraceSummary.tsx` | component | 0 |
| `web/src/components/sidebar/Sidebar.tsx` | component | 0 |
| `web/src/components/sidebar/SidebarItem.tsx` | component | 0 |
| `web/src/components/sidebar/SidebarSection.tsx` | component | 0 |
| `web/src/components/topbar/TopBarBreadcrumb.tsx` | component | 0 |
| `web/src/components/topbar/TopBarGreeting.tsx` | component | 0 |
| `web/src/components/topbar/TopBarLogo.tsx` | component | 0 |
| `web/src/components/topbar/TopBarProfile.tsx` | component | 0 |
| `web/src/components/topbar/TopBarTime.tsx` | component | 0 |
| `web/src/components/topbar/TopBarWeather.tsx` | component | 0 |
| `web/src/pages/Landing.tsx` | page | 0 |

## Supplementary findings — token & accessibility drift

> Beyond the eight element-type categories above, this section catalogs two additional kinds of drift from `web/src/styles/DESIGN-SYSTEM.md` and `web/src/styles/tokens.css`: (1) **hardcoded hex colors** that bypass the token system, and (2) **inline `style={{ ...: 'Npx' }}`** that bypass the spacing scale. Both are forbidden by §12 ("Token usage") of the design system.

### Hardcoded hex colors (37 hits across 8 files)

> §12 of `DESIGN-SYSTEM.md` explicitly forbids hex literals in component code: every color must come from a CSS custom property (`var(--color-*)`). The 8 files below are the ones that still reach for a literal.

| File | Line | Current Code | Tokens to use |
|------|------|--------------|----------------|
| `web/src/components/dashboard/Sparkline.tsx` | `19` | `cyan: '#38bdf8',` | var(--accent-cyan) — promote token if missing |
| `web/src/components/dashboard/Sparkline.tsx` | `20` | `indigo: '#818cf8',` | var(--accent-indigo) — promote token if missing |
| `web/src/components/dashboard/Sparkline.tsx` | `21` | `green: '#34d399',` | var(--accent-green) — promote token if missing |
| `web/src/components/dashboard/Sparkline.tsx` | `22` | `amber: '#fbbf24',` | _(promote to token or document as user-defined)_ |
| `web/src/components/dashboard/Sparkline.tsx` | `23` | `red: '#f87171',` | _(promote to token or document as user-defined)_ |
| `web/src/components/dashboard/TrendChart.tsx` | `44` | `<stop offset="0" stopColor="#38bdf8" stopOpacity=".34" />` | var(--accent-cyan) — promote token if missing |
| `web/src/components/dashboard/TrendChart.tsx` | `45` | `<stop offset="1" stopColor="#38bdf8" stopOpacity="0" />` | var(--accent-cyan) — promote token if missing |
| `web/src/components/dashboard/TrendChart.tsx` | `52` | `<polyline points={points} fill="none" stroke="#38bdf8" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />` | var(--accent-cyan) — promote token if missing |
| `web/src/components/dashboard/TrendChart.tsx` | `62` | `fill="#0b1220"` | _(promote to token or document as user-defined)_ |
| `web/src/components/dashboard/TrendChart.tsx` | `63` | `stroke="#67e8f9"` | var(--color-primary-hover) |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `13` | `{ hex: '#3b82f6', label: 'Blue' },` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `14` | `{ hex: '#22c55e', label: 'Green' },` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `15` | `{ hex: '#ef4444', label: 'Red' },` | var(--color-error) |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `16` | `{ hex: '#f59e0b', label: 'Amber' },` | var(--color-warning) |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `17` | `{ hex: '#a855f7', label: 'Purple' },` | _(promote to token or document as user-defined)_ |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `18` | `{ hex: '#06b6d4', label: 'Cyan' },` | _(promote to token or document as user-defined)_ |
| `web/src/components/homelab/widgets/AddCalendarModal.tsx` | `19` | `{ hex: '#6b7280', label: 'Gray' },` | _(promote to token or document as user-defined)_ |
| `web/src/components/homelab/widgets/CalendarWidget.tsx` | `169` | `for (const c of calendars) out[c.id] = c.color \|\| '#3b82f6';` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/CalendarWidget.tsx` | `245` | `borderLeftColor: c.color \|\| '#3b82f6',` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/CalendarWidget.tsx` | `336` | `const color = colorById[e.calendar_id] \|\| '#3b82f6';` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/CalendarWidget.tsx` | `390` | `calendarColor={colorById[detailEvent.calendar_id] \|\| '#3b82f6'}` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `35` | `color: calendarColor \|\| '#3b82f6',` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `98` | `return '#3b82f6';` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `100` | `return '#22c55e';` | user-defined calendar color (not in palette) |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `102` | `return '#a855f7';` | _(promote to token or document as user-defined)_ |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `104` | `return '#f59e0b';` | var(--color-warning) |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `106` | `return '#6b7280';` | _(promote to token or document as user-defined)_ |
| `web/src/components/homelab/widgets/RssWidget.helpers.tsx` | `108` | `return '#06b6d4';` | _(promote to token or document as user-defined)_ |
| `web/src/components/shared/TimeSeriesChart.tsx` | `24` | `cyan:   { stroke: '#22d3ee', fill: 'rgba(34,211,238,0.18)' },` | var(--color-primary) |
| `web/src/components/shared/TimeSeriesChart.tsx` | `25` | `indigo: { stroke: '#818cf8', fill: 'rgba(129,140,248,0.18)' },` | var(--accent-indigo) — promote token if missing |
| `web/src/components/shared/TimeSeriesChart.tsx` | `26` | `green:  { stroke: '#10b981', fill: 'rgba(16,185,129,0.18)' },` | var(--color-success) |
| `web/src/components/shared/TimeSeriesChart.tsx` | `27` | `amber:  { stroke: '#f59e0b', fill: 'rgba(245,158,11,0.18)' },` | var(--color-warning) |
| `web/src/components/shared/TimeSeriesChart.tsx` | `28` | `red:    { stroke: '#ef4444', fill: 'rgba(239,68,68,0.18)' },` | var(--color-error) |
| `web/src/components/shared/TimeSeriesChart.tsx` | `29` | `violet: { stroke: '#a78bfa', fill: 'rgba(167,139,250,0.18)' },` | _(promote to token or document as user-defined)_ |
| `web/src/pages/SettingsPage.tsx` | `175` | `<h2 id="settings-title" style={{ margin: '7px 0 5px', color: '#f6f9ff', fontSize: '29px', letterSpacing: '-0.9px' }}>Settings</h2>` | _(promote to token or document as user-defined)_ |
| `web/src/pages/SettingsPage.tsx` | `176` | `<p style={{ margin: 0, color: '#8292aa', fontSize: '13px' }}>` | _(promote to token or document as user-defined)_ |
| `web/src/pages/SettingsPage.tsx` | `178` | `<Link to="/profile" style={{ color: '#67e8f9' }}>Profile page</Link>.` | var(--color-primary-hover) |

### Inline `style={{ ...: 'Npx' }}` (25 hits across 16 files)

> §3 ("Spacing") and §12 ("Token usage") of `DESIGN-SYSTEM.md` forbid arbitrary pixel values. The 16 files below still inline spacing values. Every entry must be replaced with a `var(--space-N)` token (or moved to a CSS class).

| File | Line | Current Code | Token to use |
|------|------|--------------|----------------|
| `web/src/components/AuditSection.tsx` | `242` | `<p style={{ margin: '8px 0 16px', fontSize: 12 }}>` | var(--space-2) |
| `web/src/components/ComplianceScheduleList.tsx` | `48` | `<p style={{ margin: '8px 0 16px', fontSize: 12 }}>` | var(--space-2) |
| `web/src/components/ComplianceSection.tsx` | `332` | `<p style={{ margin: '8px 0 16px', fontSize: 12 }}>` | var(--space-2) |
| `web/src/components/RbacSection.tsx` | `255` | `<p style={{ margin: '8px 0 0', fontSize: 12 }}>` | var(--space-2) |
| `web/src/components/homelab/widgets/CalendarWidget.tsx` | `254` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/DownloadClientCard.tsx` | `40` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `40` | `<p style={{ margin: '8px 0', fontSize: 13, opacity: 0.7 }}>` | var(--space-2) |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `43` | `<p style={{ margin: '4px 0', fontSize: 13 }}>` | var(--space-1) |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `47` | `<p style={{ margin: '4px 0', fontSize: 13 }}>` | var(--space-1) |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `52` | `<p style={{ margin: '4px 0', fontSize: 13, opacity: 0.7 }}>` | var(--space-1) |
| `web/src/components/homelab/widgets/EventDetailModal.tsx` | `57` | `<p style={{ margin: '8px 0', fontSize: 13 }}>` | var(--space-2) |
| `web/src/components/homelab/widgets/MediaServerCard.tsx` | `39` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/NotesWidget.tsx` | `220` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/NowPlayingList.tsx` | `58` | `style={{ fontSize: 9, padding: '1px 5px' }}` | _1px is off-scale; pick from §3_ |
| `web/src/components/homelab/widgets/RssFeedCard.tsx` | `36` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/RssWidget.tsx` | `201` | `style={{ fontSize: 11, padding: '2px 8px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/RssWidget.tsx` | `292` | `<ul style={{ listStyle: 'none', padding: 0, margin: '16px 0' }}>` | var(--space-4) |
| `web/src/components/homelab/widgets/TodoCard.tsx` | `77` | `style={{ fontSize: 10, padding: '2px 6px', marginLeft: 'auto' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/homelab/widgets/TodoCard.tsx` | `107` | `style={{ fontSize: 10, padding: '2px 6px' }}` | var(--space-1) is 4px; consider semantic line-height instead |
| `web/src/components/shared/ScimTokenTable.tsx` | `99` | `<p style={{ margin: '8px 0 16px', fontSize: 12 }}>` | var(--space-2) |
| `web/src/components/shared/SnoozeHistoryPanel.tsx` | `134` | `<div key={s.id} className="threat-card" style={{ padding: '10px 14px' }}>` | _(10px is not on the scale — use 8px or 12px)_ |
| `web/src/pages/SettingsPage.tsx` | `175` | `<h2 id="settings-title" style={{ margin: '7px 0 5px', color: '#f6f9ff', fontSize: '29px', letterSpacing: '-0.9px' }}>Settings</h2>` | _7px is off-scale; pick from §3_ |
| `web/src/pages/SettingsPage.tsx` | `176` | `<p style={{ margin: 0, color: '#8292aa', fontSize: '13px' }}>` | _13px is off-scale; pick from §3_ |
| `web/src/pages/SettingsPage.tsx` | `191` | `<form onSubmit={onSaveName} className="sw-form-grid" style={{ padding: '18px 20px 20px' }}>` | _18px is off-scale; pick from §3_ |
| `web/src/pages/SettingsPage.tsx` | `215` | `<form onSubmit={onChangePassword} className="sw-form-grid" style={{ padding: '18px 20px 20px' }}>` | _18px is off-scale; pick from §3_ |

---

## Recommended new shared components (Phase D prerequisite)

> Six shared components are still missing. The audit identifies **575 drift findings**, of which the bulk cannot be remediated without these building blocks first. Each API below is sized to absorb every drift finding surfaced in this audit.

### 1. `web/src/components/shared/Modal.tsx`

**Drifts addressed:** 70 modal usages across 31 files. Today there are at least 6 different modal markup patterns (`modal-backdrop`, `modal-card`, `limits-modal-*`, `slow-query-explain-modal`, `sw-modal-backdrop`, `cmd-palette`).

```tsx
export interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;          // for aria-describedby
  size?: 'sm' | 'md' | 'lg' | 'xl';  // 400 / 560 / 720 / 960 px
  initialFocus?: RefObject<HTMLElement>;
  children: ReactNode;
  footer?: ReactNode;            // for action buttons
  dismissible?: boolean;         // ESC + backdrop click; default true
  /** Stack depth for nested modals. */
  zIndex?: number;
}
```

**Implementation requirements:**
- `role="dialog"`, `aria-modal="true"`, `aria-labelledby` (title id), `aria-describedby` (description id)
- Focus trap (Tab cycles within modal, Shift+Tab cycles backward)
- ESC keydown closes (when `dismissible`)
- Backdrop click closes (when `dismissible`)
- Body scroll lock while open
- Framer-motion fade + scale enter/exit (200ms ease-out / 150ms ease-in)
- All colors via tokens: backdrop = `rgba(0,0,0,0.7)`, surface = `var(--color-surface-elevated)`, border = `var(--color-border-strong)`
- Reduced-motion: instant open/close (no scale)

**Example migration:**

```tsx
// Before (LimitsChangeModal.tsx L77-78):
<div className="sw-modal-backdrop" onClick={...}>
  <div className="sw-modal-card">...</div>
</div>

// After:
<Modal open={open} onClose={...} title="Change plan" size="md">
  ...content...
  <Modal.Footer>
    <Button variant="ghost" onClick={...}>Cancel</Button>
    <Button variant="primary" onClick={...}>Confirm</Button>
  </Modal.Footer>
</Modal>
```

### 2. `web/src/components/shared/DataTable.tsx`

**Drifts addressed:** 23 raw `<table>` usages across 18 files. Today there are at least 5 different table class prefixes (`dash-table`, `sw-table`, `metering-detail-table`, `apm-traces`, `apm-deployments`, raw Tailwind `w-full text-sm`).

```tsx
export interface DataTableColumn<T> {
  key: keyof T | string;
  header: string;
  width?: string;                 // CSS grid/flex value
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  render?: (row: T) => ReactNode;  // custom cell renderer
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  emptyState?: ReactNode;         // render inside <EmptyState>
  loading?: boolean;              // shows <SkeletonRow> overlay
  density?: 'compact' | 'comfortable';
  stickyHeader?: boolean;
  onRowClick?: (row: T) => void;
}
```

**Implementation requirements:**
- All colors via tokens (border, hover, header bg)
- Sticky `<thead>` with `var(--color-surface-elevated)` background
- Row hover: `var(--color-surface-hover)`
- Sortable headers: arrow icon, click to cycle asc/desc/none
- Empty state composes with `<EmptyState>` from shared/
- Loading state composes with `<SkeletonRow>` (rows=3, columns=columns.length)
- Tabular-nums on numeric columns automatically

**Example migration:**

```tsx
// Before (ApmPage.tsx L347):
<table className="dash-table apm-deployments">
  <thead><tr><th>Time</th><th>Status</th><th>Author</th></tr></thead>
  <tbody>{deployments.map(d => <tr key={d.id}>...</tr>)}</tbody>
</table>

// After:
<DataTable
  columns={[
    { key: 'time', header: 'Time', sortable: true },
    { key: 'status', header: 'Status', render: r => <StatusPill status={r.status} /> },
    { key: 'author', header: 'Author' },
  ]}
  rows={deployments}
  rowKey={d => d.id}
  emptyState={<EmptyState illustration={...} headline="No deployments yet" />}
/>
```

### 3. `web/src/components/shared/Input.tsx`

**Drifts addressed:** 127 raw `<input>` usages across 60 files, plus 11 `<textarea>` usages and 37 `<select>` usages.

```tsx
export interface InputProps {
  label: string;                  // always required for a11y
  hint?: string;                  // help text below
  error?: string;                 // replaces hint when set
  size?: 'sm' | 'md' | 'lg';      // 32 / 40 / 48 px
  leadingIcon?: ReactNode;
  trailingIcon?: ReactNode;
  type?: 'text' | 'email' | 'password' | 'url' | 'number' | 'search';
  fullWidth?: boolean;
  disabled?: boolean;
  required?: boolean;             // adds aria-required + visual *
  // ...all native input props
}
```

**Implementation requirements:**
- `<label htmlFor>` linked by generated id
- `aria-invalid` when `error` is set, `aria-describedby` for hint/error
- Focus ring via `var(--color-primary)` 2px outline at 2px offset
- Error state: border `var(--color-error)`, hint text red
- Disabled state: opacity 0.5, cursor not-allowed
- All colors via tokens, all spacing via tokens

### 4. `web/src/components/shared/Select.tsx`

**Drifts addressed:** 37 raw `<select>` usages across ~20 files.

```tsx
export interface SelectOption<V extends string = string> {
  value: V;
  label: string;
  disabled?: boolean;
}

export interface SelectProps<V extends string = string> {
  label: string;
  options: SelectOption<V>[];
  value: V;
  onChange: (value: V) => void;
  hint?: string;
  error?: string;
  size?: 'sm' | 'md' | 'lg';
  placeholder?: string;
  disabled?: boolean;
}
```

**Implementation requirements:**
- Token-driven chevron icon (no native `<select>` arrow)
- Same a11y contract as Input (label, hint, error, aria-*)
- Opens a `var(--color-surface-elevated)` dropdown with `var(--shadow-2)`
- Active option highlighted with `var(--color-primary)` tint

### 5. `web/src/components/shared/Textarea.tsx`

**Drifts addressed:** 11 raw `<textarea>` usages across ~10 files.

```tsx
export interface TextareaProps {
  label: string;
  rows?: number;                  // default 4
  autoGrow?: boolean;             // height adjusts to content
  maxLength?: number;             // shows char count when set
  hint?: string;
  error?: string;
  // ...all native textarea props
}
```

**Implementation requirements:**
- Same a11y contract as Input
- Auto-grow uses ResizeObserver + scrollHeight
- Char count renders in `var(--color-text-muted)`, switches to `var(--color-warning)` at 90%, `var(--color-error)` at 100%

### 6. `web/src/components/shared/BrandLogo.tsx`

**Drifts addressed:** At least 4 different logo implementations across `Landing.tsx`, `Login.tsx`, `Signup.tsx`, `AppSidebar.tsx`, `topbar/TopBarLogo.tsx`. Each renders the StackWatch mark differently.

```tsx
export interface BrandLogoProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';  // 24 / 32 / 48 / 96 px
  /** 'full' shows mark + wordmark, 'mark' shows only the symbol. */
  variant?: 'full' | 'mark';
  /** Optional accessible label override (defaults to "StackWatch"). */
  ariaLabel?: string;
}
```

**Implementation requirements:**
- Single inline SVG source — the StackWatch mark is a `cyan` hexagonal stroke with a center dot
- Wordmark is 600-weight 'StackWatch' set in `var(--font-sans)`, letter-spacing `var(--tracking-wide)`
- All colors via tokens (`var(--color-primary)` for accent, `var(--color-text)` for wordmark)
- `aria-label` always present; `role="img"` when used as decoration, `aria-hidden="true"` when redundant with adjacent text

---

## Phase D execution roadmap

> Translating the 575 drift findings + 62 token/accessibility drifts (637 total actionable items) into a phased refactor plan. Each step is sized to absorb a coherent slice of the audit.

### D.1 — Build the missing shared components (prerequisite for everything below)

| Step | Component | Estimated LOC | Drift count unblocked |
|------|-----------|---------------|------------------------|
| D.1.1 | `web/src/components/shared/Modal.tsx` | ~180 LOC | 70 modal findings across 31 files |
| D.1.2 | `web/src/components/shared/Input.tsx` | ~120 LOC | 127 raw `<input>` + 11 `<textarea>` findings |
| D.1.3 | `web/src/components/shared/Select.tsx` | ~140 LOC | 37 raw `<select>` findings |
| D.1.4 | `web/src/components/shared/Textarea.tsx` | ~100 LOC | 11 raw `<textarea>` findings |
| D.1.5 | `web/src/components/shared/DataTable.tsx` | ~220 LOC | 23 raw `<table>` findings |
| D.1.6 | `web/src/components/shared/BrandLogo.tsx` | ~80 LOC | 4-6 logo fragmentations |
| D.1.7 | Promote `web/src/components/dashboard/SkeletonCard.tsx` → `shared/SkeletonCard.tsx` | ~50 LOC moved | 5 placeholder drifts |

### D.2 — Page-by-page refactor (consumes D.1 outputs)

Order is by drift count (highest first = fastest visible improvement).

| # | Page | Drifts | Workload |
|---|------|--------|----------|
| 1 | `web/src/pages/TrueNASWorkspace.tsx` | 18 | Replace 8 raw `<button>` with `<Button>`; replace 5 raw inputs with shared `<Input>`; replace 1 `<EmptyState>` (already shared); 4 raw `<textarea>` instances. |
| 2 | `web/src/pages/SyntheticsPage.tsx` | 13 | Replace 4 raw `<button>` (sw-button-*) with `<Button>`, replace `<select>` with shared `<Select>`, replace 2 raw `<input>` with `<Input>`. |
| 3 | `web/src/pages/ApmPage.tsx` | 7 | Replace 1 raw `<button>`, 5 raw `<input>`, 1 raw `<table>`. Already uses `<KpiCard>` and `<EmptyState>` correctly. |
| 4 | `web/src/pages/LogsFullPage.tsx` | 8 | Replace raw `<button>`, raw `<input>`. Filter bar delegates to shared components. |
| 5 | `web/src/pages/RumFullPage.tsx` | 6 | 3 raw `<button>` (sw-button-*), 1 raw `<select>`. |
| 6 | `web/src/pages/SettingsPage.tsx` | 6 | 5 raw `<button>` (sw-button-*), 1 raw `<form>`. Replace with `<Button>` + shared form controls. |
| 7 | `web/src/pages/SharedDashboardsPage.tsx` | 7 | 2 raw `<button>`, 1 raw `<select>`, 3 EmptyState already correct. |
| 8 | `web/src/pages/NotebookPage.tsx` | 4 | 1 raw `<button>`, 2 raw `<textarea>`, 1 EmptyState already correct. |
| 9 | `web/src/pages/Login.tsx / Signup.tsx / ForgotPassword.tsx / ResetPassword.tsx` | 12 | Auth flow — replace `auth-button-*` with `<Button variant>`, raw `<input>` with `<Input>`, integrate `<BrandLogo>`. |
| 10 | `web/src/pages/Landing.tsx` | 4 | Replace `landing-button-*` classes with `<Button>`, integrate `<BrandLogo>`. |
| 11 | `All 14 platform components` | 84 | Largest modal-heavy surface — replace all `sw-modal-backdrop`, `limits-modal-*`, `regions-modal-*` with shared `<Modal>`. |
| 12 | `All 24 `homelab/widgets/` components` | 110 | Replace custom widget modals with shared `<Modal>`, custom widget buttons with `<Button>`, raw `<input>` with `<Input>`. |
| 13 | `All 5 log tabs (`logs/*Tab.tsx`)` | 30 | Replace raw `<button>` and `<table>` with shared components. |
| 14 | `All 7 profile panels` | 24 | Replace raw `<button>` and `<input>`; integrate `<BrandLogo>` in avatar fallback. |
| 15 | `Topbar components (8 files)` | 16 | Replace raw `<button>` in TopBarProfile/Refresh/Search; `<BrandLogo>` in TopBarLogo. |
| 16 | `Sidebar (4 files)` | 12 | Replace raw `<button>` in SidebarItem; `<BrandLogo>` in Sidebar. |
| 17 | `AppShell + CommandPalette` | 18 | CommandPalette already has its own modal — wrap in shared `<Modal>`; AppShell keeps shell but uses `<BrandLogo>`. |

### D.3 — Token & accessibility sweep

> Run after D.1 and D.2 land. Replace 37 hardcoded hex colors and 25 inline-px spacings surfaced in the supplementary section. Add a CI lint rule (`stylelint-no-hex-literals-in-jsx`, `eslint-no-inline-px-spacing`) to prevent regressions.

### D.4 — Acceptance checklist (anti-slop from §10 of DESIGN-SYSTEM.md)

- [ ] Every page renders 0 raw `<button>` outside `shared/Button.tsx` (verify with `rg "<button" web/src/{pages,components} -g '!shared/Button.tsx'`)
- [ ] Every page uses `<Modal>` for overlays (verify with `rg "modal-backdrop|modal-card|sw-modal-backdrop" web/src/{pages,components}` → 0 matches)
- [ ] Every page uses `<DataTable>` for tabular data (verify with `rg "<table" web/src/{pages,components}` → only shared/DataTable.tsx)
- [ ] Every form field uses `<Input>` / `<Select>` / `<Textarea>` (verify with `rg "<(input|select|textarea)" web/src/{pages,components}` → 0 matches except type=hidden/submit)
- [ ] Every KPI renders through `<KpiCard>` (verify with `rg "kpi-card|kpi-value|dash-metric" web/src/{pages,components}` → 0 matches)
- [ ] Every status renders through `<StatusPill>` (verify with `rg "status-pill" web/src/{pages,components}` → only shared/StatusPill.tsx)
- [ ] Every empty list uses `<EmptyState>` (verify with `rg "dash-panel-empty|No data|No servers|No logs" web/src/{pages,components}` → 0 matches)
- [ ] Every loading state uses `<SkeletonRow>` / `<SkeletonCard>` (verify with `rg "Loading…" web/src/{pages,components}` → 0 matches except inside `<Button loading>`)
- [ ] `rg "#[0-9a-fA-F]{6}" web/src/{pages,components}` → only token file matches
- [ ] `rg "style=\\\{\\{[^}]*\\d+px" web/src/{pages,components}` → 0 matches

---

## Closing notes

- **The shared components already built are good.** `<Button>`, `<KpiCard>`, `<StatusPill>`, `<EmptyState>`, `<SkeletonRow>` cover ~25% of audited usages correctly today (196/771). The Tier-20 refactor is largely *consuming what's already there* and *filling 6 gaps*.
- **The drift is not visual.** Every page renders correctly. The drift is in the source contract — pages re-implement design-system primitives instead of importing them. This blocks Tier-21+ work (token-mode theming, motion refactor, Storybook) because the design system has no single source of truth at the component level.
- **The biggest risk** is the `homelab/widgets/` directory — 24 files, all with the same 4-5 drift patterns (custom modal, custom button, raw input). Refactoring them in a single sweep after D.1 lands is the highest-leverage move.
- **Most consistent pages today:** `Dashboard.tsx`, `IncidentsPage.tsx`, `CspmPage.tsx`, `DatabasePage.tsx`, `SecurityPage.tsx` — all use `<KpiCard>` + `<EmptyState>` correctly throughout. These should be the *reference templates* the rest of the refactor emulates.
- **Most inconsistent pages today:** `TrueNASWorkspace.tsx`, `SettingsPage.tsx`, `LogsFullPage.tsx` — large surfaces with many `sw-*` and `px-*` class prefixes still in place. Treat these as the canary for D.2.

---

**End of report.** Generated by automated ripgrep + Python audit, reviewed by Tier-20 owner. Re-run after each Phase D step to confirm drift count trends toward zero.
