# PatchIQ — UI Specification

> Visual workflow builder, advanced UI elements, and frontend technology stack.

---

## 1. Visual Workflow Builder (React Flow)

The standout visual differentiator. Admins build patch policies and deployment pipelines by dragging and connecting nodes on a canvas — not filling out forms.

**Implementation: React Flow + ELK.js for auto-layout**

**Policy Workflow Canvas:**

```
[Trigger Node]──→[Filter Node]──→[Approval Node]──→[Wave 1]──→[Gate]──→[Wave 2]──→[Complete]
  "New Critical     "Only Linux      "Requires           "10%      "Wait 4h    "90%        "Notify
   CVE detected"     production"      CISO approval"     canary"   + check"    remaining"   team"
```

**Node types for the canvas:**

| Node Type | Purpose | Configuration |
|-----------|---------|---------------|
| **Trigger** | Starts the workflow | New patch, schedule, manual, CVE severity threshold |
| **Filter** | Narrows scope | OS type, group, tag, severity, package name regex |
| **Approval** | Requires human sign-off | Approver role, timeout, escalation chain |
| **Deployment Wave** | Executes patches on a % of targets | Percentage, max parallel, timeout per endpoint |
| **Gate** | Conditional checkpoint | Wait duration, failure threshold (halt if >X% fail), health check |
| **Script** | Run custom pre/post scripts | Inline script editor, timeout, failure behavior |
| **Notification** | Alert stakeholders | Email, Slack, webhook, PagerDuty targets |
| **Rollback** | Revert on failure | Snapshot restore, package downgrade, script |
| **Decision** | Branch based on conditions | If/else on patch type, OS, group membership |
| **Complete** | End state | Success/failure actions, report generation |

**Why this is a differentiator:** No competitor offers this. WSUS has checkboxes. ManageEngine has form wizards. Automox has basic policies. PatchIQ lets admins **see** the entire deployment flow, **understand** decision points, and **modify** them visually. This alone can be a demo-winning feature.

---

## 2. Advanced UI Elements

| UI Element | Library | Purpose |
|-----------|---------|---------|
| **Endpoint Topology Map** | React Flow (network mode) | Interactive map showing agents, their groups, connection status, and patch compliance as color-coded nodes. Click to drill into endpoint detail. |
| **Deployment Timeline** | Custom Gantt (react-chrono or visx) | Real-time horizontal timeline showing deployment wave progress across groups, with status colors and click-to-expand |
| **Live Terminal Console** | xterm.js | Stream live agent output during patch installation — like watching a CI build log |
| **Compliance Heatmap** | visx or D3 | Grid of groups × compliance metrics, color-coded from red (non-compliant) to green (fully patched) |
| **CVE Risk Matrix** | Custom scatter plot | CVSS score vs. number of affected endpoints — visually prioritize what to patch first |
| **Patch Dependency Graph** | React Flow (DAG mode) | Show patch dependencies and conflicts — "installing A requires B first, conflicts with C" |
| **AI Chat Panel** | Custom chat UI | Side panel with natural-language interaction for the MCP AI assistant |
| **Diff Viewer** | react-diff-viewer | Show configuration changes, policy diffs, and audit trail comparisons |
| **Interactive Dashboard Builder** | Configurable widget grid | Let admins drag-and-drop dashboard widgets to customize their home view |

---

## 3. Frontend Technology Stack

| Technology | Purpose |
|-----------|---------|
| **React 18 + TypeScript** | Core framework |
| **React Flow** | Workflow canvas, topology maps, dependency graphs |
| **Tailwind CSS + shadcn/ui** | Design system, accessible components |
| **TanStack Query** | Server-state management, caching, background refetch |
| **TanStack Table** | Data tables with sorting, filtering, pagination, column pinning |
| **Zustand** | Minimal client-side state |
| **Vite** | Build tool |
| **ELK.js** | Auto-layout for graph visualizations |
| **xterm.js** | Terminal emulator for live agent console |
| **visx (Airbnb)** | Charts, heatmaps, and data visualizations |
| **react-hook-form + zod** | Form handling with schema validation |
| **Framer Motion** | Animations for transitions and micro-interactions |
| **Monaco Editor** | Code editor for scripts, policy JSON, and template editing |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Workflow builder | `web/src/flows/policy-workflow/` |
| Topology map | `web/src/flows/topology-map/` |
| Dependency graph | `web/src/flows/dependency-graph/` |
| AI chat panel | `web/src/ai/` |
| Reusable components | `web/src/components/` |
| Pages | `web/src/pages/` |
