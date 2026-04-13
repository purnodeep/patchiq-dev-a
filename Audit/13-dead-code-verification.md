# Audit 13 — Dead Code Verification

Verified 2026-04-09 against branch `dev-a` (commit `7b426cb`).

Method: full-project `grep` for every export name, import path, and string reference. Checked route configs, index re-exports, lazy imports, test files, and CSS references.

---

## Frontend — web/

| # | File (actual path) | Truly Dead? | Evidence | Safe to Delete? |
|---|---|---|---|---|
| 1 | `web/src/pages/PlaceholderPage.tsx` | YES | Defined only. Not imported anywhere. Not in routes.tsx. | YES |
| 2 | `web/src/pages/settings/NotificationSettingsPage.tsx` | YES | Defined only. Not imported anywhere. Not in routes.tsx (notifications route goes to `NotificationsPage`). | YES |
| 3 | `web/src/pages/patches/PatchDeploymentDialog.tsx` | YES | Defined only. No import statements reference it anywhere. | YES |
| 4 | `web/src/pages/patches/PatchExpandedRow.tsx` | YES | Defined only. No import statements reference it anywhere. | YES |
| 5 | `web/src/pages/deployments/CreateDeploymentDialog.tsx` | YES | Defined only. No import statements reference it anywhere. | YES |
| 6 | `web/src/pages/groups/GroupsPage.tsx` | PARTIAL | Not in routes.tsx (no `/groups` route). Imported only in `web/src/__tests__/pages/groups/GroupsPage.test.tsx`. Tests would break but page is unreachable. | YES (delete with its test) |
| 7 | `web/src/pages/alerts/AlertRow.tsx` | YES | Defined only. Not imported by AlertsPage or any other file. | YES |
| 8 | `web/src/pages/workflows/workflow-card.tsx` | YES | Defined only. Not imported by WorkflowsPage or any other file. | YES |
| 9 | `web/src/pages/admin/roles/components/PermissionMatrix.tsx` | YES | Defined only. Not imported by RolesPage, RoleEditPage, or any other file. | YES |
| 10 | `web/src/pages/policies/components/ContextRing.tsx` | YES | Defined only. Referenced in `policies.css` comment but never imported in TSX. | YES (also remove CSS comment) |
| 11 | `web/src/pages/deployments/components/wave-progress.tsx` | YES | Defined only. Not imported by DeploymentsPage or DeploymentDetailPage. | YES |
| 12 | `web/src/pages/endpoints/TagsTab.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 13 | `web/src/dev-mocks.ts` | YES | Defined only. References `devMockPlugin` and `VITE_MOCK=true` in comments but nothing imports this file. No vite plugin references it. | YES |
| 14 | `web/src/components/CVSSVectorBreakdown.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 15 | `web/src/components/SegmentedProgressBar.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 16 | `web/src/components/SlidePanel.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 17 | `web/src/components/data-table/DataTableFilters.tsx` | YES | Exported from `data-table/index.ts` but no page or component imports `DataTableFilters`. | YES (also remove re-export from index.ts) |
| 18 | `web/src/components/data-table/BulkActionBar.tsx` | YES | Exported from `data-table/index.ts` but no page or component imports `BulkActionBar`. | YES (also remove re-export from index.ts) |
| 19 | `web/src/components/data-table/selection-column.tsx` | YES | Exported as `createSelectionColumn` from `data-table/index.ts` but never imported by any consumer. | YES (also remove re-export from index.ts) |
| 20a | `web/src/components/DeploymentModal.tsx` | YES | Duplicate of `web/src/pages/patches/DeploymentModal.tsx`. Zero imports reference the `components/` version. | YES |
| 20b | `web/src/pages/patches/DeploymentModal.tsx` | NO | Imported by `web/src/__tests__/pages/patches/DeploymentModal.test.tsx`. Has active tests. | NO — has test coverage |

---

## Frontend — web-hub/

