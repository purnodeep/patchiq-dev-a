# Feature: Custom RBAC System

> Status: In Progress | Phase: 1 | License: Community (preset), Professional (preset), Enterprise/MSP (full custom)

---

## Overview

A fully customizable role-based access control system where clients create their own roles with granular permissions scoped to specific resources and groups.

## Problem Statement

Regulated industries (healthcare, finance, government) need access controls that map to their organizational structure. Preset Admin/Operator/Viewer roles are insufficient. PatchIQ needs per-resource, per-group, per-action permissions with role inheritance.

---

## Design Principles

- **Fully customizable**: Clients and MSPs create their own roles — not limited to preset Admin/Operator/Viewer
- **Resource-scoped**: Permissions are per-resource-type AND per-resource-group
- **Action-based**: Each permission is a verb on a resource (read, create, update, delete, execute, approve)
- **Inheritable**: Roles can inherit from other roles to avoid duplication

---

## Permission Model

```
Permission = Action + Resource Type + Scope

Examples:
  "endpoints:read:*"              — Read all endpoints
  "endpoints:read:group:production" — Read endpoints in production group only
  "deployments:create:*"          — Create deployments for any group
  "deployments:approve:group:prod" — Approve deployments only for production
  "policies:*:*"                  — Full access to all policies
  "audit:read:*"                  — Read audit logs
  "rbac:manage:*"                 — Manage roles and permissions (admin only)
```

---

## Resource Types

| Resource | Actions |
|----------|---------|
| `endpoints` | read, update, delete, scan, tag |
| `groups` | read, create, update, delete |
| `patches` | read, sync |
| `policies` | read, create, update, delete, evaluate |
| `deployments` | read, create, approve, cancel, retry |
| `reports` | read, create, export |
| `audit` | read |
| `users` | read, create, update, delete |
| `roles` | read, create, update, delete |
| `settings` | read, update |
| `license` | read, update |
| `ai_assistant` | use |

---

## Scope Levels

| Scope | Meaning |
|-------|---------|
| `*` | All resources of this type |
| `group:<name>` | Only resources belonging to this group |
| `tenant:<id>` | Only resources in this tenant (MSP mode) |
| `own` | Only resources created by this user |

---

## Role Templates (Pre-configured, Editable)

| Role | Description |
|------|-------------|
| **Super Admin** | Full access to everything including RBAC management |
| **IT Manager** | Full read, create/approve deployments, manage policies |
| **Operator** | Read endpoints, create deployments (no approve), run scans |
| **Security Analyst** | Read-only + compliance reports + CVE search |
| **Auditor** | Read-only on all resources including audit logs |
| **Help Desk** | Read endpoints, trigger scans, view patch status |
| **MSP Admin** | Full access scoped to their assigned tenants |

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **AI Assistant** | MCP tool calls are filtered by the user's RBAC permissions |
| **Compliance** | Report access is gated by role |
| **Audit Trail** | RBAC changes are logged |
| **Multi-Tenancy** | Tenant scope enables MSP isolation |

---

## License Gating

| Tier | RBAC Capability |
|------|----------------|
| Community | 4 preset roles |
| Professional | 8 preset roles |
| Enterprise | Full custom roles + inheritance |
| MSP | Full custom + tenant-scoped roles |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Auth backend | `internal/server/auth/` |
| RBAC admin UI | `web/src/pages/admin/roles/` |
