# Product Review — Agent Dashboard (web-agent)
**Date**: 2026-04-08 | **URL**: http://localhost:3105/ | **App**: web-agent

> **Note**: The Patch Manager (port 3001) was offline. This review covers the Agent web UI at port 3105.

---

## Overall Verdict: ❌ FAIL

**4 critical · 9 major · 18 minor · 3 nitpick = 34 total findings**

> This page is not ready to ship. Address the critical and major findings, then re-run `/product-review`.

---

## Detection Layers

| Layer | Total | Passed | Failed |
|-------|-------|--------|--------|
| Automated assertions | 8 | 7 | 1 |
| UX perspective findings | 22 | — | 22 |
| QA perspective findings | 16 | — | 16 |
| Enterprise Buyer findings | 25 | — | 25 |
| **Merged / deduplicated** | **34** | — | **34** |

---

## Perspective Verdicts

| Perspective | Verdict | Critical | Major | Minor | Nitpick |
|-------------|---------|----------|-------|-------|---------|
| UX Designer | FAIL | 3 | 8 | 8 | 3 |
| QA Engineer | FAIL | 3 | 1 | 9 | 3 |
| Enterprise Buyer | FAIL | 3 | 5 | 14 | 3 |

All three perspectives voted FAIL. The CPU arithmetic subsystem and responsive layout are the primary blockers.

---

## Critical Findings (must fix before any customer demo)

| ID | Element | Observation | Suggestion |
|----|---------|-------------|------------|
| PO-001 | Top stat strip — CPU | Displays "7,953,642,733%" at 1024px, "429,496,724,000%" at 768px — 32-bit integer overflow. | Compute as float64 (delta_busy/delta_total)×100, clamp [0,100], single shared formatter for header + Resources card. |
| PO-002 | Router fallback — invalid routes | `/dashboard`, `/status` etc. render raw "Unexpected Application Error! 404 Not Found". | Add `path:'*'` catch-all with branded 404 page + "Back to Overview" button. Add redirect `/dashboard` → `/`. |
| PO-003 | Dashboard grid at 768px | Two-column grid compresses to narrow left strip. System Info text overlaps. Layout breaks, not degrades. | Below 1024px: single-column full-width grid. Stat strip wraps to 2×3. System Info stacks vertically below 768px. |
| PO-004 | Header CPU vs Resources CPU | Header shows 0% (implausible on active Mac), Resources shows 0%, Hardware shows 100% simultaneously. Same metric, three contradictory values. | Single hook source of truth for CPU value. Both widgets subscribe to same derived value. Add unit test asserting header == Resources. |

---

## Major Findings

| ID | Element | Observation | Suggestion |
|----|---------|-------------|------------|
| PO-005 | All routes — missing h1 | Zero h1 on Overview, Patches, Hardware, Software, Services, History, Logs. WCAG 2.1 AA fail. | Add `<h1>` via shared PageHeader component on every route. |
| PO-006 | Compliance card | Occupies half a row to say "check your PM dashboard". Dead placeholder. | Fetch last-cached score with stale indicator + "Open PM" deep link, or remove card entirely. |
| PO-007 | Agent Health — Version | Displays "dev". Procurement cannot track it. | Inject semver at compile time. Display "1.0.0-rc.2 (abc1234)". |
| PO-008 | Sidebar tooltips | "Patches"/"History"/"Settings" tooltips float over page content, not adjacent to sidebar icon. | Radix Tooltip `side='right'`, portal z-index above content, 8px offset, hide on route change. |
| PO-009 | Sidebar — icon-only nav | 8 unlabeled monochrome glyphs. Users cannot distinguish Hardware from Software from Services. | Labeled rail (icon + text) by default, or expandable rail with sticky-expand toggle. |
| PO-010 | CPU 100% bar vs "Healthy" pill | Bar is solid red but global status says "Healthy" — contradictory. Color only, no text label. | State machine: Healthy → Degraded → Critical. Propagate to health pill. Pair color with text "100% — Critical". |
| PO-011 | Stat strip at 1024px | Six pills crammed with no breathing room even with correct values. | Below 1280px: responsive card row with wrap, min-width per item, 16px padding. |
| PO-012 | Agent ID — raw UUID | "d4a49fb8…" with copy icon as primary identifier. Enterprise.md forbids UUIDs as primary ID. | Hostname as primary label. Move UUID to 'Diagnostics' collapsible in Settings. |
| PO-013 | CPU core count | Shows "1 cores" on 8-core MacBook Air. Load thresholds wrong. Entire CPU subsystem unreliable. | Fix: `runtime.NumCPU()` or `sysctl -n hw.logicalcpu` on darwin. Fix plural "core/cores". |

---

## Minor Findings

