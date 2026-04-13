# ADR-004: Custom RBAC with Action + Resource + Scope Model

## Status

Accepted (Updated — RBAC split with Zitadel per ADR-012)

## Context

Regulated industries need access controls that map to their organizational structure. Preset role systems are insufficient for healthcare, finance, and government customers.

With the adoption of Zitadel for IAM (ADR-012), authentication and coarse-grained role management are handled externally. The custom RBAC system focuses on fine-grained, domain-specific authorization that Zitadel cannot model.

## Decision

Implement a split RBAC model:

- **Zitadel manages**: User identity, authentication, organization membership, coarse roles (admin/operator/viewer), session management, MFA
- **PatchIQ app manages**: Fine-grained permissions defined as Action + Resource Type + Scope (e.g., `deployments:approve:group:prod`). Roles can inherit from other roles.

The PatchIQ app reads the user's coarse role from Zitadel (via JWT claims) and applies fine-grained permission checks internally. Fine-grained permissions are stored in PostgreSQL alongside the application data.

## Consequences

- **Positive**: Zitadel handles the hard parts of IAM (SSO, LDAP, MFA, session management); PatchIQ's custom RBAC focuses only on domain-specific authorization; coarse roles provide a sane default for simple deployments; fine-grained permissions serve regulated industries; role inheritance reduces duplication; MSP tenant scoping through Zitadel organizations
- **Negative**: Two systems involved in authorization decisions; must keep coarse roles in sync between Zitadel and app; potential confusion about where a permission is defined; performance impact on fine-grained checks (mitigated by caching)

## Alternatives Considered

- **Zitadel handles ALL RBAC**: Push all permissions into Zitadel — rejected because Zitadel's permission model cannot cleanly express Action+Resource+Scope for domain-specific resources
- **Zitadel handles auth only, app handles ALL RBAC**: Ignore Zitadel's role system — rejected because rebuilds coarse role management that Zitadel already provides
- **Preset roles only** (Admin/Operator/Viewer): Simple but inflexible — rejected because regulated industries need granular control
- **Casbin/OPA policy engine**: External policy engine — rejected because adds operational complexity; our model is simpler for the use case
