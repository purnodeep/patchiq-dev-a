# Enterprise Buyer Review — PatchIQ Patch Manager
**Date**: April 3, 2026
**Perspective**: Enterprise Buyer (CTO / VP Engineering, 5k–50k endpoints)
**Reviewed pages**: /endpoints, /patches?view=card, /cves?view=card, /deployments?view=card, /alerts, /settings/general, /audit
**Reviewer**: OmniProd Enterprise Buyer Agent

---

## Verdict: FAIL

The product has two **critical routing failures** that would immediately disqualify it in any live evaluation. Navigating to `/patches` and `/audit` via URL produces the wrong page (Alerts and Alerts respectively). The card view toggle on CVEs navigates away from the page entirely. If a CISO clicked "Patches" in the sidebar during a live demo and was taken to the Alerts screen, the evaluation would be over.

Beyond the routing failures, there are consistent data quality issues — raw UUID fragments as entity names in the Deployments list, unformatted large numbers in Patches, and "dev-user" actor names visible in the audit log — that would raise immediate red flags about demo readiness and data credibility.

---

## Findings Table

| ID | Severity | Element | Observation | Suggestion |
|----|----------|---------|-------------|------------|
| EB-01 | **critical** | Routing — /patches URL | Navigating directly to `http://…/patches` (via URL bar or external link) renders the **Alerts** page. The sidebar highlights Patches correctly but content is wrong. This is a React Router misconfiguration. | Fix route ordering or route definition for `/patches` so it renders the Patches component, not the Alerts component. Verify with hard-refresh on all routes. |
| EB-02 | **critical** | Routing — /audit URL | Navigating directly to `http://…/audit` renders the **Alerts** page. Clicking the Audit sidebar link from an already-loaded page works correctly (SPA navigation), but a hard-reload of /audit fails. URL-addressed navigation is broken. | Fix the fallback/catch-all route or the route ordering so /audit resolves correctly on fresh load. |
| EB-03 | **critical** | Card view toggle — CVEs | Clicking the card view toggle button on `/cves` navigates the browser to the **Alerts** page instead of re-rendering CVEs in card layout. This is a button handler wired to the wrong navigation target. | Audit the card view toggle's onClick handler on the CVEs page. It appears to be linking to `/alerts?view=card` instead of `/cves?view=card`. |
| EB-04 | **critical** | Deployments list — raw UUID as entity name | 10 deployments display raw UUID fragments as their NAME (e.g., `e1d69c1d`, `f99b8ede`, `d140617f`, `477b3b22`, `c706dfa8`, `fd29d6dd`). These are unnamed/auto-created deployments where the name field was never set. These appear in both the table and card views. | Either require a name at deployment creation time, or generate a human-readable fallback name (e.g., "Auto-deployment Mar 25, 2026") instead of surfacing UUID fragments. |
| EB-05 | **major** | Patches card view — all cards show CVES=0, CVSS=0.0, AFFECTED=0 | Every visible patch card shows zeroed metrics. Either the data is genuinely empty (implying the patch catalog has no CVE associations on the displayed records) or the metrics are not loading. Either way, this communicates broken data to an evaluator. | Verify patch-to-CVE correlation data is being fetched. If data is legitimately 0 for these records, show `—` instead of `0.0` for CVSS to distinguish "unscored" from "scored 0". |
| EB-06 | **major** | Patches stat cards — unformatted numbers | The stat cards show `207222`, `30706`, `79657`, `92484`, `4371` — all without thousands separators. This violates basic numeric formatting standards and looks unprofessional for large datasets. | Apply `toLocaleString()` or equivalent to all large numeric stat card values. Expected: `207,222`, `30,706` etc. |
| EB-07 | **major** | Deployments table — "Resource: [full-UUID]" in audit log TARGET column | The audit log TARGET column shows raw unabbreviated UUIDs as values for many rows (e.g., `Resource: 8895a66a-3dc2-4d9f-a7d0-98b4702050cb`). Full UUIDs in a visible column communicate internal implementation detail, not business context. | Show a human-readable resource reference (entity name if available, or `[type]/[short-id]` format). Where entity names are not available in the audit payload, show `[workflow] 8895a66a…` at minimum. |
| EB-08 | **major** | Audit log — "dev-user" actor name visible | The audit log prominently shows `dev-user` as an actor for 30+ compliance and workflow events. This is clearly a development/seed data artifact that should not be visible in a client-facing evaluation. | Replace seed data actor names with representative names (`admin@acme.corp`, `jsmith`, etc.) before any demo. Add a data-review step to the demo preparation checklist. |
| EB-09 | **major** | Alerts — TITLE column truncates to "Comman..." | The TITLE column in the Alerts table is too narrow — every alert shows only "Comman..." (truncated "Command timed out"). An evaluator cannot read the alert title without expanding each row individually. Truncated critical security information fails the usability bar for enterprise software. | Widen the TITLE column or add an ellipsis tooltip so the full title is accessible on hover. At minimum, "Command timed out" should be fully visible. |
| EB-10 | **major** | Alerts — RESOURCE column shows raw UUID paths | The RESOURCE column displays values like `command/88a022...`, `command/aadebb...` — opaque UUID-based paths with no human context. An evaluator cannot understand what affected resource these alerts refer to. | Resolve resource references to meaningful names: the endpoint hostname, deployment name, or policy name. Show `command/[endpoint-name]` rather than `command/[UUID]`. |
| EB-11 | **major** | Settings — Organization name is "PatchIQ Test Org" | The General settings page shows the organization name as "PatchIQ Test Org" — clearly test/seed data. This would be noticed immediately in a demo against a client's tenant. | Change the demo org name to a realistic fictional name (e.g., "Acme Corporation" or "Meridian Financial"). Never use "Test" in visible seed data for evaluation environments. |
| EB-12 | **major** | Date format inconsistency: settings vs. UI | Settings/General has "Date Format" set to `YYYY-MM-DD (ISO 8601)` but the Deployments list displays dates as `Feb 25, 2026` and `Mar 5, 2026` — the `Mar 15, 2026` format. The format setting appears to have no effect on the UI date rendering. | Either wire the date format preference to the actual date rendering throughout the UI, or remove the setting until it is implemented. A non-functional setting is worse than no setting. |
| EB-13 | **major** | Endpoints table — "LAST SEEN" column truncated | The rightmost "LAST SEEN" column header is cut off as "LAST SEE" in the table header (text overflow). The column content also appears cut off. | Fix column width constraints so the last column label is fully visible. Use overflow controls or widen the table. |
| EB-14 | **major** | Deployments card view — progress bar for "Rolled Back" shows 100% green | The "Database Maintenance Window" card shows `Rolled Back` status with a 100% green progress bar. 100% green communicates success; a rolled-back deployment is explicitly not a success state. The green bar contradicts the status badge. | For Rolled Back, Failed, and Cancelled states, use a distinct progress bar color (red/amber) or remove the progress bar display entirely. A 100% completion bar for a failed operation is misleading. |
| EB-15 | **major** | Alerts stat cards show 82 TOTAL but filter shows 33 alerts | The top stat cards show 82 TOTAL alerts, but the filtered view (Active, Last 24h) shows "33 alerts". No callout explains the discrepancy. An evaluator will immediately wonder whether the numbers are wrong or miscalculated. | Add a visible callout below the stat cards explaining the filter context: "Showing 33 active alerts in the last 24h of 82 total." The 82 count should be clearly labeled as the unfiltered lifetime total if that's its meaning. |
| EB-16 | **minor** | CVEs card view — all cards show "Unknown" for vector/package | Every CVE card in the card view shows "Unknown" as the package/product association. If this is real data (no association computed), it communicates that CVE correlation is incomplete. In a demo context it looks like unfinished data. | Either enrich the CVE-to-package data in the seed set, or replace "Unknown" with a more precise label like "No package linked" or "Vendor advisory only". |
| EB-17 | **minor** | CVEs — EXPLOIT and KEV columns show "—" for most entries | The majority of CVEs show dashes in the EXPLOIT and KEV columns. While some CVEs genuinely have no exploit code, having nearly every entry show "—" makes these columns feel inert and useless in the default view. | Add at least several CVEs with confirmed exploit data and KEV listing in the seed set to demonstrate these filter capabilities during evaluation. |
| EB-18 | **minor** | Audit log — timestamps show only time, no full date for same-day entries | The audit log groups by date (e.g., "APRIL 3, 2026") but individual rows only show the time (`10:42:02.000`). There is no timezone indicator. For a multi-user enterprise tool, time-only without timezone is ambiguous. | Append the timezone to timestamps (e.g., `10:42:02 UTC`) or show full ISO 8601 timestamps on hover/tooltip. |
| EB-19 | **minor** | Settings — green dot on "Identity & Access" sidebar item | The settings sidebar shows a green dot next to "Identity & Access". The meaning of this indicator is not explained — it could mean connected, configured, active, or have an alert. Unexplained status indicators erode trust. | Add a tooltip explaining what the dot means (e.g., "Zitadel connected" or "SSO active"). If it indicates a configuration status, it should also have a red/amber state and be explained in context. |
| EB-20 | **minor** | Endpoints table — stat card for "PATCHING" shows "—" (dash) | The 4th stat card on the Endpoints page shows "—" for the PATCHING count with a muted label. A dash in a stat card position looks like a loading failure or a null data error rather than an intentional zero or unavailable state. | If the value is 0 or unavailable, show `0` or label the card "N/A — scanning in progress" with context. A bare dash in a headline metric card looks broken. |
| EB-21 | **minor** | Deployments table — "TARGETS" column shows two identical values split by "/" | The TARGETS column shows both "wave progress" and "endpoint total" as overlapping values (e.g., `5/5` in PROGRESS, and then `5/5` in TARGETS). The column semantics are unclear — are these the same number? What does each fraction represent? | Clearly differentiate column semantics with distinct labels. Consider "ENDPOINTS" for total, and keep "PROGRESS" for wave completion. Each column should be self-explanatory without reading documentation. |
| EB-22 | **nitpick** | Deployments card — redundant subtitle (policy name = card title) | Many deployment cards show the deployment name as the title (e.g., "Windows Update Policy") and the policy name as the subtitle (also "Windows Update Policy"). The same text appears twice with no differentiation. | If the deployment name matches the policy name, suppress the subtitle to reduce visual noise. Otherwise, differentiate with a "Policy:" prefix to add context. |
| EB-23 | **nitpick** | Audit log — avatar initials for "System" actor always show "SY" | The system actor shows a `SY` initials badge in teal. This is visually similar to a user avatar which could confuse evaluators into thinking "SY" is a user. | Use a distinct system icon (gear icon, robot icon, or `⚙`) instead of initials for automated/system events to visually distinguish system actions from user actions at a glance. |
| EB-24 | **nitpick** | CVEs stat cards — no count label on "KEV LISTED" card | The CVEs page has 4 stat cards: CRITICAL (6), HIGH (14), MEDIUM (28), KEV LISTED (11). The first three use severity badge colors; KEV LISTED uses amber. No tooltip explains what "KEV" means to an evaluator who may not know the CISA KEV catalog. | Add a tooltip: "Known Exploited Vulnerabilities — CISA catalog of actively exploited CVEs." This adds credibility and educates evaluators on PatchIQ's threat intelligence coverage. |
| EB-25 | **nitpick** | Alerts — "30s" auto-refresh interval is too aggressive for a demo | The alerts page shows an auto-refresh selector set to "30s". Every 30 seconds the table refreshes during a demo, which can be visually disruptive and confusing to an evaluator watching a live presentation. | Default to a longer interval (5 minutes) or offer a "Manual refresh" option as the default. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| Critical | 4 |
| Major | 12 |
| Minor | 6 |
| Nitpick | 3 |
| **Total** | **25** |