| # | File (actual path) | Truly Dead? | Evidence | Safe to Delete? |
|---|---|---|---|---|
| 21 | `web-hub/src/pages/PlaceholderPage.tsx` | YES | Defined only. Not imported anywhere. Not in routes.tsx. | YES |
| 22 | `web-hub/src/components/StatsStrip.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 23 | `web-hub/src/components/SeverityPills.tsx` | YES | Defined only. Not imported anywhere. | YES |
| 24 | `web-hub/src/types/settings.ts` | YES | Defines `HubSettings`, `IAMSettings`, `WebhookSettings` types. None are imported anywhere (settings pages define their own inline prop types). `getSetting()` in `api/settings.ts` is also never called (only `getSettings` plural is used via `useSettings` hook). | YES |

---

## Frontend — web-agent/

| # | File (actual path) | Truly Dead? | Evidence | Safe to Delete? |
|---|---|---|---|---|
| 25 | `web-agent/src/app/layout/TabNav.tsx` | YES | Defined only. Not imported anywhere. | YES |

---

## Root-level artifacts

| # | File/Dir | Truly Dead? | Evidence | Safe to Delete? |
|---|---|---|---|---|
| 26 | `agent-overview-snapshot.md` | YES | Only referenced in Audit reports. Not used by any build, code, or docs system. | YES |
| 27 | `hardware-snapshot.md` | YES | Only referenced in Audit reports. Not used by any build, code, or docs system. | YES |
| 28 | `compliance-page.yml` | YES | Only referenced in Audit reports. Not used by any build, code, or docs system. | YES |
| 29 | `policy-detail-snap.md` | YES | Only referenced in Audit reports. Not used by any build, code, or docs system. | YES |
| 30 | `prototype-ui/` | YES | Self-contained prototype directory (HTML mockups, WhatsApp screenshots, separate package-lock.json). Not imported or referenced by any production code. Only referenced in Audit reports. | YES |

---

## Go dead code

| # | File/Dir | Truly Dead? | Evidence | Safe to Delete? |
|---|---|---|---|---|
| 31 | `internal/server/module.go` | YES | Defines `Module` interface, `EventHandler` type, `ModuleDeps` struct. None are referenced outside this file. No code uses `server.Module`, `server.EventHandler`, or `server.ModuleDeps`. Note: `domain.EventHandler` (in shared) is a separate, actively-used type. | YES |
| 32 | `internal/agent/patcher/download.go` | NO | File is at `internal/agent/patcher/download.go` (not `internal/agent/downloader.go`). `Downloader` type has no production callers, BUT it has 4 passing tests in `download_test.go` and is a legitimate patcher module needed for agent patch installation. It will be wired up when the agent patcher flows are connected E2E. | RISKY — keep for now |
| 33 | `internal/server/engine/` | YES | Empty directory with only `.gitkeep`. No Go files, no references. | YES |
| 34 | `internal/server/apm/` | YES | Empty directory with only `.gitkeep`. No Go files, no references. | YES |
| 35 | `internal/server/mcp/` | PARTIAL | Empty directory with only `.gitkeep`. No Go files. However, CLAUDE.md lists `mcp/` as "Model Context Protocol (AI integration)" under Server Services, suggesting planned use. | YES to delete, but re-create when needed |

---

## Unused sqlc queries (sample of 10)

Checked 20 function names total; found these 10 with zero callers outside `sqlcgen/` (excluding test files):

| sqlc Function | File | Callers Outside sqlcgen? | Truly Unused? |
|---|---|---|---|
| `CreateApprovalRequest` | `server/store/sqlcgen/deployments.sql.go` | None | YES |
| `CountRoleUsers` | `server/store/sqlcgen/roles.sql.go` | None | YES |
| `GetBinaryFetchState` | `hub/store/sqlcgen/*.sql.go` | None | YES |
| `GetClientSyncHistory` | `hub/store/sqlcgen/*.sql.go` | None | YES |
| `GetDeploymentScheduleByDeployment` | `server/store/sqlcgen/deployment_schedules.sql.go` | None | YES |
| `GetEndpointByAgentID` | `server/store/sqlcgen/endpoints.sql.go` | None | YES |
| `GetGroupEndpointCount` | `server/store/sqlcgen/groups.sql.go` | None | YES |
| `GetPatchByKBArticle` | `server/store/sqlcgen/patches.sql.go` | None | YES |
| `ListEndpointsByGroupID` | `server/store/sqlcgen/endpoints.sql.go` | None | YES |
| `ListPatchesByEndpoint` | `server/store/sqlcgen/patches.sql.go` | None | YES |

**Note on sqlc queries**: These are generated code. Do NOT delete the `.go` files directly. Instead, remove or comment out the corresponding SQL in `internal/{server,hub}/store/queries/*.sql` and run `make sqlc` to regenerate. However, some of these queries (e.g., `GetEndpointByAgentID`, `GetPatchByKBArticle`) may be useful for future features. Consider keeping the SQL definitions and documenting them as "available but unwired" rather than deleting.

Also found with zero callers but deferred from sample:
- `SoftDeleteTagRule`
- `UpdateDeploymentTargetResult`
- `UpdateEndpointLastScan`

---

## Summary

| Category | Total Checked | Truly Dead | Safe to Delete |
|---|---|---|---|
| web/ frontend files | 20 | 19 (one `DeploymentModal` in pages/ has tests) | 19 |
| web-hub/ frontend files | 4 | 4 | 4 |
| web-agent/ frontend files | 1 | 1 | 1 |
| Root-level artifacts | 5 | 5 | 5 |
| Go dead code | 5 | 4 (download.go has tests + future use) | 3 definitely, 1 with caution |
| Unused sqlc queries (sample) | 10 | 10 | 10 (via query SQL removal + regenerate) |

**Total confirmed dead: 43 out of 45 items checked.**

### Cleanup notes

When deleting web/ files, also clean up:
- `web/src/components/data-table/index.ts` — remove 3 re-exports (DataTableFilters, BulkActionBar, createSelectionColumn)
- `web/src/pages/policies/policies.css` — remove ContextRing CSS comment/animation
- `web/src/__tests__/pages/groups/` — delete entire test directory (tests GroupsPage + related dialogs)
- `web/src/pages/groups/` — delete entire directory (GroupsPage + EditGroupDialog + CreateGroupDialog)
