# AI Patch Pipeline v2

**Status**: Planned
**Milestone**: M4
**Dependencies**: M2 Hub catalog pipeline, M3 AI Patch Pipeline v1 (top-50 coverage), MCP server (M2)

---

## Vision

Extend the automated patch pipeline from 50 to 200 applications by using LLM-powered crawlers for semi-structured vendor pages and sandboxed installer analysis to handle unknown binary formats.

## Deliverables

### LLM-Powered Crawler
- [ ] Vendor advisory crawler: LLM extracts version, download URL, release notes from unstructured HTML
- [ ] Structured output schema validated before catalog ingestion
- [ ] Confidence score per extraction (0.0–1.0); low-confidence items queued for human review
- [ ] Rate-limited crawl scheduler per vendor (respect robots.txt, configurable interval)
- [ ] Coverage expansion: 50 → 200 applications (prioritized by install frequency from telemetry)

### AI Installer Analysis
- [ ] Installer type classifier: NSIS, Inno Setup, InstallShield, WiX, MSI, MSIX, generic EXE
- [ ] Silent-install flag inference per installer type (e.g., `/S`, `/quiet`, `/norestart`)
- [ ] Fallback to LLM when classifier confidence < threshold
- [ ] Flag unknown formats for manual review with extracted metadata

### Sandbox Testing
- [ ] Disposable VM provisioning (lightweight VM or container) per installer test run
- [ ] Pre/post state diff: installed files, registry keys, services, version string
- [ ] Version verification: confirm reported version matches installed artifact
- [ ] Rollback test: verify uninstaller removes artifact cleanly
- [ ] Test results attached to catalog entry; failed tests block auto-publish

### Quality Loop
- [ ] Human review queue UI in Hub for flagged items (low confidence, unknown type, test failure)
- [ ] Reviewer approve/reject/edit workflow; approved entries auto-publish
- [ ] Client feedback loop: deployment failure reports feed back to confidence scoring
- [ ] Metrics dashboard: coverage count, confidence distribution, test pass rate, review queue depth

## License Gating

- AI Patch Pipeline v2 (crawler + sandbox): ENTERPRISE