---

## Must-Fix Before Any Client Demo

The following 4 critical and 5 most damaging major findings must be resolved before this product is shown to any external evaluator:

1. **EB-01** — Fix /patches URL routing (shows Alerts)
2. **EB-02** — Fix /audit URL routing (shows Alerts)
3. **EB-03** — Fix CVEs card view toggle (navigates to Alerts)
4. **EB-04** — Fix UUID-named deployments in list and card views
5. **EB-06** — Add thousands separators to Patches stat card numbers
6. **EB-07** / **EB-08** — Replace raw UUIDs and "dev-user" in Audit log with readable data
7. **EB-09** — Widen Alerts TITLE column so full title is readable
8. **EB-11** — Remove "PatchIQ Test Org" from settings; use a realistic demo org name
9. **EB-15** — Explain the 82-total vs 33-shown discrepancy in Alerts

---

## What Is Working Well

These elements reflect solid product thinking and should be preserved:

- **Endpoints page**: The gold standard page lives up to its billing. Stat cards, status indicators, OS icons, tag badges, risk score bars, and the search/filter bar are all coherent and professional. This is the benchmark the other pages should match.
- **CVEs card view**: The CVSS gauge rings are a distinctive and credible design choice. The color progression from green → amber → red → dark-red communicates severity intuitively. The stat card breakdown (CRITICAL / HIGH / MEDIUM / KEV LISTED) is decision-relevant.
- **Deployments card view**: The progress bar with TARGETS / SUCCEEDED / FAILED breakdown is information-dense and useful. The Rollback CTA on completed deployments communicates operational awareness.
- **Audit log**: The Activity Stream / Timeline View toggle, export to CSV/JSON, and the retention footer ("Audit logs retained for 365 days") are enterprise-grade signals that demonstrate security maturity.
- **Alerts categorization**: The filter bar (Deployments / Agents / CVEs / Compliance / System) and status tabs (Active / Acknowledged / Dismissed) show thoughtful alert lifecycle management.
- **Navigation consistency**: The sidebar, topbar with search/notifications, and stat card pattern are applied consistently across reviewed pages. The visual language is coherent when pages load correctly.
