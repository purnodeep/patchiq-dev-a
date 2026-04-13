# Reporting Engine for Patch Manager
**Date:** 2026-04-10
**Status:** Design — pending plan
**Track:** Standard

---

## Problem

PatchIQ has rich data across endpoints, patches, CVEs, deployments, and compliance — but no way to export it as structured, professional reports. The only exports today are bare CSV dumps for endpoints (`/endpoints/export`) and audit logs (`/audit/export`). There are no PDF reports, no XLSX exports, no scheduled generation, and no email delivery.

For the client POC deployment, this is a critical gap:
- **Compliance auditors** need formal evidence documents with framework scores, control results, and remediation status.
- **Security teams** need CVE exposure reports with CVSS scores, exploit status, and affected endpoint counts.
- **Operations teams** need deployment reports showing success/failure rates and error details.
- **Executives** need a one-page security posture summary.

Every competing product (SCCM, Ivanti, Automox, Tanium, ManageEngine, BigFix, NinjaOne, ConnectWise) ships scheduled PDF/CSV reports with email delivery. This is table stakes.

---

## Goals

- Generate rich, professionally structured reports as PDF, CSV, and XLSX.
- Support 6 report types: Endpoints, Patches, CVEs, Deployments, Compliance (per framework), Executive Summary.
- Every report has: header, summary stats, charts, breakdowns, highlights, detailed data table, footer with page numbers.
- Reports are downloadable via API (user opens in their own browser/viewer).
- Store generated reports in MinIO with SHA-256 checksum for tamper evidence.
- Respect tenant isolation (data + stored files).
- RBAC-gated: `reports:read`, `reports:create`, `reports:export`.
- All timestamps rendered in IST (Asia/Kolkata, UTC+5:30).
- Pure Go — no external binaries (no Chrome, no wkhtmltopdf).

## Non-Goals

- In-app PDF viewer — user downloads and opens externally.
- Scheduled report generation — deferred to Phase 2.
- Email delivery of reports — deferred to Phase 2 (infrastructure ready via Shoutrrr).
- Custom report builder (user-defined columns/filters) — deferred to Phase 2.
- Historical trend analysis beyond what compliance already computes — deferred.
- Report white-labeling/branding — future.
- Configurable timezone per tenant — hardcode IST for now.

---

## Design

### Architecture

```
internal/server/
├── reports/                        <- NEW package
│   ├── service.go                  <- Orchestration: fetch data -> render -> store
│   ├── types.go                    <- ReportData, StatBox, ChartSpec, BreakdownTable, etc.
│   ├── renderer_pdf.go             <- maroto/v2: shared layout (header, footer, stat boxes, tables)
│   ├── renderer_csv.go             <- encoding/csv: detail table as flat rows
│   ├── renderer_xlsx.go            <- excelize/v2: summary sheet + detail sheet
│   ├── charts.go                   <- vicanso/go-charts -> PNG bytes for embedding
│   ├── endpoints_report.go         <- Fetch endpoint data, assemble ReportData
│   ├── patches_report.go           <- Fetch patch data, assemble ReportData
│   ├── cves_report.go              <- Fetch CVE data, assemble ReportData
│   ├── deployments_report.go       <- Fetch deployment data, assemble ReportData
│   ├── compliance_report.go        <- Fetch compliance data, assemble ReportData
│   └── executive_report.go         <- Aggregate all areas into one-page summary
├── api/v1/
│   └── reports.go                  <- NEW handler: generate, list history, download
├── store/
│   ├── queries/reports.sql         <- NEW: report_generations history table
│   └── migrations/046_reports.sql  <- NEW: report_generations table
```

### New Dependencies

| Library | Version | License | Purpose |
|---------|---------|---------|---------|
| `github.com/johnfercher/maroto/v2` | latest | MIT | PDF generation — report-focused builder with tables, headers, footers, page numbers |
| `github.com/vicanso/go-charts` | latest | MIT | Pure Go chart rendering to PNG (bar, pie, line, gauge) |
| `github.com/xuri/excelize/v2` | latest | BSD-3 | XLSX generation with rich formatting, charts, auto-filters |

All pure Go, no CGo, no external binaries. All commercially permissive licenses.

### Data Model

