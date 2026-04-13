# Infrastructure Automation

**Status**: Planned
**Milestone**: M4
**Dependencies**: M2 REST API stable, M2 Tag system, M2 Policy engine, M1 agent deployment

---

## Vision

Allow infrastructure teams to manage PatchIQ resources as code — defining endpoints, tags, policies, and deployments in Terraform or Ansible — and integrate PatchIQ into existing GitOps and automation pipelines.

## Deliverables

### Terraform Provider
- [ ] Provider published to Terraform Registry (`hashicorp/patchiq` namespace)
- [ ] Resources: `patchiq_endpoint`, `patchiq_tag`, `patchiq_group`, `patchiq_policy`, `patchiq_deployment_schedule`
- [ ] Data sources: `patchiq_endpoints` (filtered), `patchiq_patch_catalog`, `patchiq_compliance_status`
- [ ] Import support for all resources (adopt existing objects into state)
- [ ] Provider documentation with examples for common IaC patterns

### Ansible Collection
- [ ] Collection published to Ansible Galaxy (`patchiq.patchiq`)
- [ ] Modules: `patchiq_agent` (deploy/remove agent), `patchiq_policy` (apply/remove policy), `patchiq_endpoint_tag` (set tags)
- [ ] Inventory plugin: dynamic Ansible inventory sourced from PatchIQ endpoint list
- [ ] Playbook examples: onboard new server fleet, enforce policy, trigger scan

### GitOps Integration
- [ ] Policy-as-YAML schema: policies defined in YAML files committed to git
- [ ] Sync controller: CI job (GitHub Actions / GitLab CI example workflows) diffs YAML vs API state, applies changes
- [ ] Drift detection: periodic job flags manual changes diverging from declared state
- [ ] PR-based policy change workflow documented with example repo

### Webhook Triggers
- [ ] Outbound webhooks: emit events (deployment complete, compliance breach, new CVE) to external URLs
- [ ] Webhook configuration UI: URL, secret, event filter, retry policy
- [ ] Inbound trigger endpoint: `POST /api/v1/triggers/{name}` starts a named deployment or scan
- [ ] HMAC signature verification for inbound triggers

## License Gating

- Terraform provider: ENTERPRISE
- Ansible collection: ENTERPRISE
- GitOps sync controller: ENTERPRISE
- Webhook triggers (outbound): STANDARD and above
- Webhook triggers (inbound): ENTERPRISE
