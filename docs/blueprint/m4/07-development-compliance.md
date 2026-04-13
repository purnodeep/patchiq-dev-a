# Development Compliance

**Status**: Planned
**Milestone**: M4
**Dependencies**: M2 Compliance Engine v1, M3 Compliance Engine v2, M3 script library, M3 extended collectors

---

## Vision

Extend PatchIQ compliance enforcement into development environments, ensuring developer endpoints meet license policy, data handling, and toolchain requirements alongside standard patch compliance.

## Deliverables

### License Auditing
- [ ] Collector: scan `package.json`, `go.mod`, `requirements.txt`, `pom.xml`, `Cargo.toml` for dependencies
- [ ] SPDX license identifier extraction per dependency (direct and transitive)
- [ ] Results stored per endpoint, correlated to compliance frameworks
- [ ] License inventory report: all unique licenses in use across fleet

### License Policy Engine
- [ ] Policy YAML schema: `allowed`, `restricted`, `prohibited` license lists
- [ ] Evaluation job: compare endpoint scan results against active license policy
- [ ] Restricted license: generates warning finding; prohibited license: generates critical finding
- [ ] Exception workflow: per-package overrides with justification and expiry

### HIPAA Data Handling Controls
- [ ] Collector: verify encrypted connections (TLS 1.2+) enforced in app configs
- [ ] PII detection check: scan code directories for common PII patterns (regex heuristics)
- [ ] Control mapped to HIPAA §164.312 technical safeguards in compliance framework
- [ ] Findings surfaced in existing compliance dashboard

### Toolchain Enforcement
- [ ] Approved IDE versions: VS Code, JetBrains family — version range policy
- [ ] Approved runtime versions: Node.js, Python, Go, Java — version range policy
- [ ] Git hooks check: pre-commit hooks presence (secret scanning, lint)
- [ ] Container registry policy: only approved registries (block Docker Hub, allow internal)
- [ ] Container image license scanning: pull manifest, extract base image, check license

### Operations
- [ ] Dev compliance framework in compliance engine: maps controls to CIS DevSec benchmarks
- [ ] Remediation scripts in script library: install approved runtime, configure git hooks
- [ ] Dashboard filter: "Developer Endpoints" view showing dev-specific compliance posture

## License Gating

- Development Compliance: ENTERPRISE