```sql
-- migration: 046_reports.sql
CREATE TABLE report_generations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    report_type     TEXT NOT NULL,          -- endpoints, patches, cves, deployments, compliance, executive
    format          TEXT NOT NULL,          -- pdf, csv, xlsx
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending, generating, completed, failed
    filters         JSONB NOT NULL DEFAULT '{}',      -- {severity, os_family, date_from, date_to, framework_id, ...}
    file_path       TEXT,                  -- MinIO object key (set on completion)
    file_size_bytes BIGINT,               -- file size
    checksum_sha256 TEXT,                  -- tamper evidence
    row_count       INT,                  -- number of data rows in report
    error_message   TEXT,                  -- set on failure
    created_by      UUID NOT NULL,         -- user who triggered generation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,  -- retention: created_at + 90 days
    CONSTRAINT valid_report_type CHECK (report_type IN ('endpoints', 'patches', 'cves', 'deployments', 'compliance', 'executive')),
    CONSTRAINT valid_format CHECK (format IN ('pdf', 'csv', 'xlsx')),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'generating', 'completed', 'failed'))
);

CREATE INDEX idx_report_generations_tenant ON report_generations(tenant_id);
CREATE INDEX idx_report_generations_expires ON report_generations(expires_at) WHERE status = 'completed';

-- RLS
ALTER TABLE report_generations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON report_generations
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

MinIO storage path: `reports/{tenant_id}/{report_type}/{id}.{format}`

### Report Data Abstraction

Every report type assembles the same `ReportData` structure, then passes it to format-specific renderers:

```go
type ReportType string
const (
    ReportEndpoints   ReportType = "endpoints"
    ReportPatches     ReportType = "patches"
    ReportCVEs        ReportType = "cves"
    ReportDeployments ReportType = "deployments"
    ReportCompliance  ReportType = "compliance"
    ReportExecutive   ReportType = "executive"
)

type ReportMeta struct {
    Title       string
    Subtitle    string       // e.g., "PCI DSS v4.0" for compliance
    TenantName  string
    DateFrom    time.Time    // filter range start (or report generation time if no range)
    DateTo      time.Time
    GeneratedAt time.Time    // always IST
    GeneratedBy string       // user name/email
}

type StatBox struct {
    Label string
    Value string             // "247" or "87.3%"
    Trend string             // "up", "down", "flat", ""
    Color string             // "green", "red", "orange", "blue", "gray"
}

type ChartSpec struct {
    Title    string
    Type     string          // "pie", "bar", "horizontal_bar", "line", "gauge"
    Data     []ChartDataPoint
    Width    int             // pixels
    Height   int             // pixels
}

type ChartDataPoint struct {
    Label string
    Value float64
    Color string             // hex color
}

type BreakdownTable struct {
    Title   string           // e.g., "By OS Family"
    Columns []string
    Rows    [][]string
    RowColors []string       // optional per-row background color
}

type HighlightSection struct {
    Title       string       // "Immediate Action Required"
    Description string
    Columns     []string
    Rows        [][]string
    RowColors   []string
}

type DetailTable struct {
    Columns     []string
    Rows        [][]string
    RowColors   []string     // severity-based: red, orange, yellow, green
    MaxPDFRows  int          // cap for PDF (500), unlimited for CSV/XLSX
    TotalRows   int          // actual count (shown as "Showing 500 of 2,341")
}

