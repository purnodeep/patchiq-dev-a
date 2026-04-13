# MSP Portal (Full)

**Status**: Planned
**Milestone**: M4
**Dependencies**: M3 MSP foundations (multi-tenant view, per-tenant policies, cross-tenant dashboards)

---

## Vision

Complete the MSP experience by adding white-label branding, billing metrics, and a guided tenant provisioning workflow, turning PatchIQ into a fully resellable managed service platform.

## Deliverables

### White-Label Support
- [ ] Per-MSP branding configuration: logo, primary color, favicon, login page headline
- [ ] Custom domain support for tenant-facing UI (CNAME → Hub subdomain)
- [ ] Email notification templates with MSP branding (logo, footer, reply-to)
- [ ] Branding assets stored in MinIO, served via Hub CDN endpoint

### Billing Metrics
- [ ] Per-tenant usage metrics: managed endpoint count, active deployments, API call volume
- [ ] Billing period snapshots (monthly, exportable as CSV/JSON)
- [ ] Usage dashboard in Hub MSP view with per-tenant breakdown
- [ ] Webhook export to billing systems (Stripe, external ERP) on period close

### Tenant Provisioning Workflow
- [ ] Guided wizard: tenant name → tier selection → admin user invite → initial policy template
- [ ] Automated DB schema provisioning (tenant isolation enforced at creation)
- [ ] Welcome email to tenant admin with onboarding link
- [ ] Tenant lifecycle management: suspend, reactivate, archive
- [ ] Bulk tenant import via CSV for MSP onboarding large client lists

### MSP Dashboard Enhancements
- [ ] Cross-tenant compliance score trend (rolling 90 days)
- [ ] SLA breach alerts: tenants below compliance threshold
- [ ] Top CVEs affecting MSP-managed fleet (aggregated, de-duplicated)
- [ ] Per-tenant health score card (exportable as PDF report)

## License Gating

- MSP Portal (full): MSP tier
- White-label branding: MSP tier
- Billing metrics export: MSP tier
