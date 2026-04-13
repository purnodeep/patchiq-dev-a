# ADR-012: Zitadel for Identity and Access Management

## Status

Accepted

## Context

PatchIQ needs enterprise IAM: SAML/OIDC SSO, LDAP/AD sync, multi-tenancy, and audit trail. Building custom auth from scratch would take 3-5 months and produce an inferior result. We evaluated Keycloak, Zitadel, Ory Stack, Casdoor, and Authentik.

Key requirements:
- Self-hosted / air-gapped deployment (on-prem Patch Manager)
- Native multi-tenancy (MSP portal in Hub Manager)
- SAML + OIDC + LDAP/AD protocols
- Audit trail for compliance (HIPAA, SOC2)
- Go-native integration (matches our backend stack)

## Decision

Use Zitadel as the IAM platform for all PatchIQ deployments. Zitadel handles: users, organizations, SSO (SAML/OIDC), LDAP/AD sync, coarse-grained roles (admin/operator/viewer), and MFA. PatchIQ application handles fine-grained domain-specific permissions via the custom RBAC model (see ADR-004).

**RBAC split:**
- **Zitadel manages**: User identity, authentication, organization membership, coarse role assignment, session management
- **PatchIQ app manages**: Fine-grained permissions (Action + Resource + Scope model, e.g., `deployments:approve:group:prod`), policy-based access decisions, resource-level scoping

## Consequences

- **Positive**: Go-native (single binary, same language as PatchIQ); Apache 2.0 license; native multi-tenancy maps directly to MSP portal; built-in event-sourced audit trail (every mutation is an immutable event); SOC 2 Type II certified; official Go SDK (zitadel-go); single binary simplifies on-prem and air-gapped deployment
- **Negative**: Smaller ecosystem than Keycloak; fewer enterprise references; requires PostgreSQL (or CockroachDB) as backend — not a problem since we already use PostgreSQL; team must learn Zitadel's administration model

## Alternatives Considered

- **Keycloak**: Most mature, largest ecosystem — rejected because Java runtime adds 512MB-1GB RAM overhead; realm-based multi-tenancy is less elegant than Zitadel's native organizations; not Go-native
- **Ory Stack (Kratos + Hydra + Keto)**: All Go, fine-grained — rejected because 3 separate services is heavy for on-prem; more operational complexity than Zitadel's single binary
- **Casdoor**: Go + React, growing fast — rejected because younger project, less proven in air-gapped enterprise deployments
- **Authentik**: Comprehensive — rejected because Python-based; adds language diversity to a Go-native stack
- **Custom auth**: Full control — rejected because 3-5 months of effort to build what Zitadel provides out of the box