type ReportData struct {
    Meta        ReportMeta
    Summary     []StatBox
    Charts      []ChartSpec
    Breakdowns  []BreakdownTable
    Highlights  []HighlightSection
    Detail      DetailTable
}
```

### Renderer Interface

```go
type Renderer interface {
    Render(data *ReportData) ([]byte, error)
    ContentType() string
    FileExtension() string
}
```

Three implementations: `PDFRenderer`, `CSVRenderer`, `XLSXRenderer`.

- **PDFRenderer** (maroto/v2): Full report — header, stat boxes, charts (PNG embedded), breakdown tables, highlights, detail table with colored rows, footer with page numbers. Detail table capped at 500 rows with note "Showing 500 of N — download CSV/XLSX for full data."
- **CSVRenderer** (encoding/csv): Detail table only — all rows, all columns. No summary/charts.
- **XLSXRenderer** (excelize/v2): Sheet 1 "Summary" — stat boxes + breakdown tables. Sheet 2 "Data" — full detail table with auto-filters, freeze panes, conditional formatting on severity columns.

### API Endpoints

```
POST   /api/v1/reports/generate          <- Trigger on-demand generation
GET    /api/v1/reports                    <- List report history (paginated)
GET    /api/v1/reports/{id}               <- Get report metadata + status
GET    /api/v1/reports/{id}/download      <- Download generated file (redirect to MinIO presigned URL or stream)
DELETE /api/v1/reports/{id}               <- Delete report + MinIO object
```

#### POST /api/v1/reports/generate

```json
{
    "report_type": "endpoints",
    "format": "pdf",
    "filters": {
        "status": "online",
        "os_family": "linux",
        "severity": "critical",
        "date_from": "2026-03-01T00:00:00+05:30",
        "date_to": "2026-04-10T23:59:59+05:30",
        "framework_id": "pci-dss-v4",
        "tag_id": "uuid-here"
    }
}
```

Response: `201 Created` with report generation record (status: "generating").

Generation happens synchronously for small reports (<5s), asynchronously via River for large ones. The API returns immediately with a `pending` or `completed` status. Frontend polls until `completed`.

#### GET /api/v1/reports/{id}/download

Returns the file with appropriate `Content-Type` and `Content-Disposition` headers. Verifies tenant ownership before serving. Options:
- **Option A**: Stream directly from MinIO through the API.
- **Option B**: Generate a MinIO presigned URL (5-minute expiry) and 302 redirect.

**Decision: Option A** (stream through API) — simpler, no MinIO URL exposure, RBAC enforced at download time.

### Report Content Per Type

#### Endpoints Report
- **Summary**: Total endpoints, online %, compliance %, critical CVE count, pending patches
- **Charts**: OS distribution pie, status distribution donut, risk score histogram, compliance by OS bar
- **Breakdowns**: By OS Family (count, online, CVEs, patches, compliance %), By Status
- **Highlights**: Top risk endpoints (risk_score >= 7) with hostname, OS, critical CVEs, pending patches
- **Detail**: All endpoints — hostname, OS, version, status, agent version, IP, risk score, CVE counts (crit/high/med), pending patches, compliance %, tags, last seen
- **Row colors**: Red (risk >= 7), orange (4-6), green (0-3)

#### Patches Report
- **Summary**: Total patches, critical count, high count, average remediation %, patches with 0% remediation
- **Charts**: Severity distribution pie, remediation % by severity (grouped bar), top 10 by affected endpoints (horizontal bar), patch aging (days since release)
- **Breakdowns**: By severity, by OS family, by status (available/superseded/withdrawn)
- **Highlights**: Patches with remediation_pct < 50% AND severity critical/high
- **Detail**: All patches — name, version, severity, OS, CVE count, highest CVSS, affected endpoints, deployed count, remediation %, released date, status
- **Row colors**: Based on severity

#### CVEs Report
- **Summary**: Total CVEs, critical count, CISA KEV count, exploit available count, CVEs without patches
- **Charts**: Severity distribution pie, CVSS histogram, attack vector breakdown bar, top 10 by affected endpoints
- **Breakdowns**: By severity, by attack vector, by remediation status (affected/patched/mitigated/ignored)
- **Highlights**: CISA KEV CVEs with due dates + CVEs with known exploits — sorted by CVSS desc
- **Detail**: All CVEs — ID, CVSS, severity, attack vector, exploit status, CISA KEV due date, patch available, patch count, affected endpoints, published date
- **Row colors**: Red (CVSS >= 9), orange (7-8.9), yellow (4-6.9), green (< 4)

#### Deployments Report
- **Summary**: Total deployments, success rate %, failed count, average duration, pending count
- **Charts**: Status distribution pie, deployment timeline (line chart), failure reasons breakdown
- **Breakdowns**: By status, by triggered_by user
- **Highlights**: Failed deployments with error details, targets, and affected endpoints
- **Detail**: All deployments — name, status, total targets, success/failed/pending counts, triggered by, started, completed, duration
- **Row colors**: Red (failed), orange (partial), green (succeeded), gray (pending)

#### Compliance Report (per framework)
- **Summary**: Framework name + version, overall score, controls passing/total, endpoints evaluated, overdue count
- **Charts**: Score gauge, control status pie (pass/fail/partial/NA), score trend line (last 90 days), non-compliant by category bar
- **Breakdowns**: Control results by category (control ID, name, status, passing/total endpoints, SLA, days overdue)
- **Highlights**: Overdue controls with remediation hints
- **Detail**: Non-compliant endpoints — hostname, OS, score, CVE breakdown (compliant/at-risk/non-compliant/late)
- **Row colors**: Based on score (red < 50%, orange < 75%, yellow < 90%, green >= 90%)

#### Executive Summary
- **Summary**: Overall compliance %, total endpoints (online %), critical CVEs unpatched, deployment success rate, top risk score
- **Charts**: Compliance by framework (horizontal bar), CVE severity distribution, deployment trend (7-day), endpoint status donut
- **Breakdowns**: Framework compliance table (name, score, status), top 5 risks (CVE + affected count)
- **Highlights**: Items requiring executive attention — CISA KEV overdue, critical patches < 25% remediated, compliance frameworks below threshold
- **Detail**: None (executive summary is charts and summaries only)

### Generation Flow

```
1. API handler validates request, creates report_generations row (status: pending)
2. Calls reports.Service.Generate(ctx, reportGenID)
3. Service updates status to "generating"
4. Service calls type-specific assembler (e.g., EndpointsReportAssembler)
   a. Assembler runs store queries with filters
   b. Assembler builds ReportData struct
