# Development Compliance

**Status**: Planned
**Milestone**: M4
**Dependencies**: Compliance Engine v2, script library, extended agent collectors

---

## Vision

Ensure development endpoints comply with organizational policies around software licensing, data handling, and regulatory requirements. The agent becomes a governance tool that audits developer workstations for license violations, HIPAA compliance, data anonymization practices, and approved toolchain usage.

## User Value

- **License compliance**: Detect when developers use packages with incompatible licenses (GPL in proprietary projects, expired commercial licenses, unapproved OSS)
- **HIPAA / data handling**: Verify that endpoints handling sensitive data follow anonymization rules, encryption requirements, and access controls
- **Approved toolchain**: Ensure developers use sanctioned IDE versions, compilers, runtimes, and CI tools
- **Audit trail**: Generate evidence for compliance auditors showing continuous monitoring and enforcement
- **Policy enforcement**: Block or alert on policy violations in real-time

## Architecture

### License Auditing

Agent scans:
- Package manager manifests (`package.json`, `go.mod`, `requirements.txt`, `Gemfile`, `pom.xml`)
- Installed development tools and their license terms
- Container images and their layer licenses

Server correlates against a license policy:
```
Allowed: MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC
Restricted (requires approval): LGPL-2.1, LGPL-3.0, MPL-2.0
Prohibited: GPL-2.0, GPL-3.0, AGPL-3.0, SSPL
```

Violations surface in compliance dashboard and trigger `compliance.drift` workflow events.

### Data Handling Checks

Script-based collectors verify:
- Database connections use encrypted channels (TLS)
- Local data stores are encrypted at rest
- No PII in unencrypted files (regex scan for SSN, email, phone patterns)
- VPN active when accessing production data
- Approved anonymization tools installed and configured

### Toolchain Enforcement

Built-in collectors verify:
- IDE version matches approved baseline
- Runtime versions (Node, Python, Go, Java) within supported range
- CI/CD agent is installed and connected
- Git hooks are configured (pre-commit, secrets scanning)
- Approved container registry configured (not Docker Hub public)

## Foundations Built in M2

- **`run_script` command**: Custom compliance checks run as scripts on agent
- **Script library**: Reusable compliance check scripts managed server-side
- **Tag expressions**: Scope development compliance to `role:developer` endpoints
- **Compliance workflow node**: Gates and notifications for violations

## Foundations Built in M3

- **Compliance Engine v2**: Custom frameworks and rule composition
- **Extended collectors**: Application inventory, certificate store scanning

## License Gating

- Development compliance: ENTERPRISE only
- All sub-features (license audit, data handling, toolchain): ENTERPRISE
