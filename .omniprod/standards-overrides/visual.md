# PatchIQ Visual Design Standards — Project Override

> Overrides the base `visual.md` standard with PatchIQ-specific design tokens, color system, and theming rules.

---

## Design Philosophy: Precision Clarity

PatchIQ uses a 95% grayscale palette with exactly 3 semantic signal colors and a user-configurable accent. The platform is "dark-first but light-complete" — both modes must be fully functional and visually polished.

**Core principle**: grayscale carries structure, signals carry meaning, accent carries interaction. If you see a color that is not grayscale, a signal, or the accent — it is a violation.

---

## Color Token System (Non-Negotiable)

Every color in the platform MUST come from these CSS variables. Any hardcoded hex, rgb, hsl, or Tailwind color class (like `text-red-500`, `bg-emerald-600`) is a **critical violation**.

### Background Tokens

| Token | Dark Value | Light Value | Usage |
|-------|-----------|-------------|-------|
| `--bg-page` | #000000 | #f5f5f5 | Page background, app shell |
| `--bg-card` | #111113 | #ffffff | Cards, panels, table containers |
| `--bg-card-hover` | #1a1a1a | #f0f0f0 | Card hover state, MonoTag bg |
| `--bg-elevated` | #141414 | #fafafa | Dropdowns, popovers, slide panels |
| `--bg-canvas` | #080808 | #f8f8f8 | Canvas areas, inset regions |
| `--bg-inset` | #0a0a0a | #f0f0f0 | Icon containers, recessed wells |

**Check**: inspect any colored background. It must resolve to one of these variables. No `bg-zinc-900`, no `bg-neutral-800`, no `bg-gray-950`.

### Text Tokens

| Token | Dark Value | Light Value | Usage |
|-------|-----------|-------------|-------|
| `--text-emphasis` | #ffffff | #0a0a0a | Page titles, stat values, active tabs |
| `--text-primary` | #d4d4d4 | #1a1a1a | Body text, table cell content |
| `--text-secondary` | #a1a1a1 | #525252 | Subtitles, descriptions, timestamps |
| `--text-muted` | #7a7a7a | #737373 | Labels, placeholders, inactive items |
| `--text-faint` | #5c5c5c | #a1a1a1 | Disabled text, decorative text |

**Check**: every text element must use one of these. If you see `text-gray-400` or `color: #888` — violation.

### Border Tokens

| Token | Dark Value | Light Value | Usage |
|-------|-----------|-------------|-------|
| `--border` | #222222 | #e5e5e5 | Default borders on cards, inputs, tables |
| `--border-hover` | #2e2e2e | #d4d4d4 | Hover state borders |
| `--border-strong` | #333333 | #cccccc | Emphasized borders, active sections |
| `--border-faint` | #1a1a1a | #f0f0f0 | Subtle dividers, skeleton borders |

### Signal Colors (Semantic — NEVER use for decoration)

| Signal | Color | Subtle (8%) | Border (30%) | Meaning |
|--------|-------|-------------|--------------|---------|
| `--signal-healthy` | #22c55e | rgba(34,197,94,0.08) | rgba(34,197,94,0.3) | Success, online, compliant, passing, low risk |
| `--signal-critical` | #ef4444 | rgba(239,68,68,0.08) | rgba(239,68,68,0.3) | Error, offline, critical severity, failing, high risk |
| `--signal-warning` | #f59e0b | rgba(245,158,11,0.08) | rgba(245,158,11,0.3) | Warning, degraded, medium severity, at-risk |

**Rules**:
- Green ONLY means healthy/good/passing. Never "active" or "enabled" generically.
- Red ONLY means critical/error/danger. Never "important" or "highlighted".
- Amber ONLY means warning/caution/degraded. Never "in progress" or "pending".
- If a state does not map to healthy/critical/warning, use `--accent` or `--text-muted`.
- **Never use signal colors for backgrounds on large areas** — use the `-subtle` variant for backgrounds and `-border` for borders.

**Check**: find every green, red, and amber element. Does its semantic meaning match the signal definition? A green "Active" badge where active means "turned on" (not "healthy") is a violation — use accent instead.

### Accent Color System (User-Configurable)

| Token | Purpose |
|-------|---------|
| `--accent` | Primary interactive color: buttons, links, selected states, active tabs, focus rings |
| `--accent-subtle` | Accent at 8% opacity: selected card backgrounds, hover highlights, onboarding banners |
| `--accent-border` | Accent at 30% opacity: selected card borders, active section borders |