5. Service calls renderer (PDF/CSV/XLSX)
   a. PDF renderer: renders charts to PNG via go-charts, builds PDF via maroto
   b. CSV renderer: writes detail table rows
   c. XLSX renderer: creates summary + detail sheets
6. Service uploads bytes to MinIO
7. Service computes SHA-256 checksum
8. Service updates report_generations row: status=completed, file_path, file_size, checksum, completed_at
9. Service emits domain event: report.generated
```

On failure at any step: status=failed, error_message set.

### RBAC

Add to existing permission system:

| Route | Permission |
|-------|-----------|
| POST /reports/generate | `reports:create` |
| GET /reports | `reports:read` |
| GET /reports/{id} | `reports:read` |
| GET /reports/{id}/download | `reports:export` |
| DELETE /reports/{id} | `reports:delete` |

### Retention

- `expires_at` set to `created_at + 90 days` on every report.
- River periodic job (daily): delete expired reports from MinIO + DB.
- Pattern: same as `AuditRetentionJob`.

### Frontend

```
web/src/
├── pages/reports/
│   ├── index.tsx                    <- Reports page (history table + generate)
│   ├── columns.tsx                  <- Report history table columns
│   └── GenerateReportDialog.tsx     <- Shared dialog (used here AND on resource pages)
├── api/hooks/
│   └── useReports.ts               <- generate, list, download, delete hooks
```

#### Reports Page (`/reports`) — Follows Existing List Page Pattern

The page matches the established PatchIQ list page layout: stat cards → filter bar → data table → pagination.

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│  Reports                                          [Generate ▼]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐        │
│  │  All   │ │Completed│ │Generating│ │ Failed │ │ Today  │       │
│  │  142   │ │  138    │ │    1    │ │   3    │ │   4    │        │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘        │
│  (clickable stat cards that filter the table — same as           │
│   deployments page pattern)                                      │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 🔍 Search reports...  │ Type ▾ │ Format ▾ │ Date range  │   │
│  └──────────────────────────────────────────────────────────┘   │
│  (filter bar — bg: var(--bg-card), border, 10px 14px padding)   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ REPORT NAME        TYPE       FORMAT  STATUS   GENERATED │   │
│  │──────────────────────────────────────────────────────────│   │
│  │ Endpoints Report   endpoints  PDF     ● Done   10 Apr    │   │
│  │ CVE Exposure        cves      XLSX    ● Done   10 Apr    │   │
│  │ PCI DSS Compliance  compliance PDF    ● Done    9 Apr    │   │
│  │ Patch Remediation   patches   CSV     ● Done    9 Apr    │   │
│  │ Weekly Executive    executive  PDF    ◌ Gen...   now      │   │
│  │──────────────────────────────────────────────────────────│   │
│  │                              ◀ Previous │ Next ▶         │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

**Stat Cards Row:**
- `All` (total count, var(--text-emphasis))
- `Completed` (count, var(--signal-healthy))
- `Generating` (count, var(--accent), pulse animation)
- `Failed` (count, var(--signal-critical))
- `Today` (count generated today, var(--text-secondary))
- Clicking a card filters the table (same active state pattern as deployments)

**Filter Bar:**
- Search input (searches report name — same bg-inset + border pattern)
- Type dropdown: All | Endpoints | Patches | CVEs | Deployments | Compliance | Executive
- Format dropdown: All | PDF | CSV | XLSX
- Date range: from/to date inputs (native HTML date, same styling as audit)
- Clear filters button (shown when filters active)

**Table Columns:**

| Column | Field | Rendering |
|--------|-------|-----------|
| Report Name | `name` | Auto-generated: "{Type} Report — {filters summary}". Monospace, bold. |
| Type | `report_type` | MonoTag badge (e.g., `ENDPOINTS`, `CVES`) |
| Format | `format` | MonoTag badge with icon (PDF icon / CSV icon / XLSX icon) |
| Filters | `filters` | Small muted text summarizing active filters ("severity: critical, os: linux") or "No filters" |
| Status | `status` | StatusBadge: completed (green dot), generating (blue dot, pulse), failed (red dot), pending (gray dot) |
| Size | `file_size_bytes` | Formatted: "2.4 MB", "340 KB". Shown only when completed. |
| Generated | `created_at` | Relative time ("5 min ago", "2 days ago") with full IST timestamp on hover |
| Generated By | `created_by` | User name/email |
| Actions | — | Download button (enabled when completed), Delete button (dropdown menu) |

**Row Behavior:**
- Completed rows: download icon button on right (clicking triggers file download)
- Generating rows: subtle pulse animation on status dot, row has `var(--accent)` left border
- Failed rows: red left border (`borderLeft: 2px solid var(--signal-critical)`), error message shown in expanded row or tooltip
- Hover: `var(--bg-card-hover)` background

**Pagination:** Cursor-based, same `Previous / Next` pattern as all other pages.

**Empty State:**
```
┌──────────────────────────────────────┐
│         📄 No reports yet            │
│                                      │
│   Generate your first report to      │
│   see it here.                       │
│                                      │
│       [Generate Report]              │
└──────────────────────────────────────┘
```

**Error State:** Standard `<ErrorState title="Failed to load reports" onRetry={refetch} />`.

**Loading State:** 7 skeleton rows (`<Skeleton className="h-11 rounded-lg" />`).

#### Generate Report Dialog (Shared Component)

Used in two places:
1. "Generate Report" button on the `/reports` page
2. "Export Report" button on each resource page header (pre-fills type + current filters)

```
┌─────────────────────────────────────────────┐
│  Generate Report                        [X] │
├─────────────────────────────────────────────┤
│                                             │
│  Report Type                                │
│  ┌─────────────────────────────────────┐    │
│  │ Endpoints                         ▾ │    │
│  └─────────────────────────────────────┘    │
│                                             │
│  Format                                     │
│  ┌─────┐ ┌─────┐ ┌──────┐                  │
│  │ PDF │ │ CSV │ │ XLSX │   (toggle group)  │
│  └─────┘ └─────┘ └──────┘                  │
│                                             │
│  Filters (optional)                         │
│  ┌──────────────┐ ┌──────────────┐          │
│  │ Severity   ▾ │ │ OS Family  ▾ │          │
│  └──────────────┘ └──────────────┘          │
│  ┌──────────────┐ ┌──────────────┐          │
│  │ Date from    │ │ Date to      │          │
│  └──────────────┘ └──────────────┘          │
│  ┌──────────────┐                           │
│  │ Status     ▾ │  (type-specific filters)  │
│  └──────────────┘                           │
│                                             │
│  (For compliance type: framework picker)    │
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │ Active filters: severity=critical,  │    │
│  │ os_family=linux                     │    │
│  └─────────────────────────────────────┘    │
│  (shown when opened from resource page      │
│   with pre-filled filters)                  │
│                                             │
├─────────────────────────────────────────────┤
│              [Cancel]  [Generate Report]    │
└─────────────────────────────────────────────┘
```

**Dialog behavior:**
- Report Type dropdown shows all 6 types. When opened from a resource page, pre-selected and optionally locked.
- Format toggle: PDF selected by default. Toggle between PDF / CSV / XLSX.
- Filters section shows type-appropriate filters:
  - Endpoints: status, os_family, tag
  - Patches: severity, os_family, status
  - CVEs: severity, exploit_available, cisa_kev, attack_vector, has_patch
  - Deployments: status, date range
  - Compliance: framework picker (required)
  - Executive: date range only
- "Generate Report" button: POST to API, close dialog, show toast ("Report generating..."), navigate to `/reports` page (or stay if already there).
- Uses `@patchiq/ui` Dialog, Button, Select components.

#### Per-Resource Page Integration

Each resource page gets an "Export Report" button in its header/filter bar area (rightmost position, same as existing action buttons):

| Page | Button Location | Pre-filled |
|------|----------------|------------|
| `/endpoints` | Filter bar, right side (next to existing export button) | type=endpoints, inherits current status/os/tag/search filters |
| `/patches` | Filter bar, right side | type=patches, inherits severity/os/status filters |
| `/cves` | Filter bar, right side | type=cves, inherits severity/exploit/kev filters |
| `/deployments` | Filter bar, right side | type=deployments, inherits status/date filters |
| `/compliance/frameworks/{id}` | Framework detail page header | type=compliance, framework_id pre-filled |
| `/dashboard` | Dashboard header area | type=executive |

The button opens `GenerateReportDialog` with `defaultType` and `defaultFilters` props. Existing CSV export buttons on endpoints and audit pages remain unchanged.

#### Sidebar Navigation

Add under existing nav group (after Compliance, before Settings):

```typescript
// In AppSidebar.tsx navGroups:
{
  label: 'Analytics',  // or place under existing 'Compliance' group
  items: [
    { label: 'Compliance', icon: ShieldCheck, to: '/compliance', resource: 'compliance', action: 'read' },
    { label: 'Reports', icon: FileBarChart, to: '/reports', resource: 'reports', action: 'read' },
    { label: 'Audit Log', icon: ScrollText, to: '/audit', resource: 'audit', action: 'read' },
  ],
}
```

Icon: `FileBarChart` from lucide-react (or `FileText` — matches the report concept).

#### Routes

```typescript
// In routes.tsx:
{ path: '/reports', element: <RequirePermission resource="reports" action="read"><ReportsPage /></RequirePermission> },
```

#### API Hooks (`useReports.ts`)

```typescript
// List report history
useReports(params?: { cursor, limit, status, report_type, format, date_from, date_to })

