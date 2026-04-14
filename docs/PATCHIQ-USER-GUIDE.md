# PatchIQ Patch Manager - Complete User Guide

> Enterprise Patch Management Platform
> Version 1.0 | For Administrators and IT Operations Teams

---

## Table of Contents

1. [Welcome to PatchIQ](#1-welcome-to-patchiq)
2. [Getting Started](#2-getting-started)
3. [Navigating the Interface](#3-navigating-the-interface)
4. [Dashboard](#4-dashboard)
5. [Managing Endpoints](#5-managing-endpoints)
6. [Patch Catalog](#6-patch-catalog)
7. [Deploying Patches](#7-deploying-patches)
8. [Deployment Monitoring & Management](#8-deployment-monitoring--management)
9. [CVE & Vulnerability Management](#9-cve--vulnerability-management)
10. [Policies - Automated Patch Management](#10-policies---automated-patch-management)
11. [Compliance Management](#11-compliance-management)
12. [Alerts & Notifications](#12-alerts--notifications)
13. [Audit Log](#13-audit-log)
14. [Reports](#14-reports)
15. [Workflows](#15-workflows)
16. [Settings & Administration](#16-settings--administration)
17. [Common Workflows - Step by Step](#17-common-workflows---step-by-step)
18. [Troubleshooting](#18-troubleshooting)
19. [Glossary](#19-glossary)

---

## 1. Welcome to PatchIQ

### What is PatchIQ?

PatchIQ is an enterprise patch management platform that helps you discover vulnerabilities, deploy patches, and maintain compliance across your entire IT infrastructure. It works across Windows, Linux, and macOS endpoints from a single management console.

### How PatchIQ Works

PatchIQ uses a three-tier architecture to deliver patch management at scale:

```
                    YOUR ORGANIZATION
    ┌──────────────────────────────────────────────────┐
    │                                                  │
    │   ┌──────────────────────────────────────────┐   │
    │   │         PATCH MANAGER (Server)           │   │
    │   │                                          │   │
    │   │   Your central management console.       │   │
    │   │   This is where YOU log in and work.     │   │
    │   │                                          │   │
    │   │   - View all endpoints                   │   │
    │   │   - Browse patch catalog                 │   │
    │   │   - Deploy patches                       │   │
    │   │   - Create policies                      │   │
    │   │   - Monitor compliance                   │   │
    │   │   - Generate reports                     │   │
    │   └──────────┬───────────────────────────────┘   │
    │              │                                    │
    │              │  Secure gRPC Connection            │
    │              │  (encrypted, authenticated)        │
    │              │                                    │
    │   ┌──────────▼───────────────────────────────┐   │
    │   │           AGENTS (on Endpoints)           │   │
    │   │                                          │   │
    │   │   Small software running on each device. │   │
    │   │                                          │   │
    │   │   +--------+  +--------+  +--------+    │   │
    │   │   | Win PC |  | Linux  |  |  Mac   |    │   │
    │   │   | Agent  |  | Server |  | Agent  |    │   │
    │   │   +--------+  | Agent  |  +--------+    │   │
    │   │               +--------+                 │   │
    │   │                                          │   │
    │   │   Agents automatically:                  │   │
    │   │   - Report installed software            │   │
    │   │   - Install patches when instructed      │   │
    │   │   - Send status back to server           │   │
    │   └──────────────────────────────────────────┘   │
    │                                                  │
    └──────────────────────────────────────────────────┘
                          │
                          │  Internet (REST API)
                          │
              ┌───────────▼──────────────┐
              │    HUB (Cloud Service)    │
              │                          │
              │  Aggregates patches and  │
              │  vulnerability data from │
              │  6 global sources:       │
              │                          │
              │  - NVD (US Gov)          │
              │  - CISA KEV              │
              │  - Microsoft MSRC        │
              │  - Red Hat Advisories    │
              │  - Ubuntu Security       │
              │  - Apple Security        │
              └──────────────────────────┘
```

### Key Concepts

| Concept | What it Means |
|---------|--------------|
| **Endpoint** | Any device managed by PatchIQ (server, workstation, laptop) |
| **Agent** | The small PatchIQ software installed on each endpoint |
| **Patch** | A software update that fixes a bug, vulnerability, or adds improvements |
| **CVE** | Common Vulnerabilities and Exposures - a publicly known security flaw |
| **Deployment** | The process of sending patches to one or more endpoints |
| **Wave** | A group of endpoints that receive a patch together during a phased deployment |
| **Policy** | A set of rules that automatically identifies which patches go to which endpoints |
| **Compliance** | Measuring how well your systems meet security standards (CIS, HIPAA, PCI-DSS, etc.) |

---

## 2. Getting Started

### Step 1: Log In

Open your browser and navigate to your PatchIQ server URL (provided by your administrator).

```
┌─────────────────────────────────────────────┐
│                                             │
│              P A T C H I Q                  │
│                                             │
│   ┌───────────────────────────────────┐     │
│   │  Email or Username                │     │
│   └───────────────────────────────────┘     │
│                                             │
│   ┌───────────────────────────────────┐     │
│   │  Password                    [👁]  │     │
│   └───────────────────────────────────┘     │
│                                             │
│   [✓] Remember me                           │
│                                             │
│   ┌───────────────────────────────────┐     │
│   │          Sign In                  │     │
│   └───────────────────────────────────┘     │
│                                             │
│   Forgot password?    Register              │
│                                             │
└─────────────────────────────────────────────┘
```

Enter your credentials and click **Sign In**. If your organization uses Single Sign-On (SSO), you will be redirected to your identity provider.

### Step 2: Install Agents on Your Endpoints

Before PatchIQ can manage a device, you must install the PatchIQ Agent on it. Navigate to **Agent Downloads** in the sidebar.

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent Downloads                                                │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │     LINUX       │  │     macOS       │  │    WINDOWS      │ │
│  │                 │  │                 │  │                 │ │
│  │  AMD64  [↓]     │  │  ARM64   [↓]   │  │  x64    [↓]    │ │
│  │  ARM64  [↓]     │  │  Intel   [↓]   │  │  x86    [↓]    │ │
│  │                 │  │                 │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  Registration Token                                             │
│  ┌─────────────────────────────────────────────────┐            │
│  │  [Generate Token]                               │            │
│  └─────────────────────────────────────────────────┘            │
│                                                                 │
│  Token: eyJhbGciOiJSUzI1NiIsIn...  [Copy]                     │
│  Expires: 24 hours                                              │
│                                                                 │
│  Installation Command:                                          │
│  ┌─────────────────────────────────────────────────┐            │
│  │ tar xzf patchiq-agent-linux-amd64.tar.gz &&     │  [Copy]   │
│  │ sudo ./patchiq-agent install \                   │           │
│  │   --server https://your-server:8080 \            │           │
│  │   --token eyJhbGciOi...                          │           │
│  └─────────────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

**To install an agent:**

1. Click **Generate Token** to create a registration token
2. Download the agent binary for your endpoint's operating system and architecture
3. Copy the installation command
4. Run the command on the target endpoint (with administrator/root privileges)
5. The endpoint will appear in your **Endpoints** list within minutes

### Step 3: Verify Enrollment

Navigate to **Endpoints** in the sidebar. Your newly enrolled device should appear with a green "Online" status indicator.

---

## 3. Navigating the Interface

### Main Layout

PatchIQ's interface has three main areas:

```
┌──────────┬──────────────────────────────────────────────────────┐
│          │  TOP BAR                                             │
│          │  ┌─────────┐  ┌──────────────────────┐  [🔔][☀][⬇] │
│          │  │Endpoints│  │ Search endpoints...⌘K │              │
│  SIDE    │  └─────────┘  └──────────────────────┘              │
│  BAR     ├──────────────────────────────────────────────────────┤
│          │                                                      │
│  ┌─────┐ │              MAIN CONTENT AREA                       │
│  │ 📊  │ │                                                      │
│  │Dash │ │    This area changes based on which page             │
│  ├─────┤ │    you've selected in the sidebar.                   │
│  │ 🖥  │ │                                                      │
│  │Endpt│ │    Tables, charts, forms, and details                │
│  ├─────┤ │    all appear here.                                  │
│  │ 📦  │ │                                                      │
│  │Patch│ │                                                      │
│  ├─────┤ │                                                      │
│  │ 🛡  │ │                                                      │
│  │CVEs │ │                                                      │
│  ├─────┤ │                                                      │
│  │ 📜  │ │                                                      │
│  │Polic│ │                                                      │
│  ├─────┤ │                                                      │
│  │ 🚀  │ │                                                      │
│  │Dploy│ │                                                      │
│  ├─────┤ │                                                      │
│  │ 🔔  │ │                                                      │
│  │Alert│ │                                                      │
│  ├─────┤ │                                                      │
│  │ 📋  │ │                                                      │
│  │Audit│ │                                                      │
│  ├─────┤ │                                                      │
│  │ ⚙  │ │                                                      │
│  │Sett.│ │                                                      │
│  └─────┘ │                                                      │
│          │                                                      │
│  ┌─────┐ │                                                      │
│  │User │ │                                                      │
│  │Menu │ │                                                      │
│  └─────┘ │                                                      │
└──────────┴──────────────────────────────────────────────────────┘
```

### Sidebar Navigation

The sidebar is organized into logical sections:

| Section | Pages | What You'll Find |
|---------|-------|-----------------|
| **Overview** | Dashboard | At-a-glance metrics, charts, and health indicators |
| **Assets** | Endpoints | All managed devices with status, risk, and patch info |
| **Security** | Patches, CVEs, Policies | Patch catalog, vulnerability database, automation rules |
| **Operations** | Deployments | Active and historical patch deployments |
| **Compliance** | Alerts, Audit, Reports | Notifications, activity log, exportable reports |
| **System** | Settings, Agent Downloads | Configuration, user management, agent installers |

### Top Bar Features

```
┌───────────────────────────────────────────────────────────────────────┐
│  Endpoints / web-server-01    [  Search endpoints, patches, CVEs... ⌘K  ]    [🔔] [☀/🌙] [⬇]  │
│  ↑ Breadcrumb                  ↑ Command Palette                      ↑     ↑       ↑         │
│                                                                    Alerts  Theme  Register     │
└───────────────────────────────────────────────────────────────────────┘
```

- **Breadcrumb** (left) — Shows where you are. Click the parent link to go back to the list.
- **Command Palette** (center) — Press `Ctrl+K` (or `Cmd+K` on Mac) to search across everything: endpoints, patches, CVEs, policies, and more.
- **Alerts Bell** (right) — Quick access to alerts. A red badge appears when critical alerts need attention.
- **Theme Toggle** (right) — Switch between light and dark mode.
- **Register Endpoint** (right) — Quick link to the agent download page.

### User Menu

Click your name at the bottom of the sidebar to access:
- **Account Settings** — Change password, manage API tokens
- **Sign Out** — End your session

### Permissions

If you see a lock icon next to a menu item, you don't have permission to access that area. Contact your administrator to adjust your role.

---

## 4. Dashboard

The Dashboard is your command center. It provides an at-a-glance view of your entire patch management operation.

### Dashboard Layout

```
┌───────────────────────────────────────────────────────────────────────┐
│  Dashboard                               [Preset: Executive ▼] [Edit]│
│                                                                       │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│  │  ENDPOINTS   │ │ CRITICAL CVEs│ │   PATCHES    │ │ DEPLOYMENTS  │ │
│  │   ONLINE     │ │              │ │  AVAILABLE   │ │ IN PROGRESS  │ │
│  │              │ │              │ │              │ │              │ │
│  │    247       │ │     12       │ │     89       │ │      3       │ │
│  │  of 260      │ │  unpatched   │ │  to deploy   │ │   running    │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘ │
│                                                                       │
│  ┌─────────────────────────────────┐ ┌───────────────────────────────┐│
│  │   PATCH VELOCITY TREND          │ │   COMPLIANCE RINGS            ││
│  │                                 │ │                               ││
│  │   Patches deployed/week         │ │    CIS ████████░░  82%       ││
│  │                                 │ │    PCI ██████████  95%       ││
│  │   40 ┤      ╭──╮               │ │   HIPAA ███████░░░  74%      ││
│  │   30 ┤   ╭──╯  ╰──╮           │ │   NIST ████████░░  85%       ││
│  │   20 ┤╭──╯        ╰──╮        │ │                               ││
│  │   10 ┤╯              ╰──      │ │                               ││
│  │      └──────────────────       │ │                               ││
│  │       W1  W2  W3  W4  W5      │ │                               ││
│  └─────────────────────────────────┘ └───────────────────────────────┘│
│                                                                       │
│  ┌─────────────────────────────────┐ ┌───────────────────────────────┐│
│  │   DEPLOYMENT PIPELINE           │ │   TOP VULNERABILITIES         ││
│  │                                 │ │                               ││
│  │   Scheduled → Running → Done    │ │   CVE-2024-1234  CVSS 9.8   ││
│  │   [2]    →    [3]    →  [47]   │ │   CVE-2024-5678  CVSS 9.1   ││
│  │                                 │ │   CVE-2024-9012  CVSS 8.8   ││
│  │                                 │ │   CVE-2024-3456  CVSS 8.5   ││
│  └─────────────────────────────────┘ └───────────────────────────────┘│
└───────────────────────────────────────────────────────────────────────┘
```

### Customizing Your Dashboard

1. Click the **Edit** button in the top-right corner to enter edit mode
2. **Drag widgets** by their header to rearrange them
3. **Resize widgets** by dragging the bottom-right corner
4. Click **Add Widget** to open the widget drawer — drag new widgets onto your dashboard
5. Click **Reset** to restore the default layout
6. Choose a **Preset** from the dropdown (Executive, Operations, Custom, etc.)

### Available Widgets

| Widget | What It Shows |
|--------|--------------|
| **Stat Cards** | Key metrics: endpoints online, critical CVEs, patches available, active deployments |
| **Patch Velocity Trend** | Line chart of patches deployed over time |
| **Compliance Rings** | Ring gauges showing compliance score per framework |
| **Deployment Pipeline** | Visual flow of scheduled, running, and completed deployments |
| **Top Vulnerabilities** | Highest-risk CVEs that need immediate attention |
| **Risk Landscape** | Heatmap of risk across your infrastructure |
| **OS Heatmap** | Breakdown of endpoints by operating system |
| **Activity Feed** | Real-time stream of recent events |
| **Missing Patches** | Patches with the most affected endpoints |
| **MTTR Decay Curve** | Mean-time-to-remediate trend |
| **Exposure Window** | Timeline showing how long vulnerabilities have been open |
| **Blast Radius** | Impact visualization of unpatched CVEs |
| **Drift Detector** | Endpoints drifting from compliance baselines |

### Onboarding Banner

If you have zero endpoints enrolled, the dashboard shows an onboarding banner with a **Download Agent** link to help you get started.

---

## 5. Managing Endpoints

### Viewing Your Endpoints

Navigate to **Endpoints** in the sidebar.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Endpoints                                              [+ Create]   │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │   ALL    │ │  ONLINE  │ │ OFFLINE  │ │ PENDING  │ │  STALE   │   │
│  │   260    │ │   247    │ │    8     │ │    3     │ │    2     │   │
│  │          │ │  (green) │ │  (red)   │ │  (blue)  │ │  (gray)  │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
│                                                                       │
│  [Search...]                                                          │
│                                                                       │
│  ┌─┬──────────────┬──────┬────────┬──────┬───────┬──────┬─────────┐  │
│  │☐│ Hostname     │ OS   │ Status │ Risk │ Seen  │Patch │ Tags    │  │
│  ├─┼──────────────┼──────┼────────┼──────┼───────┼──────┼─────────┤  │
│  │☐│ web-srv-01   │ U 22 │ ● Onl  │  7.2 │ 2m ago│ C:3  │ prod    │  │
│  │☐│ db-srv-01    │ R 9  │ ● Onl  │  4.1 │ 5m ago│ H:2  │ prod,db │  │
│  │☐│ win-pc-042   │ W 11 │ ● Onl  │  8.5 │ 1m ago│ C:5  │ corp    │  │
│  │☐│ mac-dev-07   │ M 14 │ ● Off  │  2.0 │ 3d ago│ M:1  │ dev     │  │
│  │☐│ lin-build-03 │ D 12 │ ● Stl  │  6.3 │ 7d ago│ H:4  │ ci      │  │
│  └─┴──────────────┴──────┴────────┴──────┴───────┴──────┴─────────┘  │
│                                                                       │
│  Rows per page: [20 ▼]                         Page 1 of 13  [<] [>] │
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding the Endpoint Table

| Column | Description |
|--------|-------------|
| **Hostname** | The device name. Click to view full details. |
| **OS** | Operating system with version. Shows a letter badge: **W** (Windows), **U** (Ubuntu), **R** (RHEL), **D** (Debian), **M** (macOS), **C** (CentOS), **F** (Fedora) |
| **Status** | Connection state: **Online** (green dot), **Offline** (red dot), **Pending** (blue dot, awaiting first check-in), **Stale** (gray dot, hasn't reported in a while) |
| **Risk Score** | 0-10 scale. Color-coded: **Green** (0-3, low risk), **Yellow** (3-7, moderate), **Red** (7-10, high risk) |
| **Last Seen** | When the agent last reported in. "2m ago", "3 days ago", etc. |
| **Patches Pending** | Outstanding patches by severity. **C:3** = 3 critical, **H:2** = 2 high, **M:1** = 1 medium |
| **Tags** | Labels assigned to this endpoint for grouping and policy targeting |

### Stat Card Filters

The stat cards at the top work as **quick filters**. Click any card to filter the table:
- Click **Online** to see only online endpoints
- Click **Offline** to see only disconnected devices
- Click the active card again to clear the filter

### Bulk Actions

Select multiple endpoints using the checkboxes, then use the bulk action bar that appears at the bottom:

```
┌───────────────────────────────────────────────────────────────────┐
│  3 endpoints selected    [Scan All] [Deploy] [Assign Tags] [...]  │
└───────────────────────────────────────────────────────────────────┘
```

- **Scan All Selected** — Trigger an inventory scan on all selected endpoints
- **Deploy to Selected** — Open the Deployment Wizard targeting these endpoints
- **Assign Tags** — Apply tags to all selected endpoints at once
- **Decommission All** — Remove selected endpoints from management
- **Export as CSV** — Download endpoint data

### Endpoint Detail Page

Click any hostname to open the full detail view:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Endpoints                                                         │
│                                                                       │
│  web-server-01                      [Deploy Patches] [Scan Now] [···]│
│  ● Online  |  Ubuntu 22.04  |  Agent v1.2.3  |  192.168.1.100       │
│  Enrolled 30 days ago                                                 │
│                                                                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│  │ RISK SCORE  │ │ PATCH       │ │ COMPLIANCE  │ │ LAST SCAN   │    │
│  │             │ │ COVERAGE    │ │             │ │             │    │
│  │    7.2      │ │    92%      │ │   3/5       │ │  2 hrs ago  │    │
│  │  ████████░░ │ │  █████████░ │ │  ██████░░░░ │ │             │    │
│  │   (red)     │ │  (green)    │ │  (yellow)   │ │             │    │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
│                                                                       │
│  [Overview] [Hardware] [Software] [Patches] [CVEs] [Deployments]     │
│  ──────────────────────────────────────────────────────────────       │
│                                                                       │
│                    (Tab content appears here)                         │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### Endpoint Detail Tabs

| Tab | What You'll See |
|-----|----------------|
| **Overview** | Hostname, OS, architecture, agent version, IP address, enrollment date |
| **Hardware** | CPU model, cores, RAM size, disk layout, BIOS/firmware info |
| **Software** | All installed packages/applications with versions |
| **Patches** | Patches applicable to this endpoint — deployed, pending, and failed |
| **CVE Exposure** | All CVEs affecting this endpoint, with severity, CVSS score, and exploit status |
| **Deployments** | History of all deployments targeting this endpoint |
| **Audit** | Complete activity log for this endpoint |

### Endpoint Actions

| Action | How to Access | What It Does |
|--------|---------------|-------------|
| **Deploy Patches** | Button in header | Opens deployment wizard pre-targeted to this endpoint |
| **Scan Now** | Button in header | Triggers an immediate inventory scan (shows spinner while running) |
| **Export Report** | More menu (...) | Downloads a CSV report for this endpoint |
| **Delete Endpoint** | More menu (...) | Decommissions the endpoint (confirmation required) |

---

## 6. Patch Catalog

### Browsing Available Patches

Navigate to **Patches** in the sidebar.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Patches                                                    [Refresh]│
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │ CRITICAL │ │   HIGH   │ │  MEDIUM  │ │   LOW    │               │
│  │    23    │ │    41    │ │    18    │ │     7    │               │
│  │  (red)   │ │ (orange) │ │ (yellow) │ │  (gray)  │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  [Search patches...]                                                  │
│                                                                       │
│  ┌─┬───────────────────┬─────────┬──────┬──────┬─────────┬────────┐  │
│  │☐│ Patch Name        │Severity │  OS  │ CVSS │Affected │Remediate│  │
│  ├─┼───────────────────┼─────────┼──────┼──────┼─────────┼────────┤  │
│  │☐│ curl 7.88.1-10    │CRITICAL │ U 22 │  9.8 │  142    │  12%   │  │
│  │ │ +deb12u5          │  (red)  │      │ ████ │endpoints│ ██░░░░ │  │
│  ├─┼───────────────────┼─────────┼──────┼──────┼─────────┼────────┤  │
│  │☐│ KB5034441         │  HIGH   │ W 11 │  8.1 │   87    │  45%   │  │
│  │ │ 2024-01 Security  │(orange) │      │ ███░ │endpoints│ ████░░ │  │
│  ├─┼───────────────────┼─────────┼──────┼──────┼─────────┼────────┤  │
│  │☐│ openssl 3.0.13    │ MEDIUM  │ R 9  │  6.5 │   34    │  78%   │  │
│  │ │                   │(yellow) │      │ ██░░ │endpoints│ ██████ │  │
│  └─┴───────────────────┴─────────┴──────┴──────┴─────────┴────────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding Patch Table Columns

| Column | Description |
|--------|-------------|
| **Patch Name** | Name and version of the patch or security update |
| **Severity** | Risk level: **Critical** (red), **High** (orange), **Medium** (yellow), **Low** (gray) |
| **OS** | Target operating system (letter badge + version) |
| **CVSS** | Highest CVSS score among linked CVEs (0.0 - 10.0). Visual color bar indicates severity. |
| **Affected Endpoints** | How many of your endpoints need this patch |
| **Remediation %** | Progress bar showing what percentage of affected endpoints have been patched |
| **Released** | When the vendor released this patch |
| **Status** | **Pending** (needs deployment), **Deployed** (100% remediated), **Not Applicable** (superseded or recalled) |

### CVSS Score Color Scale

```
  0.0          4.0          7.0          9.0         10.0
   │            │            │            │            │
   ├────────────┼────────────┼────────────┼────────────┤
   │    LOW     │   MEDIUM   │    HIGH    │  CRITICAL  │
   │   (blue)   │  (yellow)  │  (orange)  │   (red)    │
```

### Patch Detail Page

Click any patch name to see its full details:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Patches                                                           │
│                                                                       │
│  curl 7.88.1-10+deb12u5                                  [Deploy]   │
│  CRITICAL  |  Debian 12  |  Released Jan 14, 2025                    │
│                                                                       │
│  Description:                                                         │
│  Security update for curl addressing buffer overflow vulnerability    │
│  in HTTP/2 header processing.                                        │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  REMEDIATION STATUS                                          │    │
│  │                                                              │    │
│  │  Affected     Patched      Pending      Failed               │    │
│  │    150          80           50           20                  │    │
│  │  ████████   █████░░░░    ████░░░░░░   ██░░░░░░░░            │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  LINKED CVEs                                                         │
│  ┌────────────────┬──────────┬──────┬────────────┬───────────────┐   │
│  │ CVE ID         │ Severity │ CVSS │ Exploit?   │ CISA KEV?     │   │
│  ├────────────────┼──────────┼──────┼────────────┼───────────────┤   │
│  │ CVE-2024-1234  │ CRITICAL │  9.8 │ Yes        │ Yes (Due 2/1) │   │
│  │ CVE-2024-5678  │ HIGH     │  7.5 │ No         │ No            │   │
│  └────────────────┴──────────┴──────┴────────────┴───────────────┘   │
│                                                                       │
│  AFFECTED ENDPOINTS                                                   │
│  ┌──────────────┬────────┬─────────┬──────────────────┐              │
│  │ Hostname     │ OS     │ Status  │ Patch Status     │              │
│  ├──────────────┼────────┼─────────┼──────────────────┤              │
│  │ web-srv-01   │ Deb 12 │ Online  │ Pending          │              │
│  │ web-srv-02   │ Deb 12 │ Online  │ Deployed Jan 15  │              │
│  │ api-srv-01   │ Deb 12 │ Offline │ Failed           │              │
│  └──────────────┴────────┴─────────┴──────────────────┘              │
│                                                                       │
│  DEPLOYMENT HISTORY                                                   │
│  ┌─────────────┬────────────┬────────┬─────────┬─────────┬────────┐ │
│  │ ID          │ Status     │ By     │ Targets │ Success │ Failed │ │
│  ├─────────────┼────────────┼────────┼─────────┼─────────┼────────┤ │
│  │ dep-abc123  │ Completed  │ admin  │   50    │   48    │   2    │ │
│  │ dep-def456  │ Running    │ system │   100   │   42    │   3    │ │
│  └─────────────┴────────────┴────────┴─────────┴─────────┴────────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 7. Deploying Patches

Deploying patches is the core action in PatchIQ. There are multiple ways to initiate a deployment.

### Quick Deploy (From Patch Detail)

The fastest way to deploy a single patch:

1. Go to **Patches** and click a patch name
2. Click the **Deploy** button in the header
3. Fill in the deployment dialog:

```
┌───────────────────────────────────────────────────────────────────┐
│  Deploy: curl 7.88.1-10+deb12u5                              [X] │
│                                                                   │
│  Deployment Name                                                  │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │ curl 7.88.1 - Deployment                                 │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Description                                                      │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │ Deploying critical curl security update to all Linux      │    │
│  │ endpoints.                                                │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Configuration Type                                               │
│  (●) Install    ( ) Rollback                                      │
│                                                                   │
│  Target Endpoints                                                 │
│  ┌─────────────────────────────────────┐                          │
│  │ All Endpoints                    [▼]│                          │
│  │ ─────────────────────────────────── │                          │
│  │  All Endpoints                      │                          │
│  │  Windows Only                       │                          │
│  │  Linux Only                         │                          │
│  │  Critical Endpoints                 │                          │
│  └─────────────────────────────────────┘                          │
│                                                                   │
│  -- OR select specific endpoints --                               │
│                                                                   │
│  ☐ web-srv-01 (Ubuntu 22.04)                                     │
│  ☐ web-srv-02 (Debian 12)                                        │
│  ☐ api-srv-01 (Debian 12)                                        │
│  ☐ db-srv-01  (RHEL 9)                                           │
│                                                                   │
│  Schedule Deployment (optional)                                   │
│  ┌──────────────────┐  ┌──────────────────┐                      │
│  │ Start Date       │  │ Start Time       │                      │
│  │ 2025-01-20       │  │ 02:00 AM         │                      │
│  └──────────────────┘  └──────────────────┘                      │
│                                                                   │
│  Patches to Deploy:                                               │
│  ┌──────────────────────┬─────────────────┬───────────┐          │
│  │ curl 7.88.1-10+deb12 │ 7.88.1-10+deb12 │ CRITICAL │          │
│  └──────────────────────┴─────────────────┴───────────┘          │
│                                                                   │
│                    [Cancel]  [Save as Draft]  [Publish]           │
└───────────────────────────────────────────────────────────────────┘
```

4. Click **Publish** to start the deployment immediately, or **Save as Draft** to review later

### Deploy Critical Patches (From Endpoint Detail)

Deploy multiple critical patches to a single endpoint at once:

1. Go to an endpoint's detail page
2. Click **Deploy Patches**
3. Select the patches to deploy
4. The system creates ONE deployment with all patches, rather than separate deployments for each

### Deployment Wizard (Full Control)

For complex deployments with wave-based rollout:

1. Click **+ New Deployment** from the **Deployments** page
2. Walk through the 4-step wizard:

```
Step 1                Step 2               Step 3               Step 4
SELECT SOURCE    →    SELECT TARGETS   →   SET STRATEGY     →   REVIEW
                                                                 & DEPLOY
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌──────────┐
│ Choose what  │      │ Choose who  │      │ Choose how  │      │ Confirm  │
│ to deploy:   │      │ receives it:│      │ to roll out:│      │ and      │
│              │      │             │      │             │      │ submit   │
│ ○ Policy     │      │ ○ All       │      │ ○ All at    │      │          │
│ ○ Patches    │      │ ○ By tags   │      │   once      │      │ Name     │
│ ○ Catalog    │      │ ○ Manual    │      │ ○ Sequential│      │ Summary  │
│   search     │      │   select    │      │ ○ Wave-based│      │ Impact   │
└─────────────┘      └─────────────┘      └─────────────┘      └──────────┘
```

### Wave-Based Deployment (Recommended for Production)

Wave-based deployment lets you deploy in phases, monitoring success before proceeding:

```
  WAVE 1 (10%)          WAVE 2 (30%)          WAVE 3 (60%)
  ┌────────────┐        ┌────────────┐        ┌────────────┐
  │ 10 servers │  30min │ 30 servers │  60min │ 60 servers │
  │            │ delay  │            │ delay  │            │
  │ ●●●●●●●●●● │ ────→ │ ●●●●●●●●●● │ ────→ │ ●●●●●●●●●● │
  │            │        │ ●●●●●●●●●● │        │ ●●●●●●●●●● │
  │ Success    │        │ ●●●●●●●●●● │        │ ●●●●●●●●●● │
  │ threshold: │        │            │        │ ●●●●●●●●●● │
  │   95%      │        │ Success    │        │ ●●●●●●●●●● │
  │            │        │ threshold: │        │ ●●●●●●●●●● │
  │ Max error: │        │   90%      │        │            │
  │   10%      │        │            │        │ Success    │
  └────────────┘        │ Max error: │        │ threshold: │
                        │   15%      │        │   80%      │
       If Wave 1        └────────────┘        └────────────┘
       fails:
       AUTO-ROLLBACK              If any wave exceeds its
       of all changes             error threshold, the entire
                                  deployment is rolled back.
```

**How waves work:**

1. **Wave 1** deploys to a small percentage (e.g., 10%) of targets
2. PatchIQ monitors the success rate
3. If the success rate meets the threshold (e.g., 95%), it waits the configured delay (e.g., 30 minutes)
4. **Wave 2** deploys to the next batch (e.g., 30%)
5. This continues until all waves complete
6. If any wave's failure rate exceeds the maximum error rate, PatchIQ **automatically rolls back** the entire deployment

### What Happens During a Deployment

Behind the scenes, this is what PatchIQ does:

```
YOU click "Deploy"
        │
        ▼
┌───────────────────┐
│ 1. DEPLOYMENT     │  Deployment record created in database
│    CREATED        │  Status: "created"
└───────┬───────────┘
        │  (within seconds)
        ▼
┌───────────────────┐
│ 2. DEPLOYMENT     │  Executor activates the first wave
│    RUNNING        │  Status: "running"
└───────┬───────────┘
        │  (every 30 seconds, the Wave Dispatcher checks)
        ▼
┌───────────────────┐
│ 3. COMMANDS       │  For each endpoint in the current wave:
│    DISPATCHED     │  - Build install payload (package, version, installer type)
│                   │  - Include download URL if binary needed
│                   │  - Include SHA256 checksum for verification
│                   │  - Respect maintenance windows
│                   │  - Respect max concurrent limit
└───────┬───────────┘
        │  (agent checks in via gRPC)
        ▼
┌───────────────────┐
│ 4. AGENT          │  On the endpoint, the agent:
│    INSTALLS       │  a) Downloads patch binary (if needed)
│                   │  b) Verifies checksum
│                   │  c) Runs pre-install script (if configured)
│                   │  d) Runs OS-specific installer (apt, yum, msi, etc.)
│                   │  e) Runs post-install script (if configured)
│                   │  f) Records rollback information
│                   │  g) Triggers reboot if required
└───────┬───────────┘
        │  (agent reports back)
        ▼
┌───────────────────┐
│ 5. RESULTS        │  Server updates:
│    PROCESSED      │  - Command status (succeeded/failed)
│                   │  - Deployment target status
│                   │  - Wave success/failure counters
│                   │  - Deployment-level counters
└───────┬───────────┘
        │  (wave completion check)
        ▼
┌───────────────────┐
│ 6. WAVE           │  If success threshold met:
│    EVALUATION     │    → Start next wave (after delay)
│                   │  If error rate exceeded:
│                   │    → AUTO-ROLLBACK all changes
│                   │  If last wave completed:
│                   │    → Mark deployment COMPLETED
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 7. NOTIFICATIONS  │  Email, Slack, Discord, or webhook
│    SENT           │  notification sent to configured channels
└───────────────────┘
```

---

## 8. Deployment Monitoring & Management

### Deployments List

Navigate to **Deployments** in the sidebar.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Deployments                                    [+ Create Deployment]│
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │ RUNNING  │ │COMPLETED │ │  FAILED  │ │CANCELLED │               │
│  │    3     │ │    47    │ │    2     │ │    1     │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  ┌──────────┬────────────┬──────────┬──────────┬──────────┬───────┐  │
│  │ Name     │ Status     │ Progress │ Targets  │ Duration │ By    │  │
│  ├──────────┼────────────┼──────────┼──────────┼──────────┼───────┤  │
│  │ curl     │ ● Running  │ ████░░░░ │ 42/100   │ 12 min   │ admin │  │
│  │ deploy   │   (blue)   │  42%     │          │          │       │  │
│  ├──────────┼────────────┼──────────┼──────────┼──────────┼───────┤  │
│  │ KB5034   │ ✓ Complete │ ████████ │ 87/87    │ 45 min   │system │  │
│  │ rollout  │   (green)  │  100%    │          │          │       │  │
│  ├──────────┼────────────┼──────────┼──────────┼──────────┼───────┤  │
│  │ openssl  │ ✗ Failed   │ ██████░░ │ 28/34    │ 20 min   │ admin │  │
│  │ update   │   (red)    │  82%     │ (6 fail) │          │       │  │
│  └──────────┴────────────┴──────────┴──────────┴──────────┴───────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

### Deployment Status Colors

| Status | Color | Meaning |
|--------|-------|---------|
| **Scheduled** | Gray | Waiting for scheduled time |
| **Created** | Blue (light) | Just created, about to start |
| **Running** | Blue (animated) | Actively deploying to endpoints |
| **Completed** | Green | All waves succeeded |
| **Failed** | Red | Error threshold exceeded |
| **Cancelled** | Gray | Manually cancelled by user |
| **Rolling Back** | Orange (animated) | Undoing installed patches |
| **Rolled Back** | Orange | Rollback completed successfully |
| **Rollback Failed** | Dark Red | Rollback could not complete |

### Deployment Detail Page

Click a deployment to see the full detail view:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Deployments                                                       │
│                                                                       │
│  curl 7.88.1 - Deployment                                            │
│  ● RUNNING  |  Started 12 min ago  |  By: admin                     │
│                                                                       │
│  Progress: 42/100 endpoints (42%)                                    │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│   │
│  │  38 success       4 active       2 failed       54 pending   │   │
│  │  (green)          (blue)         (red)          (gray)       │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  [Cancel] [Retry Failed] [Rollback]                                  │
│                                                                       │
│  ═══════════════════════════════════════════════════════════════      │
│  WAVE PIPELINE                                                        │
│  ═══════════════════════════════════════════════════════════════      │
│                                                                       │
│  Wave 1 (10%)           Wave 2 (30%)          Wave 3 (60%)           │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐       │
│  │ ✓ COMPLETED  │ ──→  │ ● RUNNING    │ ──→  │ ○ PENDING    │       │
│  │              │      │              │      │              │       │
│  │ 10/10 (100%)│      │ 28/30 (93%)  │      │ 0/60         │       │
│  │ ●●●●●●●●●●  │      │ ●●●●●●●●●●  │      │ ○○○○○○○○○○  │       │
│  │ all green    │      │ ●●●●●●●●●●  │      │ ○○○○○○○○○○  │       │
│  │              │      │ ●●●●●●●●○○  │      │ ○○○○○○○○○○  │       │
│  │ Completed    │      │              │      │              │       │
│  │ 12 min ago   │      │ 2 failed     │      │ Starts after │       │
│  └──────────────┘      │ 28 success   │      │ Wave 2 + 60m │       │
│                        └──────────────┘      └──────────────┘       │
│                                                                       │
│  ═══════════════════════════════════════════════════════════════      │
│  ENDPOINT STATUS                                                      │
│  ═══════════════════════════════════════════════════════════════      │
│                                                                       │
│  ┌──────────────┬────────┬──────────┬──────────────────────────┐     │
│  │ Hostname     │ Wave   │ Status   │ Details                  │     │
│  ├──────────────┼────────┼──────────┼──────────────────────────┤     │
│  │ web-srv-01   │ Wave 1 │✓ Success │ Installed in 45s         │     │
│  │ web-srv-02   │ Wave 1 │✓ Success │ Installed in 32s         │     │
│  │ api-srv-01   │ Wave 2 │✗ Failed  │ Exit code 1: dep conflict│     │
│  │ api-srv-02   │ Wave 2 │● Active  │ Installing...            │     │
│  │ db-srv-01    │ Wave 3 │○ Pending │ Waiting for Wave 3       │     │
│  └──────────────┴────────┴──────────┴──────────────────────────┘     │
└───────────────────────────────────────────────────────────────────────┘
```

### Deployment Actions

| Action | When Available | What It Does |
|--------|---------------|-------------|
| **Cancel** | While Running or Created | Immediately stops the deployment. Pending targets are cancelled. Already-installed patches remain. |
| **Retry Failed** | After deployment has Failed | Resets all failed targets back to "pending" and restarts the deployment. Only failed endpoints are retried. |
| **Rollback** | While Running or after Completion | Sends rollback commands to endpoints that already installed the patch, reverting them to their previous version. |
| **Download Logs** | Any time | Exports detailed logs for troubleshooting |

### Understanding Rollback

When you trigger a rollback (or it happens automatically):

```
ROLLBACK PROCESS
─────────────────

1. Deployment status changes to "Rolling Back"

2. All remaining waves are CANCELLED
   (endpoints that haven't received the patch yet are safe)

3. All pending commands are CANCELLED

4. For endpoints that already installed the patch:
   → Rollback command sent to each agent
   → Agent reverts to the previous package version
   → Agent reports success/failure

5. If all rollbacks succeed:
   → Status: "Rolled Back" (orange)

6. If any rollback fails:
   → Status: "Rollback Failed" (dark red)
   → Manual intervention required
```

---

## 9. CVE & Vulnerability Management

### CVE Catalog

Navigate to **CVEs** in the sidebar to view all known vulnerabilities affecting your infrastructure.

```
┌───────────────────────────────────────────────────────────────────────┐
│  CVEs                                                                 │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                             │
│  │ CRITICAL │ │   HIGH   │ │  MEDIUM  │                             │
│  │    12    │ │    34    │ │    56    │                             │
│  └──────────┘ └──────────┘ └──────────┘                             │
│                                                                       │
│  [Search CVE ID or description...]                                    │
│                                                                       │
│  ┌──────────────┬──────┬──────────┬────────┬──────┬─────┬─────────┐  │
│  │ CVE ID       │ CVSS │ Severity │ Attack │Explt?│ KEV │Affected │  │
│  ├──────────────┼──────┼──────────┼────────┼──────┼─────┼─────────┤  │
│  │CVE-2024-1234 │  9.8 │ CRITICAL │Network │ Yes  │ Yes │  142    │  │
│  │CVE-2024-5678 │  9.1 │ CRITICAL │Network │ No   │ Yes │   87    │  │
│  │CVE-2024-9012 │  8.8 │  HIGH    │Network │ POC  │ No  │   34    │  │
│  │CVE-2024-3456 │  7.5 │  HIGH    │Local   │ No   │ No  │   12    │  │
│  └──────────────┴──────┴──────────┴────────┴──────┴─────┴─────────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding CVE Columns

| Column | Description |
|--------|-------------|
| **CVE ID** | The unique vulnerability identifier (e.g., CVE-2024-1234) |
| **CVSS** | Common Vulnerability Scoring System score (0.0-10.0). Higher = more dangerous. |
| **Severity** | Derived from CVSS: Critical (9.0+), High (7.0-8.9), Medium (4.0-6.9), Low (0-3.9) |
| **Attack Vector** | How the vulnerability can be exploited: **Network** (remote), **Adjacent** (local network), **Local** (physical access), **Physical** (direct hardware access) |
| **Exploit?** | Whether a known exploit exists: **Yes** (actively exploited), **POC** (proof of concept available), **No** |
| **KEV** | Whether CISA has listed this as a Known Exploited Vulnerability. **Yes** means active exploitation in the wild with a remediation deadline. |
| **Affected** | Number of your endpoints vulnerable to this CVE |

### CVE Detail Page

Click a CVE ID to see the full breakdown:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← CVEs                                                              │
│                                                                       │
│  CVE-2024-1234                                        [Deploy Fixes] │
│  CRITICAL  |  CVSS 9.8  |  Published Dec 1, 2024                    │
│                                                                       │
│  Buffer overflow in curl HTTP/2 header processing allows remote      │
│  code execution via crafted HTTP/2 HEADERS frame.                    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  CVSS v3.1 BREAKDOWN                                         │    │
│  │                                                              │    │
│  │  Attack Vector:     Network         Scope:       Unchanged  │    │
│  │  Attack Complexity: Low             Confidentiality: High   │    │
│  │  Privileges Req:    None            Integrity:      High    │    │
│  │  User Interaction:  None            Availability:   High    │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ⚠  CISA KEV: Remediation due by February 1, 2025                   │
│  ⚠  Exploit: Active exploitation confirmed in the wild              │
│                                                                       │
│  AVAILABLE PATCHES                                                    │
│  ┌──────────────────────┬──────────┬──────────┬──────────────────┐   │
│  │ Patch Name           │ OS       │ Released │ Coverage         │   │
│  ├──────────────────────┼──────────┼──────────┼──────────────────┤   │
│  │ curl 7.88.1-10+deb12 │ Debian 12│ Jan 14   │ 80/142 patched  │   │
│  │ curl 8.5.0-1ubuntu1  │ Ubuntu 22│ Jan 15   │ 45/87 patched   │   │
│  └──────────────────────┴──────────┴──────────┴──────────────────┘   │
│                                                                       │
│  AFFECTED ENDPOINTS                                                   │
│  ┌──────────────┬──────────┬──────────┬────────────────────────────┐ │
│  │ Hostname     │ OS       │ Status   │ Remediation                │ │
│  ├──────────────┼──────────┼──────────┼────────────────────────────┤ │
│  │ web-srv-01   │ Debian 12│ Online   │ Vulnerable (patch pending) │ │
│  │ web-srv-02   │ Debian 12│ Online   │ Patched (Jan 15)           │ │
│  │ api-srv-01   │ Ubuntu 22│ Offline  │ Vulnerable (deploy failed) │ │
│  └──────────────┴──────────┴──────────┴────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 10. Policies - Automated Patch Management

Policies let you define rules for automatic patch deployment. Instead of manually deploying every patch, you create policies that handle it for you.

### Policy Modes

| Mode | Icon | Behavior |
|------|------|----------|
| **Automatic** | Green badge | Patches are deployed automatically on schedule. No human approval needed. |
| **Manual** | Blue badge | Policy identifies applicable patches, but a human must approve each deployment. |
| **Advisory** | Gray badge | Policy only sends notifications about applicable patches. No deployment occurs. |

### Creating a Policy

Navigate to **Policies** and click **+ New Policy**:

```
STEP 1: BASICS
═══════════════════════════════════════════════════════════════

  Policy Name:     [Critical Linux Security Updates          ]
  Description:     [Auto-deploy critical security patches to ]
                   [all production Linux servers              ]

  Mode:
  (●) Automatic — Deploy patches without manual approval
  ( ) Manual    — Require approval before deployment
  ( ) Advisory  — Notify only, do not deploy


STEP 2: PATCH SELECTION
═══════════════════════════════════════════════════════════════

  Severity Filter (select which severities to include):
  [✓] Critical    [✓] High    [ ] Medium    [ ] Low

  OS Filter:
  [✓] Linux (all distributions)
  [ ] Windows
  [ ] macOS

  Additional Filters:
  [ ] Exclude superseded patches
  [✓] Only patches with CVE links

  Preview: 23 patches currently match these criteria


STEP 3: TARGET SELECTION
═══════════════════════════════════════════════════════════════

  Target Endpoints:
  ( ) All endpoints
  (●) By tags
  ( ) Select manually

  Tag Expression:
  ┌──────────────────────────────────────────────────────┐
  │  environment = "production"  AND  os = "linux"       │
  └──────────────────────────────────────────────────────┘

  Preview: 87 endpoints match this criteria


STEP 4: SCHEDULE (Automatic mode only)
═══════════════════════════════════════════════════════════════

  Frequency:    [Weekly               ▼]
  Day:          [Sunday               ▼]
  Time:         [02:00 AM             ▼]
  Timezone:     [UTC                  ▼]

  Maintenance Window:
  Start: [01:00 AM]    End: [05:00 AM]
  (Patches will only install during this window)

  Rollback Configuration:
  [✓] Auto-rollback if failure rate exceeds: [20]%

  Wave Strategy:
  (●) All at once
  ( ) Phased rollout (waves)


STEP 5: REVIEW
═══════════════════════════════════════════════════════════════

  Policy:     Critical Linux Security Updates
  Mode:       Automatic
  Patches:    Critical + High severity, Linux only (23 patches)
  Targets:    87 endpoints (production Linux servers)
  Schedule:   Every Sunday at 2:00 AM UTC
  Window:     1:00 AM - 5:00 AM
  Rollback:   Auto-rollback at 20% failure rate

                              [Cancel]  [Create Policy]
```

### Managing Policies

```
┌───────────────────────────────────────────────────────────────────────┐
│  Policies                                           [+ New Policy]   │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │   ALL    │ │AUTOMATIC │ │  MANUAL  │ │ ADVISORY │               │
│  │    8     │ │    3     │ │    3     │ │    2     │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  ┌──────────────────────┬──────────┬──────┬──────────┬─────┬──────┐  │
│  │ Policy Name          │ Mode     │Target│ Schedule │On/Of│ Menu │  │
│  ├──────────────────────┼──────────┼──────┼──────────┼─────┼──────┤  │
│  │ Critical Linux       │AUTOMATIC │ 87   │ Sun 2AM  │ [●] │ [···]│  │
│  │ Windows Monthly      │ MANUAL   │ 156  │ 2nd Tue  │ [●] │ [···]│  │
│  │ macOS Advisory       │ADVISORY  │ 23   │ Daily 8AM│ [○] │ [···]│  │
│  └──────────────────────┴──────────┴──────┴──────────┴─────┴──────┘  │
└───────────────────────────────────────────────────────────────────────┘
```

### Policy Actions (from the menu)

| Action | Description |
|--------|-------------|
| **Edit** | Modify the policy settings |
| **Evaluate** | Run the policy evaluation immediately (see how many patches/endpoints match) |
| **Deploy** | Manually trigger a deployment using this policy's criteria |
| **Copy** | Duplicate the policy as a starting point for a new one |
| **Delete** | Remove the policy |
| **Enable/Disable toggle** | Turn the policy on or off without deleting it |

---

## 11. Compliance Management

PatchIQ evaluates your infrastructure against industry security frameworks.

### Supported Frameworks

| Framework | Focus Area |
|-----------|-----------|
| **CIS Benchmarks** | System hardening best practices |
| **PCI-DSS** | Payment card data protection |
| **HIPAA** | Healthcare data privacy |
| **NIST** | Federal cybersecurity standards |
| **ISO 27001** | Information security management |
| **SOC 2** | Service organization controls |

### Compliance Dashboard

```
┌───────────────────────────────────────────────────────────────────────┐
│  Compliance                          [Manage Frameworks] [Evaluate]  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  OVERALL COMPLIANCE SCORE                                    │    │
│  │                                                              │    │
│  │         ╭──────╮                                             │    │
│  │        ╱        ╲      Frameworks: 4 enabled                │    │
│  │       │   84%    │     Overdue Controls: 7                  │    │
│  │        ╲        ╱                                            │    │
│  │         ╰──────╯                                             │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌──────────┐ │
│  │  CIS          │ │  PCI-DSS      │ │  HIPAA        │ │  NIST    │ │
│  │               │ │               │ │               │ │          │ │
│  │  Score: 82%   │ │  Score: 95%   │ │  Score: 74%   │ │Score:85% │ │
│  │  ████████░░   │ │  █████████░   │ │  ███████░░░   │ │████████░ │ │
│  │               │ │               │ │               │ │          │ │
│  │  42/50 ctrls  │ │  28/30 ctrls  │ │  37/50 ctrls  │ │34/40 ctl │ │
│  │               │ │               │ │               │ │          │ │
│  │  Last eval:   │ │  Last eval:   │ │  Last eval:   │ │Last eval:│ │
│  │  2 hours ago  │ │  2 hours ago  │ │  2 hours ago  │ │2 hrs ago │ │
│  │               │ │               │ │               │ │          │ │
│  │  [Evaluate]   │ │  [Evaluate]   │ │  [Evaluate]   │ │[Evaluate]│ │
│  └───────────────┘ └───────────────┘ └───────────────┘ └──────────┘ │
│                                                                       │
│  OVERDUE CONTROLS                                                     │
│  ┌───────────┬────────────────┬──────────────────┬──────────────┐    │
│  │ Framework │ Control ID     │ Control Name     │ Days Overdue │    │
│  ├───────────┼────────────────┼──────────────────┼──────────────┤    │
│  │ HIPAA     │ 164.312(a)(1)  │ Access Control   │ 12 days      │    │
│  │ CIS       │ 5.1.4          │ Ensure rsyslog   │ 5 days       │    │
│  │ HIPAA     │ 164.312(e)(1)  │ Transmission Sec │ 3 days       │    │
│  └───────────┴────────────────┴──────────────────┴──────────────┘    │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 12. Alerts & Notifications

### Alerts Page

Navigate to **Alerts** in the sidebar. A red badge shows the count of unread critical/warning alerts.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Alerts                              [Alert Rules] [Refresh: 30s ▼] │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                             │
│  │ UNREAD   │ │ CRITICAL │ │ WARNING  │                             │
│  │    5     │ │    2     │ │    3     │                             │
│  └──────────┘ └──────────┘ └──────────┘                             │
│                                                                       │
│  ┌──────────┬────────────────────────────────────────┬──────┬──────┐ │
│  │ Severity │ Alert                                  │Status│ Time │ │
│  ├──────────┼────────────────────────────────────────┼──────┼──────┤ │
│  │ CRITICAL │ Deployment "KB5034" failed on 6        │Unread│ 5m   │ │
│  │  (red)   │ endpoints. Error rate exceeded 20%.    │      │ ago  │ │
│  ├──────────┼────────────────────────────────────────┼──────┼──────┤ │
│  │ CRITICAL │ CVE-2024-1234 exploit detected.        │Unread│ 1h   │ │
│  │  (red)   │ 142 endpoints affected.                │      │ ago  │ │
│  ├──────────┼────────────────────────────────────────┼──────┼──────┤ │
│  │ WARNING  │ 3 endpoints offline for >24 hours.     │Unread│ 2h   │ │
│  │ (orange) │ Last heartbeat: db-srv-03, web-04...   │      │ ago  │ │
│  └──────────┴────────────────────────────────────────┴──────┴──────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

### Alert Actions

- **Mark as Read** — Acknowledge you've seen the alert
- **Acknowledge** — Confirm you're working on it
- **Dismiss** — Remove the alert from the active list
- **View Details** — Navigate to the related resource (deployment, CVE, endpoint)

### Configuring Notification Channels

Navigate to **Settings > Notifications** to set up where alerts are delivered:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Notification Channels                                                │
│                                                                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │                  │  │                 │  │                 │      │
│  │    EM            │  │    SL           │  │    WH           │      │
│  │    Email         │  │    Slack        │  │    Webhook      │      │
│  │                  │  │                 │  │                 │      │
│  │  smtp://smtp...  │  │  Not configured │  │  https://...    │      │
│  │                  │  │                 │  │                 │      │
│  │    [Test]        │  │    [Set up]     │  │    [Test]       │      │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘      │
│                                                                       │
│  Event Preferences                                                    │
│  ┌────────────────────────────┬───────┬───────┬─────────┬──────────┐ │
│  │ Event                      │ Email │ Slack │ Webhook │ Urgency  │ │
│  ├────────────────────────────┼───────┼───────┼─────────┼──────────┤ │
│  │ Patch deployed             │ [●]   │ [○]   │  [●]    │Immediate │ │
│  │ Patch failed               │ [●]   │ [●]   │  [●]    │Immediate │ │
│  │ CVE critical discovered    │ [●]   │ [●]   │  [●]    │Immediate │ │
│  │ Endpoint offline           │ [○]   │ [●]   │  [○]    │ Daily    │ │
│  │ Policy triggered           │ [○]   │ [○]   │  [●]    │ Daily    │ │
│  └────────────────────────────┴───────┴───────┴─────────┴──────────┘ │
│                                                                       │
│  Digest Settings                                                      │
│  Frequency: [Daily ▼]   Time: [08:00 AM ▼]   Format: [HTML ▼]      │
│                                                  [Send Test Digest]   │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 13. Audit Log

The Audit Log is an **immutable record** of every action taken in PatchIQ. It cannot be edited or deleted.

Navigate to **Audit** in the sidebar:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Audit Log                                     [Stream] [Timeline]   │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │  TOTAL   │ │  SYSTEM  │ │   USER   │ │  TODAY   │               │
│  │  12,847  │ │  8,234   │ │  4,613   │ │   342    │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  Filters:                                                             │
│  Event: [All Types ▼]  Actor: [Search...]  Date: [Last 24h ▼]       │
│                                                                       │
│  ┌──────────────────┬──────────┬──────────┬──────────────────────┐   │
│  │ Timestamp        │ Actor    │ Resource │ Event                │   │
│  ├──────────────────┼──────────┼──────────┼──────────────────────┤   │
│  │ 10:30:45 AM      │ admin    │Deployment│ deployment.completed │   │
│  │ (2 min ago)      │          │ dep-abc  │ 87/87 targets success│   │
│  ├──────────────────┼──────────┼──────────┼──────────────────────┤   │
│  │ 10:28:12 AM      │ System   │ Patch    │ patch.discovered     │   │
│  │ (5 min ago)      │          │ curl 7.88│ New patch from Hub   │   │
│  ├──────────────────┼──────────┼──────────┼──────────────────────┤   │
│  │ 10:15:00 AM      │ admin    │ Policy   │ policy.evaluated     │   │
│  │ (18 min ago)     │          │ Critical │ 23 patches, 87 endpts│   │
│  └──────────────────┴──────────┴──────────┴──────────────────────┘   │
│                                                                       │
│  Click any row to expand and see the full event details (JSON).      │
│                                                                       │
│                                              [Export CSV] [Export JSON]│
└───────────────────────────────────────────────────────────────────────┘
```

### Audit Event Categories

| Category | Events Captured |
|----------|----------------|
| **Endpoint** | Created, updated, deleted, enrolled, scanned |
| **Patch** | Discovered, synced from Hub |
| **Deployment** | Created, started, completed, failed, cancelled, rolled back |
| **Policy** | Created, updated, deleted, evaluated, auto-deployed |
| **CVE** | Discovered, linked to endpoint, remediation available |
| **Compliance** | Framework enabled, evaluation completed, threshold breach |
| **User/Role** | Role created, user role assigned, login, logout |
| **Notification** | Channel configured, notification sent, notification failed |
| **Settings** | General updated, IAM updated, license loaded |

---

## 14. Reports

### Generating Reports

Navigate to **Reports** in the sidebar:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Reports                                         [+ Generate Report] │
│                                                                       │
│  ┌──────────────────┬─────────────┬──────┬───────────┬──────┬──────┐ │
│  │ Report Name      │ Type        │Format│ Status    │ Size │ Date │ │
│  ├──────────────────┼─────────────┼──────┼───────────┼──────┼──────┤ │
│  │ Q1 Patch Status  │ Patch Status│ PDF  │✓ Complete │ 2.4MB│ Jan 1│ │
│  │ HIPAA Compliance │ Compliance  │ XLSX │✓ Complete │ 1.1MB│ Dec 1│ │
│  │ Monthly Vulns    │Vulnerability│ CSV  │● Generating│  —  │ Now  │ │
│  └──────────────────┴─────────────┴──────┴───────────┴──────┴──────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

### Generate Report Dialog

Click **+ Generate Report** and configure:

| Field | Options |
|-------|---------|
| **Report Type** | Patch Status, Compliance, Vulnerability, Deployment Summary |
| **Date Range** | Last 7 days, Last 30 days, Last 90 days, Custom range |
| **Format** | PDF (formatted), CSV (raw data), XLSX (Excel) |
| **Scope** | All endpoints, Specific groups, Specific endpoints |
| **Sections** | Executive Summary, Details, Remediation Status, Compliance, etc. |

Click **Generate** and the report appears in the list. Downloads start automatically when ready.

---

## 15. Workflows

Workflows let you build custom automation using a visual drag-and-drop canvas.

### Workflow Editor

Navigate to **Workflows** and create or edit a workflow:

```
┌───────────────────────────────────────────────────────────────────────┐
│  My Workflow                    [Saved ✓]  [?] [↩] [↪] [⊞] [Save]  │
│                                                                       │
│  ┌──────────┐  ┌────────────────────────────────────────────────────┐│
│  │ PALETTE  │  │                                                    ││
│  │          │  │                                                    ││
│  │ Triggers │  │     ┌──────────┐                                   ││
│  │ ├ Event  │  │     │ TRIGGER  │                                   ││
│  │ ├ Sched  │  │     │ On CVE   │                                   ││
│  │          │  │     │ Critical │                                   ││
│  │ Actions  │  │     └────┬─────┘                                   ││
│  │ ├ Deploy │  │          │                                         ││
│  │ ├ Script │  │     ┌────▼─────┐                                   ││
│  │ ├ Notify │  │     │ DECISION │                                   ││
│  │ ├ Scan   │  │     │ Is prod? │                                   ││
│  │          │  │     └──┬────┬──┘                                   ││
│  │ Control  │  │   Yes  │    │  No                                  ││
│  │ ├ Branch │  │   ┌────▼──┐ ┌──▼────┐                             ││
│  │ ├ Delay  │  │   │DEPLOY │ │NOTIFY │                             ││
│  │ ├ End    │  │   │Waves  │ │Slack  │                             ││
│  │          │  │   └───────┘ └───────┘                             ││
│  └──────────┘  │                                                    ││
│                └────────────────────────────────────────────────────┘│
│  2 nodes, 3 connections                               Valid ✓       │
└───────────────────────────────────────────────────────────────────────┘
```

### Available Node Types

| Node Type | Purpose |
|-----------|---------|
| **Trigger** | Starts the workflow (on event, on schedule) |
| **Action** | Performs an operation (deploy patches, run script, send notification, scan endpoint) |
| **Decision** | Branches the workflow based on a condition (endpoint attribute, event data) |
| **Delay** | Pauses the workflow for a specified time |
| **Script** | Runs a custom script |
| **End** | Terminates the workflow |

---

## 16. Settings & Administration

### Settings Navigation

```
┌──────────────────┐
│ SETTINGS         │
│                  │
│ Configuration    │
│ ├ General        │  ← Organization name, timezone, scan interval
│ ├ Identity       │  ← SSO / OIDC configuration (Zitadel)
│ ├ Notifications  │  ← Email, Slack, Discord, webhook channels
│                  │
│ Account          │
│ ├ My Account     │  ← Password, API tokens, sessions
│ ├ License        │  ← License tier, seats, expiration
│ ├ Appearance     │  ← Theme (light/dark), density
│                  │
│ Admin            │
│ ├ Tags           │  ← Create/manage endpoint tags
│ ├ Roles          │  ← RBAC role definitions with permission matrix
│ ├ User Roles     │  ← Assign roles to users
│                  │
│ System           │
│ ├ About          │  ← Version info, build details
└──────────────────┘
```

### Role-Based Access Control (RBAC)

PatchIQ uses a permission matrix to control access:

```
┌───────────────────┬───────┬────────┬────────┬────────┐
│ Resource          │ Read  │ Create │ Update │ Delete │
├───────────────────┼───────┼────────┼────────┼────────┤
│ Endpoints         │  [✓]  │  [✓]   │  [✓]   │  [○]   │
│ Patches           │  [✓]  │  [○]   │  [○]   │  [○]   │
│ Deployments       │  [✓]  │  [✓]   │  [✓]   │  [○]   │
│ Policies          │  [✓]  │  [✓]   │  [✓]   │  [○]   │
│ CVEs              │  [✓]  │  [○]   │  [○]   │  [○]   │
│ Audit             │  [✓]  │  [○]   │  [○]   │  [○]   │
│ Reports           │  [✓]  │  [✓]   │  [○]   │  [✓]   │
│ Settings          │  [✓]  │  [○]   │  [✓]   │  [○]   │
│ Roles             │  [○]  │  [○]   │  [○]   │  [○]   │
└───────────────────┴───────┴────────┴────────┴────────┘
  [✓] = Permitted    [○] = Denied
```

---

## 17. Common Workflows - Step by Step

### Workflow 1: Emergency Critical Patch Deployment

**Scenario:** A critical zero-day CVE is announced. You need to patch all affected systems immediately.

```
STEP 1                    STEP 2                    STEP 3
Go to CVEs            →   Find the CVE          →   Click "Deploy Fixes"
                          (filter: Critical)

STEP 4                    STEP 5                    STEP 6
Select target         →   Choose strategy:       →   Click "Deploy"
endpoints                  Wave-based (10/30/60)
(All affected)             for safety

STEP 7                    STEP 8                    STEP 9
Monitor on            →   Check per-endpoint     →   If any fail:
Deployments page          status in detail view       Click "Retry Failed"
```

**Visual flow:**

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  CVEs   │ ──→ │  Find   │ ──→ │ Deploy  │ ──→ │ Monitor │
│  page   │     │  CVE    │     │  Fixes  │     │Progress │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
                                      │
                                      ▼
                                ┌─────────┐
                                │  Wave 1 │ 10% → verify → 
                                │  Wave 2 │ 30% → verify →
                                │  Wave 3 │ 60% → done
                                └─────────┘
```

### Workflow 2: Setting Up Automated Patching

**Scenario:** You want critical and high-severity patches deployed automatically every Sunday at 2 AM to production servers.

```
STEP 1                    STEP 2                    STEP 3
Go to Policies        →   Click "+ New Policy"   →   Name: "Prod Auto-Patch"
                                                      Mode: Automatic

STEP 4                    STEP 5                    STEP 6
Patch Selection:      →   Target Selection:      →   Schedule:
[✓] Critical              By tags:                    Weekly, Sunday
[✓] High                  env = production            2:00 AM UTC
[ ] Medium                os = linux                  Window: 1AM-5AM
[ ] Low

STEP 7                    STEP 8
Review & Create       →   Policy runs automatically
                          every Sunday at 2 AM.
                          Check Deployments page
                          Monday morning.
```

### Workflow 3: Onboarding New Endpoints

**Scenario:** You're adding 50 new servers to PatchIQ.

```
STEP 1                    STEP 2                    STEP 3
Go to Agent           →   Generate a              →   Download agent
Downloads                  Registration Token          for target OS

STEP 4                    STEP 5                    STEP 6
Run install command   →   Endpoints appear in     →   Assign tags to
on each server             Endpoints list               new endpoints
(use automation tool       (status: Online)             (e.g., "prod", "web")
like Ansible)

STEP 7                    STEP 8
Trigger Scan on       →   Existing policies
new endpoints              automatically evaluate
(bulk select → Scan)       new endpoints on
                          next scheduled run
```

### Workflow 4: Investigating a Failed Deployment

**Scenario:** A deployment shows "Failed" status. You need to find out why and fix it.

```
STEP 1                    STEP 2                    STEP 3
Go to Deployments     →   Click the failed        →   Check the progress
                          deployment                   bar: how many failed?

STEP 4                    STEP 5                    STEP 6
Scroll to Endpoint    →   Find failed endpoints   →   Click endpoint to
Status table               and check error             see full details
                          messages

STEP 7                    STEP 8
Fix the root cause    →   Go back to deployment
(e.g., dependency          and click
conflict, disk space)      "Retry Failed"

                                    │
                          ┌─────────▼─────────┐
                          │  Common Causes:    │
                          │                    │
                          │  - Disk full       │
                          │  - Dependency      │
                          │    conflict        │
                          │  - Agent offline   │
                          │  - Maintenance     │
                          │    window closed   │
                          │  - Permission      │
                          │    denied          │
                          └────────────────────┘
```

### Workflow 5: Generating a Compliance Report for Auditors

```
STEP 1                    STEP 2                    STEP 3
Go to Compliance      →   Review framework        →   Click "Evaluate All"
                          scores                       to refresh scores

STEP 4                    STEP 5                    STEP 6
Go to Reports         →   Click "+ Generate       →   Configure:
                          Report"                      Type: Compliance
                                                       Range: Last 90 days
                                                       Format: PDF

STEP 7                    STEP 8
Report generates      →   Download and share
(may take a few            with auditors
minutes for large
environments)
```

---

## 18. Troubleshooting

### Endpoint Not Appearing After Agent Install

| Check | Solution |
|-------|----------|
| Is the agent service running? | Run `patchiq-agent status` on the endpoint |
| Can the endpoint reach the server? | Test connectivity to server port 50051 (gRPC) |
| Is the registration token valid? | Tokens expire after 24 hours. Generate a new one. |
| Is the endpoint's clock accurate? | TLS certificates require accurate system time |

### Deployment Stuck in "Running"

| Check | Solution |
|-------|----------|
| Are target endpoints online? | Check Endpoints page for offline devices |
| Is it within the maintenance window? | Patches only install during configured windows |
| Has the max concurrent limit been reached? | Check deployment's max_concurrent setting |
| Are commands timing out? | Default timeout is 30 minutes. Check agent logs. |

### Patches Not Appearing in Catalog

| Check | Solution |
|-------|----------|
| Is Hub sync working? | Check Settings for last catalog sync time |
| Is the patch for your OS? | Patches are OS-specific. Verify OS family matches. |
| Has the patch been recalled? | Recalled patches are hidden from the active catalog |

### Agent Showing "Stale" Status

The agent hasn't sent a heartbeat recently. Common causes:
- Endpoint is powered off or unreachable
- Agent service crashed — restart it
- Network firewall blocking gRPC port 50051
- Agent needs updating to latest version

---

## 19. Glossary

| Term | Definition |
|------|-----------|
| **Agent** | The PatchIQ software installed on managed endpoints that reports inventory and executes patch commands |
| **Attack Vector** | How a vulnerability can be exploited (Network, Adjacent, Local, Physical) |
| **CISA KEV** | Cybersecurity and Infrastructure Security Agency's Known Exploited Vulnerabilities catalog |
| **Compliance Framework** | A set of security controls and benchmarks (CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2) |
| **CVE** | Common Vulnerabilities and Exposures — a unique identifier for a publicly known security vulnerability |
| **CVSS** | Common Vulnerability Scoring System — rates vulnerability severity from 0.0 (none) to 10.0 (critical) |
| **Deployment** | The process of distributing and installing patches on one or more endpoints |
| **Deployment Target** | A single endpoint+patch combination within a deployment |
| **Endpoint** | Any device managed by PatchIQ (server, workstation, laptop, virtual machine) |
| **gRPC** | The encrypted communication protocol used between agents and the Patch Manager server |
| **Hub** | The central cloud service that aggregates vulnerability data from global sources |
| **Maintenance Window** | A scheduled time period during which patches are allowed to be installed |
| **Patch** | A software update that fixes bugs, vulnerabilities, or adds improvements |
| **Patch Manager** | The on-premises server that orchestrates all patch management operations |
| **Policy** | A set of rules defining which patches should be deployed to which endpoints and when |
| **RBAC** | Role-Based Access Control — permissions system controlling who can do what |
| **Remediation** | The process of fixing a vulnerability by applying the appropriate patch |
| **Rollback** | Reverting an installed patch to the previous version |
| **Severity** | Risk classification: Critical, High, Medium, Low |
| **Tag** | A label applied to endpoints for grouping and policy targeting |
| **Tenant** | An organizational unit in PatchIQ (for multi-tenant/MSP deployments) |
| **Wave** | A group of endpoints that receive a patch together during phased deployment |

---

*PatchIQ Patch Manager User Guide v1.0*
*For support, contact your system administrator or visit the PatchIQ documentation portal.*
