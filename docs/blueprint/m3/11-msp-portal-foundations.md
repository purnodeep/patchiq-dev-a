# MSP Portal Foundations

**Status**: Planned
**Wave**: 3 — AI (alongside AI Assistant)
**Moved from**: M4
**Note**: Full MSP features (white-label, billing) remain in M4. This is foundations only.

---

## Vision

Multi-tenant management view in Hub Manager that lets MSPs see and manage all their client Patch Manager instances from a single dashboard. Foundation work that enables the full MSP Portal in M4.

## Deliverables (M3 Scope — Foundations Only)

### Multi-Tenant Management View
- [ ] Hub Dashboard: aggregate view across all connected PMs
- [ ] Per-tenant drill-down: endpoint count, patch compliance, active deployments
- [ ] Tenant health indicators: sync status, agent connectivity, compliance score

### Per-Tenant Policies
- [ ] Create policies at Hub level that propagate to all connected PMs
- [ ] Per-tenant policy overrides (tenant can customize Hub-pushed policies)
- [ ] Policy sync status per tenant

### Cross-Tenant Dashboards
- [ ] Aggregate compliance scores across tenants
- [ ] Cross-tenant CVE exposure (which tenants are affected by CVE-X?)
- [ ] Tenant comparison view (compliance score ranking)

## Out of Scope (Stays in M4)
- White-label support
- Billing metrics collection
- Tenant provisioning workflow
- Full MSP Portal UI with branding

## Dependencies
- Hub→PM sync working reliably (Hub Manager E2E feature)
- PM client registration

## License Gating
- MSP Portal: MSP tier only