**8 presets** (stored in localStorage as `patchiq-theme-accent`):
- forest (#10b981), amethyst (#7c3aed), ocean (#3b82f6), arctic (#06b6d4)
- ruby (#f43f5e), ember (#f97316), twilight (#8b5cf6), mint (#2dd4bf)

**Critical rules**:
- The accent MUST flow through `var(--accent)` everywhere. Never hardcode a specific preset color.
- When accent changes, EVERY interactive element must update: buttons, links, focus rings, selected states, active tabs, toggle switches, progress bars, ring gauges.
- **Test**: switch between all 8 accent presets. Every accent-colored element must change. If any element stays the same color, it is hardcoded — critical violation.
- **Test**: switch between dark and light mode with each preset. The accent must remain visible and have sufficient contrast in both modes.
- Accent is NOT a signal color. Do not use accent to mean "healthy" or "success" — use `--signal-healthy` for that.

### Color Consistency Violations (Common Bugs)

Reviewers MUST check for these specific patterns:

1. **Hardcoded Tailwind colors**: `text-emerald-500`, `bg-red-100`, `text-amber-600` — must use CSS variables instead.
2. **Hardcoded hex in style objects**: `color: '#10b981'` — must use `color: 'var(--accent)'`.
3. **Signal/accent confusion**: Using `--accent` for status indicators (should be signal), or `--signal-healthy` for buttons (should be accent).
4. **Missing light mode**: Element looks correct in dark mode but has wrong colors in light mode — often caused by hardcoded dark-mode colors.
5. **Accent leak into signals**: When user picks a red accent preset (ruby), health indicators should still be green (`--signal-healthy`), not red like the accent.
6. **Stale color on theme switch**: Element does not update until page reload — CSS variable not wired correctly.

---

## Typography Scale

| Role | Size | Weight | Font | Tracking | Transform | Token |
|------|------|--------|------|----------|-----------|-------|
| Page title (h1) | 22px | 600 | sans | — | — | `--text-emphasis` |
| Section heading | 13-15px | 600 | sans | — | — | `--text-emphasis` |
| Section label | 10px | 600 | mono | 0.06em | uppercase | `--text-emphasis` or `--text-muted` |
| Body text | 13px | 400 | sans | — | — | `--text-primary` |
| Table header | 11px | 500 | mono | 0.05em | uppercase | `--text-muted` |
| Table cell | 13px | 400 | sans | — | — | `--text-primary` |
| Stat value | 28px | 700 | mono | -0.03em | — | `--text-emphasis` |
| Stat label | 10px | 500 | mono | 0.06em | uppercase | `--text-muted` |
| Stat sublabel | 11px | 400 | mono | — | — | `--text-secondary` |
| Small/meta | 11-12px | 400 | sans/mono | — | — | `--text-secondary` or `--text-muted` |
| Code/IDs | 12-13px | 400 | mono | — | — | `--text-secondary` |

**Rules**:
- `--font-sans` (Geist) for prose, labels, descriptions.
- `--font-mono` (GeistMono) for data, numbers, IDs, hostnames, technical values, table headers, section labels.
- All uppercase text MUST use letter-spacing 0.05-0.06em. Uppercase without tracking looks cramped.
- No arbitrary font sizes outside the scale (no 15px, 17px, 19px).

---

## Spacing & Sizing Scale

| Name | Value | Common Usage |
|------|-------|-------------|
| xs | 4px | Tight gaps, icon-to-text inline |
| sm | 8px | Tag padding, tight card gaps |
| md | 12px | Dashboard grid gap, table cell padding |
| lg | 16px | Section gaps, form field spacing |
| xl | 20px | Card vertical padding |
| 2xl | 24px | Card horizontal padding, large section gaps |
| 3xl | 32px | Page-level spacing |
| 4xl | 48px | Major section breaks |

**Card padding standard**: `20px 24px` (vertical horizontal).
**Grid gap standard**: `12px` for dashboard grids. `16px` for form field grids.
**Table cell padding**: `12px` horizontal, `9-12px` vertical.

---

## Border Radius Scale

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 4px | Small badges, inline tags |
| `--radius-md` | 6px | Buttons, inputs, form controls |
| `--radius-lg` | 8px | Cards, panels, modals |
| `--radius-xl` | 12px | Large containers, hero sections |

**Rules**:
- Interactive controls (buttons, inputs, selects): `--radius-md` (6px).
- Containers (cards, panels, dialogs): `--radius-lg` (8px).
- Never mix radii in the same context — all cards in a grid use the same radius.
- Toggle switches: `border-radius: 100px` (pill shape).

---

## Shadow & Elevation

| Token | Usage |
|-------|-------|
| `--shadow-sm` | Cards, default elevation |
| `--shadow-md` | Dropdowns, popovers |
| `--shadow-lg` | Modals, slide panels, kebab menus |
| `--shadow-glow` | Special emphasis (sparingly) |

---

## Transition Timing

| Token | Value | Usage |
|-------|-------|-------|
| `--transition-fast` | 150ms | Color changes, hover states, border transitions |
| `--transition-normal` | 200ms | Tab switches, panel slides, expand/collapse |
| `--transition-slow` | 300ms | Page transitions, complex animations |

**Rules**:
- All hover effects: `150ms ease`.
- Expand/collapse animations: `200ms ease-out`.
- No animation > 400ms in the product.
- All motion must respect `prefers-reduced-motion`.
