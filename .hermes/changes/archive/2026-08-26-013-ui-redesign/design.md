# Design: 013 — UI/UX Redesign (System + Shell)

## Approach

Build the design system FIRST, then wrap the shell around every existing route. Pages keep their internal renderer — only the chrome around them changes. This means we ship visible improvements (topbar/sidebar look better) without touching the risky 30 page renderers.

**Layer order (bottom → top):**
1. `tokens.css` — design system variables
2. `base.css` — reset, typography, scrollbars, focus
3. `<AppShell>` — layout wrapper (topbar + sidebar + main)
4. Existing page renderers — unchanged, just rendered inside `<AppShell>`

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| CSS custom properties over Tailwind | StackWatch already uses Vite + plain CSS, no Tailwind installed. Tokens are lighter than Tailwind for a design system foundation. | Tailwind ecosystem patterns can't be reused; need to hand-write utilities. |
| No new npm deps for design system | Avoid dependency creep. Pure CSS handles tokens. | If we want Radix/shadcn later, we'd need to add deps. |
| AppShell wraps all routes in App.tsx | One shell = consistent layout, no per-page boilerplate. | If shell breaks, all pages break. Mitigated by single-file AppShell. |
| Keep page renderers unchanged | Don't break 30+ pages. Pure polish pass. | Pages still look inconsistent inside the new shell (visual debt for future phases). |
| framer-motion for transitions | Already in package.json; user-mandated universal standard. | Adds ~50KB to bundle (acceptable for UX quality). |
| Remove .go files via git rm | Backend handler files don't belong in frontend tree. | Lose history if anyone needs it — but they're duplicates of `internal/handler/*`. |

## Implementation details

### File tree

```
web/src/
├── styles/
│   ├── tokens.css           (NEW — 200 LOC, all CSS variables)
│   ├── base.css             (REWRITE — reset + typography, ~150 LOC)
│   ├── dashboard.css        (existing — unchanged)
│   ├── profile.css          (existing — unchanged)
│   ├── landing.css          (existing — unchanged)
│   ├── auth.css             (existing — unchanged)
│   ├── proxmox*.css         (existing — unchanged)
│   └── DESIGN-SYSTEM.md     (NEW — human-readable design language doc)
├── components/
│   ├── AppShell.tsx         (NEW — main shell wrapper)
│   ├── topbar/
│   │   ├── TopBar.tsx       (NEW — orchestrator)
│   │   ├── TopBarLogo.tsx   (NEW — 60 LOC)
│   │   ├── TopBarBreadcrumb.tsx (NEW — 80 LOC)
│   │   ├── TopBarSearch.tsx (NEW — command palette trigger, 100 LOC)
│   │   ├── TopBarTime.tsx   (NEW — wraps existing TimeWidget)
│   │   ├── TopBarWeather.tsx (NEW — wraps existing weather)
│   │   ├── TopBarRefresh.tsx (NEW — refresh + animation, 60 LOC)
│   │   ├── TopBarNotifications.tsx (NEW — bell + dropdown, 120 LOC)
│   │   └── TopBarProfile.tsx (NEW — wraps existing ProfileMenu)
│   ├── sidebar/
│   │   ├── Sidebar.tsx      (NEW — orchestrator)
│   │   ├── SidebarSection.tsx (NEW — section header + items, 80 LOC)
│   │   ├── SidebarItem.tsx  (NEW — single nav item, 100 LOC)
│   │   ├── SidebarFooter.tsx (NEW — status + version + signout, 80 LOC)
│   │   └── nav-config.ts    (NEW — typed nav structure, 150 LOC)
│   ├── AppSidebar.tsx       (DELETE — replaced by sidebar/Sidebar.tsx)
│   ├── shared/handler/      (DELETE — 145 .go files)
│   ├── ...                  (existing — unchanged)
└── App.tsx                  (MODIFY — wrap routes in AppShell)
```

### tokens.css structure

