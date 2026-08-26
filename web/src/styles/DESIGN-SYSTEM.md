# StackWatch Design System (v1.0)

> **Dark-first, Datadog-inspired, professional monitoring UI.**
> Single source of truth for colors, spacing, typography, elevation, and motion.
> All values defined as CSS custom properties in `web/src/styles/tokens.css`.

---

## 1. Philosophy

> **Taste over templates. Constraints breed creativity. Details matter.**

- Every color, spacing value, and duration comes from this system. No magic numbers.
- Consistent, = professional. Users value predictability.
- The system is the source of truth. Components consume tokens, never hardcode values.

---

## 2. Color

### Surfaces (dark theme — only theme this phase)

| Token | Value | Use |
|-------|-------|-----|
| `--color-bg` | `#0a0e1a` | App background |
| `--color-surface` | `#131826` | Card / panel background |
| `--color-surface-elevated` | `#1a2138` | Elevated card, active sidebar item |
| `--color-surface-hover` | `#1f2742` | Hover state on surface |

### Text

| Token | Value | Use |
|-------|-------|-----|
| `--color-text` | `#f0f4fc` | Primary text |
| `--color-text-muted` | `#94a3b8` | Secondary text, labels |
| `--color-text-disabled` | `#475569` | Disabled state |

### Accent

| Token | Value | Use |
|-------|-------|-----|
| `--color-primary` | `#22d3ee` | Brand accent, links, primary CTA |
| `--color-primary-hover` | `#67e8f9` | Primary hover |

### Semantic

| Token | Value | Use |
|-------|-------|-----|
| `--color-success` | `#10b981` | UP, healthy, online |
| `--color-warning` | `#f59e0b` | Stale, warning state |
| `--color-error` | `#ef4444` | DOWN, error state |

### Borders

| Token | Value | Use |
|-------|-------|-----|
| `--color-border` | `rgba(255,255,255,0.06)` | Subtle dividers |
| `--color-border-strong` | `rgba(255,255,255,0.12)` | Card borders, input borders |

---

## 3. Spacing

Use ONLY these values. Pick from the scale, never interpolate.

```
--space-1:   4px
--space-2:   8px
--space-3:  12px
--space-4:  16px
--space-6:  24px
--space-8:  32px
--space-12: 48px
--space-16: 64px
--space-24: 96px
--space-32: 128px
```

**Common patterns:**
- Card padding: `var(--space-6)` (24px)
- Card gap: `var(--space-4)` (16px)
- Inline gap: `var(--space-2)` (8px)
- Section gap: `var(--space-8)` (32px)

---

## 4. Typography

### Font stacks

```css
--font-sans: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
--font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
```

### Scale

| Token | Size | Use |
|-------|------|-----|
| `--text-xs` | 11px | Section headers (uppercase, letter-spaced) |
| `--text-sm` | 12px | Small labels, captions |
| `--text-base` | 14px | **Body text (default)** |
| `--text-md` | 16px | Large body, card titles |
| `--text-lg` | 18px | Section titles |
| `--text-xl` | 20px | Page titles |
| `--text-2xl` | 24px | Big page titles |
| `--text-3xl` | 30px | KPI values |
| `--text-4xl` | 36px | Hero KPI |
| `--text-5xl` | 48px | Landing hero |
| `--text-6xl` | 60px | Marketing hero |

### Line height & tracking

```css
--leading-tight:   1.2;   /* Headings */
--leading-normal:  1.5;   /* Body */
--leading-relaxed: 1.75;  /* Long-form text */
--tracking-wide:    0.04em; /* Buttons, labels */
--tracking-widest: 0.08em; /* Uppercase section headers */
```

**Rule:** body is 14px (not 16px). Monitoring tools need density.

**Rule:** tabular-nums on numeric KPI values:
```css
font-variant-numeric: tabular-nums;
```

---

## 5. Radius

```
--radius-sm:   4px  /* Badges, tags */
--radius-md:   6px  /* Buttons, inputs */
--radius-lg:   8px  /* Cards */
--radius-xl:  12px  /* Larger cards */
--radius-2xl: 16px  /* Panels, modals */
```

---

## 6. Elevation

