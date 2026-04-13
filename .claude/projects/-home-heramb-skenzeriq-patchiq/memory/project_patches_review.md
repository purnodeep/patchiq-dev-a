---
name: OmniProd Patches page review
description: Full 5-perspective deep review of /patches — FAIL with 7 critical, 20 major, 42 total findings. Root cause is catalog data pipeline not enriching data.
type: project
---

OmniProd review of Patches page (http://localhost:3001/patches) on 2026-04-04: **FAIL**

**Why:** 7 critical + 20 major + 11 minor + 4 nitpick = 42 deduplicated findings from 5 perspectives (UX, Enterprise Buyer, QA, Product Manager, End User). All 5 perspectives returned unanimous FAIL.

**Top blockers (critical):**
1. PO-001: Patch names are CVE identifiers, not real patch names (KB/USN/RHSA)
2. PO-002: CVE Links empty for all 207K patches — correlation pipeline not running
3. PO-003: CVSS shows "--" on list but "0.0/10" on detail (zero vs null confusion)
4. PO-004: AFFECTED shows "0" for all patches (null displayed as zero)
5. PO-005: OS Family filter returns 0 for all OS types — os_family field unpopulated
6. PO-006: Stat card sum off by 4 (207,218 vs 207,222)
7. PO-007: Category column blank in list but shows "Security" on detail

**Root cause:** Catalog data pipeline ingests raw feed data but performs no enrichment — no CVE correlation, no OS tagging, no endpoint matching, no status progression, no proper naming.

**How to apply:** Fix the data pipeline FIRST (eliminates 10 findings including 5 critical). Then fix UI issues (15 findings). Then accessibility (5 findings). Full report at `.omniprod/reviews/2026-04-04-patches.md`, findings JSON at `.omniprod/findings/2026-04-04-patches.json`.