```css
:root {
  /* Color */
  --color-bg: #0a0e1a;
  --color-surface: #131826;
  --color-surface-elevated: #1a2138;
  --color-surface-hover: #1f2742;
  --color-border: rgba(255, 255, 255, 0.06);
  --color-border-strong: rgba(255, 255, 255, 0.12);

  --color-text: #f0f4fc;
  --color-text-muted: #94a3b8;
  --color-text-disabled: #475569;
  --color-text-inverse: #0a0e1a;

  --color-primary: #22d3ee;
  --color-primary-hover: #67e8f9;
  --color-primary-muted: rgba(34, 211, 238, 0.12);

  --color-success: #10b981;
  --color-success-muted: rgba(16, 185, 129, 0.12);
  --color-warning: #f59e0b;
  --color-warning-muted: rgba(245, 158, 11, 0.12);
  --color-error: #ef4444;
  --color-error-muted: rgba(239, 68, 68, 0.12);

  /* Spacing */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --space-12: 48px;
  --space-16: 64px;
  --space-24: 96px;
  --space-32: 128px;

  /* Type */
  --font-sans: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-mono: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  --text-xs: 11px;
  --text-sm: 12px;
  --text-base: 14px;
  --text-md: 16px;
  --text-lg: 18px;
  --text-xl: 20px;
  --text-2xl: 24px;
  --text-3xl: 30px;
  --text-4xl: 36px;
  --text-5xl: 48px;
  --text-6xl: 60px;
  --leading-tight: 1.2;
  --leading-normal: 1.5;
  --leading-relaxed: 1.75;
  --tracking-wide: 0.04em;
  --tracking-widest: 0.08em;

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-xl: 12px;
  --radius-2xl: 16px;

  /* Shadow */
  --shadow-0: none;
  --shadow-1: 0 1px 2px rgba(0, 0, 0, 0.2), 0 1px 1px rgba(0, 0, 0, 0.12);
  --shadow-2: 0 4px 8px rgba(0, 0, 0, 0.2), 0 2px 4px rgba(0, 0, 0, 0.12);
  --shadow-3: 0 12px 24px rgba(0, 0, 0, 0.3), 0 4px 8px rgba(0, 0, 0, 0.18);

  /* Motion */
  --duration-fast: 100ms;
  --duration-base: 150ms;
  --duration-medium: 200ms;
  --duration-slow: 300ms;
  --duration-slower: 500ms;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-in: cubic-bezier(0.4, 0, 1, 1);
  --ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);

  /* Layout */
  --topbar-height: 56px;
  --sidebar-width: 240px;
  --sidebar-width-collapsed: 64px;
  --content-max-width: 1440px;
  --content-padding: var(--space-6);
}

* { box-sizing: border-box; }
```

### AppShell layout

```tsx
<>
  <a href="#main" className="skip-link">Skip to main content</a>
  <div className="app-shell">
    <Sidebar />
    <TopBar />
    <main id="main" className="app-main">
      <AnimatePresence mode="wait">
        <motion.div
          key={location.pathname}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -8 }}
          transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
        >
          <Outlet /> {/* or children */}
        </motion.div>
      </AnimatePresence>
    </main>
  </div>
</>
```

### Sidebar structure

`nav-config.ts`:
```typescript
export const NAV_SECTIONS = [
  {
    label: 'WORKSPACE',
    items: [
      { path: '/dashboard', label: 'Overview', icon: 'home' },
      { path: '/billing', label: 'Billing', icon: 'credit-card' },
      { path: '/homelab', label: 'Homelab', icon: 'home' },
      { path: '/profile', label: 'Profile', icon: 'user' },
      { path: '/settings', label: 'Settings', icon: 'cog' },
    ],
  },
  // ... 3 more sections
] as const;
```

## Risks & mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Token values feel off (wrong shade of blue, too cramped) | High — visual rejection | Build Phase 0 in isolation, user reviews DESIGN-SYSTEM.md BEFORE building shell |
| Shell wraps break page renderers | High — broken UX for all routes | Smoke-test all 21 routes immediately after wrapping; rollback commit if needed |
| Mobile drawer has a11y issues | Medium | Use Radix-style focus trap (or simple manual trap), aria-modal, escape closes |
| framer-motion bundle size | Low | Already in package.json, no new dep |
| Sidebar items don't match routes | High | Single source of truth (`nav-config.ts`) drives both sidebar AND route guards |
| TypeScript types break | Medium | Run `tsc --noEmit` after every batch of file changes |

## Anti-slop checklist (must pass at end of Phase 1)

- [ ] All colors come from tokens, no hardcoded hex in component CSS
- [ ] All spacing is from the scale (4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 / 96 / 128)
- [ ] Hover / focus / active states on every interactive element
- [ ] Skip-link to main content works
- [ ] Keyboard nav works (Tab, Shift+Tab, Escape)
- [ ] All animations use framer-motion + token durations
- [ ] No placeholder text anywhere
- [ ] Empty states have guidance (not just "no data")
- [ ] Loading states have spinners (not blank screens)
- [ ] Error states explain what to do
- [ ] Lighthouse a11y ≥90 on Overview