```
--shadow-0: none;                                  /* Flat */
--shadow-1: 0 1px 2px rgba(0,0,0,0.2), 0 1px 1px rgba(0,0,0,0.12);  /* Cards (default) */
--shadow-2: 0 4px 8px rgba(0,0,0,0.2), 0 2px 4px rgba(0,0,0,0.12); /* Dropdowns */
--shadow-3: 0 12px 24px rgba(0,0,0,0.3), 0 4px 8px rgba(0,0,0,0.18); /* Modals */
```

---

## 7. Motion

### Durations

```
--duration-fast:    100ms  /* Hover state changes */
--duration-base:    150ms  /* Default transition */
--duration-medium:  200ms  /* Page transitions, panel slides */
--duration-slow:    300ms  /* Modal enter/exit */
--duration-slower:  500ms  /* Refresh button spin, hero animations */
```

### Easings

```
--ease-out:     cubic-bezier(0.16, 1, 0.3, 1);       /* Default for entering */
--ease-in:      cubic-bezier(0.4, 0, 1, 1);          /* Default for exiting */
--ease-in-out:  cubic-bezier(0.4, 0, 0.2, 1);        /* Looping, toggling */
```

**Rules:**
- Every interactive element has a hover transition
- Page transitions use 200ms ease-out
- Animations guide attention, never decorate

---

## 8. Layout

```
--topbar-height:    56px
--sidebar-width:    240px  /* Desktop */
--content-max-width: 1440px
--content-padding:  var(--space-6)  /* 24px */
```

---

## 9. Component patterns

### Cards

```css
.card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  box-shadow: var(--shadow-1);
  transition: box-shadow var(--duration-base) var(--ease-out);
}
.card:hover {
  box-shadow: var(--shadow-2);
}
```

### Buttons

```css
.btn-primary {
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-4);
  height: 40px;
  font-size: var(--text-base);
  font-weight: 500;
  transition: background var(--duration-base) var(--ease-out);
}
.btn-primary:hover {
  background: var(--color-primary-hover);
}
```

### KPI cards

```css
.kpi-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  position: relative;
  overflow: hidden;
}
.kpi-card::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 3px;
  background: var(--color-primary);  /* or success/warning/error */
}
.kpi-value {
  font-family: var(--font-mono);
  font-size: var(--text-3xl);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
```

---

## 10. Anti-slop checklist

Before declaring any UI work done, verify:

- [ ] No placeholder text (no "Lorem ipsum", no "TODO", no "Lorem")
- [ ] No stock photos
- [ ] All spacing from the scale (no 7px, 11px, 13px, etc.)
- [ ] All colors from tokens (no hardcoded hex in component CSS)
- [ ] Hover / focus / active states on every interactive element
- [ ] Skip-link to main content
- [ ] Loading states with spinners
- [ ] Empty states with guidance ("Add your first server" not "No data")
- [ ] Error states that explain what to do
- [ ] Mobile responsive (works at 320px and 1920px)
- [ ] Micro-interactions (button feedback, smooth transitions, not instant pops)
- [ ] Tabular-nums on all KPI values
- [ ] WCAG AA contrast minimum (4.5:1 for body text)

---

## 11. Reference products

We borrow aesthetic language from:

- **Datadog** — sidebar nav, KPI strip, dark theme, status pills
- **Linear** — micro-interactions, command palette, keyboard nav
- **Grafana Cloud** — graph styling, panel layout
- **Stripe Dashboard** — typography hierarchy, card density
- **Vercel** — minimal, clean, fast

We do NOT borrow:

- Marketing-heavy gradients on dashboards (looks great on landing, terrible for monitoring)
- Stock photos anywhere (looks cheap)
- Emoji as primary UI language (icons > emoji for professional tools)

---

## 12. Token usage

```tsx
// Good — uses tokens
<div style={{
  background: 'var(--color-surface)',
  padding: 'var(--space-6)',
  borderRadius: 'var(--radius-lg)',
}}>

// Bad — hardcoded values
<div style={{
  background: '#131826',
  padding: '23px',
  borderRadius: '7px',
}}>
```

```css
/* Good */
.card { color: var(--color-text-muted); }

/* Bad */
.card { color: #94a3b8; }
```

---

Last updated: 2026-08-26
Owner: StackWatch frontend
Next review: After Phase 1 ships (when shell is real)