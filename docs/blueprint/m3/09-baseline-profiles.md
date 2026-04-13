# Baseline Profiles

**Status**: Planned
**Wave**: 2 — Automation & Extensibility
**Dependencies**: Extended agent collectors, deployment engine, tags

---

## Vision

Declarative desired-state management for endpoints. Define what an endpoint should look like, detect drift, and auto-remediate. This is the differentiator — competitors don't have visual baseline profiles with wave-based auto-remediation.

## Deliverables

### Profile Schema
- [ ] OS requirements: family, distribution, version, minimum patch level
- [ ] Required applications: catalog_id, version constraint (exact/minimum/range/latest/pinned)
- [ ] Denied applications: flag, block, or auto-remove unauthorized software
- [ ] Security requirements: antivirus, firewall, encryption (maps to extended collectors)
- [ ] Configuration requirements: screen lock, USB policy, VPN (maps to compliance rules)

### Profile Creation
- [ ] Capture from golden endpoint: snapshot current state as baseline
- [ ] Manual composition: form-based profile builder
- [ ] Import/clone: copy from another profile, modify as needed

### Profile Assignment
- [ ] Assign to endpoints via tag expressions (e.g., `role:workstation AND os:windows`)
- [ ] One active profile per endpoint (latest assignment wins)
- [ ] Profile inheritance: base profile + override profile

### Drift Detection
- [ ] Agent-side: periodic comparison of inventory vs cached profile
- [ ] Server-side: on every inventory sync, compare against assigned profile
- [ ] Drift events: `baseline.drift_detected` domain event
- [ ] Drift dashboard: per-profile drift percentage, per-endpoint detail

### Remediation Modes
- [ ] Monitor Only: detect and report, no action
- [ ] Notify & Recommend: alert admin with recommended actions
- [ ] Auto-Remediate: create wave deployment to fix drift (install missing, upgrade outdated, remove denied)

### Profile Versioning
- [ ] Archive previous versions
- [ ] Diff display between versions
- [ ] Rollback to previous version
- [ ] Gradual wave rollout on profile update

## License Gating
- Baseline profiles (monitor only): PROFESSIONAL+
- Auto-remediate: ENTERPRISE
- Deny list enforcement: ENTERPRISE
- Profile versioning: ENTERPRISE