| ID | Element | Observation |
|----|---------|-------------|
| PO-014 | Dashboard cards | No staleness/last-updated indicator. Users can't tell if 0% CPU is fresh or stale. |
| PO-015 | Favicon | favicon.ico returns 404 — console error on every page load. |
| PO-016 | Logs page | No severity color coding, monotonic heartbeat entries. |
| PO-017 | Settings page | No visible Save button. Unclear if auto-save or explicit commit required. |
| PO-018 | Network I/O (Hardware) | Raw BSD interface names (gif0*, stf0*), asterisks unexplained, all showing 0 B/s. |
| PO-019 | Relative timestamps | "16s ago" with no absolute time tooltip. |
| PO-020 | Sidebar avatar "AG" | User avatar slot shows agent identity, not a real user. Confusing model. |
| PO-021 | System Info section | Bare row without card chrome, unlike every other section. |
| PO-022 | Services page | 490 raw launchd entries by default. No curation. |
| PO-023 | Hostname in top bar | Monospace font, no "Endpoint:" label, reads as debug breadcrumb. |
| PO-024 | History page | Description paragraph with no h1 above it. Inverted heading hierarchy. |
| PO-025 | Network stat separator | Uses "+" as RX/TX separator (reads as addition). |
| PO-026 | Patches page stat cards | Persistent green border on first card, slashed-zero glyphs read as "ø/null". |
| PO-027 | Software page | claude-code 2.1.85 listed as managed software. Will raise questions from enterprise buyers. |
| PO-028 | Services page filter | No "Showing N of 490" count when filter is active. |
| PO-029 | Theme toggle | No aria-label, no tooltip text on icon-only button. |
| PO-030 | Logs column width | Viewer narrower than available space — wastes readability. |
| PO-031 | Tenant indicator | No indication which PM tenant this agent is enrolled to. |

---

## Nitpick Findings

| ID | Element | Observation |
|----|---------|-------------|
| PO-032 | Light mode | Theme toggle exists but light mode parity unverified — every screenshot is dark. |
| PO-033 | Brand mark | Generic shield glyph. Not distinctive for a $500K product. |
| PO-034 | Software filter header | SOURCE/CATEGORY filter rows cramped spacing, buttons edge-to-edge. |

---

## Dev Checklist

### Critical — Fix immediately
- [ ] **PO-001** Fix CPU integer overflow: compute as float64, clamp [0,100], single shared formatter
- [ ] **PO-002** Add branded 404 catch-all route + redirect `/dashboard` → `/`
- [ ] **PO-003** Fix responsive layout: single-column below 1024px, stat strip wraps, System Info stacks
- [ ] **PO-004** Single source of truth for CPU value — header + Resources card must agree

### Major — Fix before next demo
- [ ] **PO-005** Add `<h1>` PageHeader to every route (Overview, Patches, Hardware, Software, Services, History, Logs)
- [ ] **PO-006** Fix Compliance card: real data + deep link, or remove it
- [ ] **PO-007** Inject build-time semver, display version string
- [ ] **PO-008** Fix sidebar tooltip positioning: `side='right'`, portal z-index, hide on route change
- [ ] **PO-009** Add labels to sidebar nav (icon + text or expandable rail)
- [ ] **PO-010** Drive health pill from resource state machine (Healthy/Degraded/Critical) + pair color with text label
- [ ] **PO-011** Responsive stat strip: wrap to 2×3 below 1280px
- [ ] **PO-012** Move Agent UUID to Diagnostics in Settings, hostname as primary identifier
- [ ] **PO-013** Fix CPU core detection (`runtime.NumCPU()` on darwin), fix plural "core/cores"

### Minor — Polish pass
- [ ] **PO-014** Add "Updated Xs ago" to each live card + manual refresh button
- [ ] **PO-015** Add favicon.ico + favicon.svg to public/
- [ ] **PO-016** Logs: color by severity, group repeated lines with count badge
- [ ] **PO-017** Settings: sticky Save button that appears when fields are dirty
- [ ] **PO-018** Network I/O: filter to active interfaces, add type labels, legend for asterisks
- [ ] **PO-019** All relative timestamps: tooltip with absolute ISO 8601 + local TZ
- [ ] **PO-020** Sidebar avatar: remove or replace with 'Local Agent' system badge
- [ ] **PO-021** System Info: wrap in Card with "System Information" title
- [ ] **PO-022** Services: default to managed services only, "Show all system services" toggle
- [ ] **PO-023** Hostname: use body font, prepend "ENDPOINT" label
- [ ] **PO-024** History page: add `<h1>History</h1>` above description
- [ ] **PO-025** Network stat: "↓ KB/s • ↑ KB/s" format, no "+" separator
- [ ] **PO-026** Patches: remove persistent green border / fix to active-filter tint; disable slashed-zero
- [ ] **PO-027** Software: categorize dev tools separately, clarify "managed" definition
- [ ] **PO-028** Services: "Showing N of 490" count on filtered views
- [ ] **PO-029** Theme toggle: add `aria-label` + tooltip
- [ ] **PO-030** Logs: full available width, horizontal scroll for long lines
- [ ] **PO-031** Add PM server/tenant indicator in Agent UI

### Nitpick — When bandwidth allows
- [ ] **PO-032** Verify light mode parity or hide toggle
- [ ] **PO-033** Commission distinctive logomark
- [ ] **PO-034** Fix Software filter row spacing to 16px gap

---

## Screenshots Captured

| File | Content |
|------|---------|
| 00-initial.png | Agent dashboard at full viewport |
| 01-scrolled.png | Full-page scroll |
| 02-hover-stat.png | CPU hover — live update to 100% |
| 03-patches-page.png | /pending — empty state |
| 04-hardware-page.png | Hardware — live system monitor |
| 05-software-page.png | Software — 17 managed packages |
| 06-services-page.png | Services — 490 services |
| 07-history-page.png | History — empty state |
| 08-logs-page.png | Logs — real-time |
| 09-settings-page.png | Settings |
| responsive-1024.png | 1024×768 — CPU overflow visible |
| responsive-768.png | 768×1024 — layout collapse visible |

---

**This page is not ready to ship.** Fix PO-001 through PO-004 first — the CPU arithmetic bug alone ends any sales demo. Then address the 9 major findings before the next review cycle.
