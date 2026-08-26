# Acceptance scenarios: Dashboard + Profile polish

## Sidebar

### Scenario S1 — User navigates to Dashboard
**Given** the user is on any page
**When** they click "Overview" in the sidebar
**Then** they land on `/dashboard`
**And** the Overview item in the sidebar shows the active style (cyan glow + left accent stripe)

### Scenario S2 — User navigates to Profile from avatar menu
**Given** the user is on the Dashboard
**When** they click the avatar in the topbar
**And** they click "Profile" in the dropdown
**Then** they land on `/profile`
**And** the Profile page shows its own sidebar with Profile as active

### Scenario S3 — Profile removed from Dashboard sidebar
**Given** the user is on `/dashboard`
**When** the sidebar renders
**Then** the sidebar contains: Overview, Billing, Proxmox, TrueNAS, Terminal
**And** "Profile" is NOT in the sidebar
**And** clicking the avatar in the topbar still shows "Profile" as a menu option

## KPI cards

### Scenario S4 — Healthy API shows green card
**Given** `/api/v1/health` returns `{"status": "ok"}`
**When** the Dashboard renders
**Then** the "API health" card has the `dash-metric-green` tone (green stripe + green icon)
**And** its value is "ok"

### Scenario S5 — Degraded API shows amber card
**Given** `/api/v1/health` returns `{"status": "degraded"}` or fails
**When** the Dashboard renders
**Then** the "API health" card has the `dash-metric-amber` tone
**And** its value is the actual status string or "—"

### Scenario S6 — All cards have SVG icons
**Given** the Dashboard renders
**When** looking at any of the 4 metric cards
**Then** the icon bubble contains an inline `<svg>` element (not text emoji)

## Error bar

### Scenario S7 — Session expired shows soft amber bar
**Given** the user has an expired/invalid JWT
**When** the Dashboard tries to load
**Then** the error bar renders with `dash-error-info` class (amber, not red)
**And** the title says "Sign in again"
**And** the body says "Your session ended. Sign back in to load live infrastructure data."
**And** the button says "Sign in"
**And** clicking the button clears the token and routes to `/login`

### Scenario S8 — Server error shows loud red bar
**Given** the API returns a 500
**When** the Dashboard tries to load
**Then** the error bar renders with the default (red) style
**And** the title says "Live data unavailable"
**And** the button says "Retry"
**And** clicking the button re-runs `loadDashboard()`

## Profile page

### Scenario S9 — Hero card has correct padding
**Given** the user is on `/profile`
**When** the hero card renders
**Then** its computed padding is `32px 36px`
**And** it shows the eyebrow "Your account" with cyan dot
**And** the email row shows a circular `@` icon followed by the email address

### Scenario S10 — Identity rows truncate cleanly
**Given** the user has a long `user_id` like `cdea78a8-ea94-4fc1-83d3-bfc01afb9a8c`
**When** the Identity panel renders
**Then** the User ID value cell shows the full UUID
**And** if the cell is narrower than the UUID, it truncates with ellipsis (no overflow)
**And** the copy button still works (copies the full UUID to clipboard)

### Scenario S11 — Copy button shows "✓ Copied" feedback
**Given** the user clicks the "Copy" button next to User ID
**When** the click handler runs
**Then** the button text changes to "✓ Copied" for ~1.6 seconds
**And** then resets to "⧉ Copy"

### Scenario S12 — Workspace stats have proper spacing
**Given** the user is on `/profile`
**When** the Workspace panel renders
**Then** each stat card has 14px 16px padding
**And** the 2x2 grid has 10px gaps between cards
**And** each stat card shows: label (uppercase tiny), value (18px), hint (tiny muted)

### Scenario S13 — Display name form has comfortable spacing
**Given** the user is on `/profile`
**When** the Display name form renders
**Then** form rows have 16px vertical gap
**Then** inputs are 14px font with 9px 12px padding
**And** when an input is focused, it has a cyan focus ring (3px `--accent-soft` shadow)
**And** the Discard / Save buttons have a top border separator above them

### Scenario S14 — Avatar upload works end-to-end
**Given** the user clicks the avatar (or the 📷 overlay)
**When** they pick a JPG file
**Then** the avatar shows a loading spinner
**Then** the file is resized to 256x256 JPEG quality 0.7 client-side
**Then** `PATCH /api/v1/auth/profile` is called with `{avatar_url: "data:image/jpeg;base64,..."}`
**And** on 200, the local state updates with the new URL
**And** the hero avatar re-renders showing the uploaded image

### Scenario S15 — API tokens inline panel
**Given** the user is on `/profile`
**When** they click "Manage" next to "API tokens"
**Then** an inline panel expands below the security list
**And** it loads the user's existing tokens from `GET /api/v1/api-keys`
**And** they can type a name, click "Create token", and see the new token displayed once
**And** the token is auto-copied to clipboard on creation

## Performance

### Scenario S16 — Bundle size budget
**Given** the polish changes are built
**When** the production bundle is generated
**Then** the CSS bundle is under 75KB (current: 72.4KB)
**And** the JS bundle is under 320KB (current: 315.1KB)
