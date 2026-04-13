---
name: Endpoints page product review
description: OmniProd review of /endpoints page — FAIL with 4 critical, 18 major, 47 total findings across 5 perspectives
type: project
---

OmniProd full deep review of Endpoints page completed 2026-04-04. **FAIL** — 4 critical, 18 major, 20 minor, 5 nitpick (47 total).

**Why:** All 5 perspectives (UX, QA, Enterprise Buyer, Product Manager, End User) unanimously failed the page. This is the central asset inventory — the most-used page in the product.

**Critical blockers:**
1. PO-001: Compliance tab renders CVE Exposure content (wrong component mapped)
2. PO-002: Breadcrumb shows raw UUID instead of hostname
3. PO-003: Detail tabs (Patches/Deployments/Audit) may navigate to global pages
4. PO-004: OS column blank in table (empty spans, no text labels)

**Key major issues:**
- Stat cards show page-level counts, not fleet totals (misleading)
- Risk score missing /10 denominator and color coding in table
- No "Stale" status filter chip
- Risk threshold >=3 vs >=4 mismatch between listing and detail (QA code-verified)
- Status label "Pending" vs "Patching" inconsistency (QA code-verified)
- Status dot colors differ between listing and detail pages
- Pagination lacks per-page selector and page numbers
- Empty states minimal (no icon/CTA)
- Export dialog too simple (no format/column selection)

**Business rules failing:** BR-003 (risk thresholds), BR-010 (status labels), BR-024 (health strip), BR-026 (status colors)

**How to apply:** Fix all 4 criticals and top majors before any POC demo involving the Endpoints page. The QA-found code-level issues (PO-012/013/014) are quick fixes with high impact.
