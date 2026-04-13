# PatchIQ Enterprise Standards — Project Override

> Overrides the base `enterprise.md` with PatchIQ-specific enterprise readiness rules for the POC deployment.

---

## Accent Color as Enterprise Feature

The configurable accent color system is a selling point — it signals brand-awareness and multi-tenant readiness. Reviewers MUST verify:

1. **ThemeConfigurator is accessible**: Users can find and use the accent/mode switcher without hunting.
2. **All 8 presets produce a polished result**: No preset should make the UI look broken (bad contrast, invisible text, clashing with signal colors).
3. **Ruby preset stress test**: When accent is ruby (#f43f5e, a red), signal-critical (#ef4444, also red) must still be distinguishable. If "Deploy" button and "Critical" badge look the same red, that's a critical violation. Signal colors are immutable — they never change with accent.
4. **Branding readiness**: A customer should be able to set their brand color as the accent and the product looks like "theirs". This is a key enterprise selling point.

## No Debug Artifacts

- No `console.log` visible in browser console during normal usage.
- No `TODO` or `FIXME` visible in the UI.
- No placeholder text ("Lorem ipsum", "Test", "asdf").
- No raw UUIDs/ULIDs as primary display — always show human-readable name.
- No "localhost" URLs visible in the UI (links, images, API responses shown to user).

## Browser Tab Titles

Every route must set `document.title` to: `"[Page Name] — PatchIQ"`.

Examples:
- "Endpoints — PatchIQ"
- "Endpoint: srv-prod-01 — PatchIQ"
- "Compliance — PatchIQ"
- "Dashboard — PatchIQ"

**Check**: open 5 tabs to different pages. Are all browser tab titles descriptive and consistent?

## First Impression (POC Demo Flow)

The product will be demoed to enterprise buyers. These pages are the first impression:

1. **Dashboard**: Must load fast, show meaningful data, communicate value in 10 seconds.
2. **Endpoints listing**: Must look data-dense and professional. Grid view should feel modern.
3. **Endpoint detail**: Must show risk context at a glance. Tabs must work smoothly.
4. **Compliance**: Must show framework scores prominently with context.
5. **Deployments**: Must show the deployment lifecycle clearly.

For these 5 pages, apply extra scrutiny — any finding is one severity level higher than usual.

## Scale Appearance

Even with seed data, the product must LOOK like it handles enterprise scale:

- Tables should show pagination controls even with < 25 items (shows the feature exists).
- Dashboard numbers should be formatted for large scale (thousands separators).
- "Showing 1-25 of 1,234" is better than "Showing all 12" for POC credibility.
- If demo seed data is sparse, the UI should still feel complete (no awkward whitespace from half-empty grids).