// Generate new report
useGenerateReport() -> useMutation

// Download report file
useDownloadReport(id: string) -> triggers browser download

// Delete report
useDeleteReport() -> useMutation
```

Follows existing hook patterns: `useQuery` with 30s refetch interval for list, `useMutation` for generate/delete. Download uses `fetch` + `blob` + `URL.createObjectURL` pattern (same as existing endpoint export).

---

## Approach Comparison

### Approach A: Per-Resource Export Endpoints (Inline)
Add `/export` to each existing handler (endpoints, patches, cves, deployments, compliance). Generate and stream directly in the HTTP response. No storage, no history.

**Pros:** Simple, no new tables, no MinIO, follows existing audit/endpoint export pattern.
**Cons:** No generation history, no checksums, no async for large reports, PDF generation blocks the HTTP request, no centralized report management page, duplicated rendering code across 6 handlers.

### Approach B: Centralized Report Service (Recommended)
Single `/api/v1/reports/generate` endpoint. Dedicated `reports/` package. Store in MinIO with history.

**Pros:** Single report management page, generation history with checksums, async-ready (River), centralized rendering (DRY), retention policy, audit trail via domain events, easy to add scheduled delivery later.
**Cons:** More upfront work, new migration, MinIO dependency for reports.

### Approach C: Hybrid
Keep existing CSV exports on resources (already working for endpoints/audit). Add centralized service only for PDF/XLSX.

**Pros:** Doesn't break existing exports, incremental.
**Cons:** Two different export paths, confusing UX, still need most of Approach B's infrastructure for PDF.

**Decision: Approach B** — centralized service. The existing CSV exports on endpoints and audit stay as-is (they're lightweight and already work), but the new reporting feature is a unified system. The marginal extra effort pays off in maintainability, auditability, and Phase 2 readiness (scheduling, email delivery).

---

## Phase 2 Roadmap (Post-POC, not in scope now)

1. **Scheduled Reports** — Add `report_schedules` table (cron expression, type, filters, recipients). River periodic job checks schedules, generates reports, delivers via Shoutrrr. Infrastructure is ready (River + Shoutrrr + MinIO).
2. **Email Delivery** — Attach report download link (MinIO presigned URL with 7-day expiry) to Shoutrrr email notification.
3. **Custom Report Builder** — User selects columns, filters, groupings, saves as template. Stored in `report_templates` table.
4. **Historical Trends** — Pre-aggregate daily snapshots of key metrics into a `report_snapshots` table for 30/60/90-day trend charts.
5. **Configurable Timezone** — Per-tenant timezone setting instead of hardcoded IST.

---

## Open Questions

1. **PDF detail table row limit** — Proposed 500 rows. Too many? Too few? Could make configurable per report type.
2. **Concurrent generation limit** — Should we cap concurrent report generations per tenant? Proposed: 3 concurrent via River queue configuration.
3. **MinIO bucket** — Single `reports` bucket with tenant-prefixed paths, or per-tenant buckets?
4. **Report generation timeout** — Proposed: 5 minutes max. Kill and mark as failed after that.
