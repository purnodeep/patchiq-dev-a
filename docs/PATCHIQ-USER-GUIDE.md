# Patch Manager - Complete User Guide

> Enterprise Patch Management Platform
> Version 1.0 | For Administrators and IT Operations Teams

---

## Table of Contents

1. [Welcome to Patch Manager](#1-welcome-to-patch-manager)
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

## 1. Welcome to Patch Manager

### What is Patch Manager?

Patch Manager is an enterprise patch management platform that helps you discover vulnerabilities, deploy patches, and maintain compliance across your entire IT infrastructure. It works across Windows, Linux, and macOS endpoints from a single management console.

### How Patch Manager Works

Patch Manager uses a three-tier architecture to deliver patch management at scale:

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
| **Endpoint** | Any device managed by Patch Manager (server, workstation, laptop) |
| **Agent** | The small Patch Manager software installed on each endpoint |
| **Patch** | A software update that fixes a bug, vulnerability, or adds improvements |
| **CVE** | Common Vulnerabilities and Exposures — a publicly known security flaw |
| **Deployment** | The process of sending patches to one or more endpoints |
| **Wave** | A group of endpoints that receive a patch together during a phased deployment |
| **Policy** | A set of rules that automatically identifies which patches go to which endpoints |
| **Compliance** | Measuring how well your systems meet security standards (CIS, HIPAA, PCI-DSS, etc.) |

---

## 2. Getting Started

### Step 1: Log In

Open your browser and navigate to your Patch Manager server URL (provided by your administrator).

```
┌─────────────────────────────────────────────┐
│                                             │
│           P A T C H   M A N A G E R         │
│                                             │
│   ┌───────────────────────────────────┐     │
│   │  Email or Username                │     │
│   └───────────────────────────────────┘     │
│                                             │
│   ┌───────────────────────────────────┐     │
│   │  Password                         │     │
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

Before Patch Manager can manage a device, you must install the Agent on it. Navigate to **Agent Downloads** in the sidebar.

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

Patch Manager's interface has three main areas:

```
┌──────────┬──────────────────────────────────────────────────────┐
│          │  TOP BAR                                             │
│          │  ┌─────────┐  ┌──────────────────────┐  [🔔][☀][⬇] │
│          │  │Endpoints│  │ Search endpoints...⌘K │              │
│  SIDE    │  └─────────┘  └──────────────────────┘              │
│  BAR     ├──────────────────────────────────────────────────────┤
│          │                                                      │
│ Overview │              MAIN CONTENT AREA                       │
│  ┌─────┐ │                                                      │
│  │Dash │ │    This area changes based on which page             │
│  │board│ │    you've selected in the sidebar.                   │
│ Assets  │ │                                                      │
│  ┌─────┐ │    Tables, charts, forms, and details                │
│  │Endpt│ │    all appear here.                                  │
│ Security│ │                                                      │
│  ┌─────┐ │                                                      │
│  │Patch│ │                                                      │
│  ├─────┤ │                                                      │
│  │CVEs │ │                                                      │
│  ├─────┤ │                                                      │
│  │Polic│ │                                                      │
│ Ops     │ │                                                      │
│  ┌─────┐ │                                                      │
│  │Dploy│ │                                                      │
│ Complnce│ │                                                      │
│  ┌─────┐ │                                                      │
│  │Alert│ │                                                      │
│  ├─────┤ │                                                      │
│  │Audit│ │                                                      │
│  ├─────┤ │                                                      │
│  │Reprt│ │                                                      │
│ System  │ │                                                      │
│  ┌─────┐ │                                                      │
│  │Sett.│ │                                                      │
│  ├─────┤ │                                                      │
│  │Agent│ │                                                      │
│  │Dwnld│ │                                                      │
│  └─────┘ │                                                      │
│  ┌─────┐ │                                                      │
│  │User │ │                                                      │
│  │Menu │ │                                                      │
│  └─────┘ │                                                      │
└──────────┴──────────────────────────────────────────────────────┘
```

### Sidebar Navigation

The sidebar is organized into 6 sections:

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
┌────────────────────────────────────────────────────────────────────────────┐
│  Endpoints / web-server-01    [ Search endpoints, patches, CVEs… ⌘K ]  [🔔] [☀] [⬇] │
│  ↑ Breadcrumb                  ↑ Command Palette                     ↑    ↑    ↑      │
│                                                                   Notif Theme Register │
└────────────────────────────────────────────────────────────────────────────┘
```

- **Breadcrumb** (left) — Shows where you are. On detail pages, shows the parent page link and the current entity name. Click the parent to go back.
- **Command Palette** (center) — Press `Ctrl+K` (or `Cmd+K` on Mac) to search across everything: endpoints, patches, CVEs, and more. The placeholder reads: *"Search endpoints, patches, CVEs..."*
- **Notifications** (right) — Bell icon. Navigates to the notifications page.
- **Theme Toggle** (right) — Toggle between light and dark mode. Shows a Moon icon in light mode, Sun icon in dark mode.
- **Register Endpoint** (right) — Download icon. Navigates to the endpoints page with the registration panel open.

### User Menu

Click your name at the bottom of the sidebar to access:
- **Account Settings** — Navigates to your account settings page
- **Sign out** — End your session

### Permissions

If a sidebar item appears dimmed with a lock icon, you don't have permission to access that area. Contact your administrator to adjust your role.

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
| **Top CVEs** | Highest-risk CVEs that need immediate attention |
| **Risk Ranking** | Risk ranking across your infrastructure |
| **Vulnerability Matrix** | Vulnerability breakdown visualization |
| **Activity Feed** | Real-time stream of recent events |
| **Activity Timeline** | Chronological activity display |
| **Missing Patches** | Patches with the most affected endpoints |
| **Remediation Progress** | Progress tracking for remediation efforts |
| **Exposure Window Timeline** | Timeline showing how long vulnerabilities have been open |
| **Blast Radius** | Impact visualization of unpatched CVEs |
| **Drift Detector** | Endpoints drifting from compliance baselines |
| **OS Heatmap** | Breakdown of endpoints by operating system |
| **Health Scorecard** | Overall infrastructure health metrics |
| **Quick Actions** | Shortcuts to common operations |

### Onboarding Banner

If you have zero endpoints enrolled, the dashboard shows an onboarding banner with a **Download Agent** link to help you get started.

---

## 5. Managing Endpoints

### Viewing Your Endpoints

Navigate to **Endpoints** in the sidebar.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Endpoints                                                           │
│                                                                       │
│  ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │TOTAL ENDPT│ │  ONLINE  │ │ OFFLINE  │ │ PENDING  │ │  STALE   │  │
│  │   260     │ │   247    │ │    8     │ │    3     │ │    2     │  │
│  │           │ │  (green) │ │  (red)   │ │  (blue)  │ │  (gray)  │  │
│  └───────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│                                                                       │
│  [Search...]                                                          │
│                                                                       │
│  ┌─┬──────────────┬──────┬────────┬─────────┬──────┬───────┬──────┬──────┐
│  │☐│ Hostname     │ OS   │ Status │Agent Ver│ Risk │ Seen  │Patch │ Tags │
│  ├─┼──────────────┼──────┼────────┼─────────┼──────┼───────┼──────┼──────┤
│  │☐│ web-srv-01   │ U 22 │ ● Onl  │ v1.2.3  │ 7.2  │ 2m ago│ C:3  │ prod │
│  │☐│ db-srv-01    │ R 9  │ ● Onl  │ v1.2.3  │ 4.1  │ 5m ago│ H:2  │ db   │
│  │☐│ win-pc-042   │ W 11 │ ● Onl  │ v1.2.1  │ 8.5  │ 1m ago│ C:5  │ corp │
│  │☐│ mac-dev-07   │ M 14 │ ● Off  │ v1.1.0  │ 2.0  │ 3d ago│ M:1  │ dev  │
│  │☐│ lin-build-03 │ D 12 │ ● Stl  │ v1.0.9  │ 6.3  │ 7d ago│ H:4  │ ci   │
│  └─┴──────────────┴──────┴────────┴─────────┴──────┴───────┴──────┴──────┘
│                                                                       │
│  Rows per page: [20 ▼]                         Page 1 of 13  [<] [>] │
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding the Endpoint Table

| Column | Description |
|--------|-------------|
| **Hostname** | The device name. Click to view full details. |
| **OS** | Operating system with version. Shows a letter badge: **W** (Windows), **U** (Ubuntu), **R** (RHEL), **D** (Debian), **M** (macOS), **C** (CentOS), **F** (Fedora) |
| **Status** | Connection state: **Online** (green dot), **Offline** (red dot), **Pending** (blue dot), **Stale** (gray dot) |
| **Agent Ver** | The version of the Patch Manager agent running on the endpoint |
| **Risk Score** | 0-10 scale shown as "X/10". Color-coded: **Green** (0-3, low risk), **Yellow** (3-7, moderate), **Red** (7-10, high risk) |
| **Last Seen** | When the agent last reported in. "2m ago", "3 days ago", etc. |
| **Patches Pending** | Outstanding patches by severity. **C:3** = 3 critical, **H:2** = 2 high, **M:1** = 1 medium |
| **Tags** | Labels assigned to this endpoint for grouping and policy targeting |

### Stat Card Filters

The stat cards at the top work as **quick filters**. Click any card to filter the table:
- Click **Online** to see only online endpoints
- Click **Offline** to see only disconnected devices
- Click the active card again to clear the filter

### Expanded Row View

Click the expand chevron on any row to see a quick summary without leaving the page:

```
┌─────────────────────────────────────────────────────────────────────┐
│  SYSTEM HEALTH                    │  PENDING PATCHES               │
│                                   │                                │
│  CPU      ████████░░  72%        │  Critical    3  (red)          │
│  Memory   █████░░░░░  48%        │  High        2  (orange)       │
│  Disk     ██████████  95%        │  Medium      1  (yellow)       │
│                                   │                                │
│           [⎌ Deploy]  [Scan]  [View Details →]                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Row Actions (Menu)

Right-click or click the menu icon on any row to access:
- **View Details** — Open the full endpoint detail page
- **Scan** — Trigger an inventory scan
- **Deploy Patches** — Open a deployment targeting this endpoint
- **Assign Tags** — Apply tags to the endpoint
- **Delete** — Decommission the endpoint (shown in red)

### Endpoint Detail Page

Click any hostname to open the full detail view:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Endpoints                                                         │
│                                                                       │
│  web-server-01                     [Deploy Patches] [Scan Now] [···] │
│  ● Online  |  Ubuntu 22.04  |  Agent v1.2.3  |  192.168.1.100       │
│  Enrolled 30 days ago                                                 │
│                                                                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│  │ RISK SCORE  │ │ PATCH       │ │ COMPLIANCE  │ │ LAST SCAN   │    │
│  │             │ │ COVERAGE    │ │             │ │             │    │
│  │  7.2 /10   │ │    92%      │ │   3/5       │ │  2 hrs ago  │    │
│  │  ████████░░ │ │  █████████░ │ │  ██████░░░░ │ │             │    │
│  │   (red)     │ │  (green)    │ │  (yellow)   │ │             │    │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
│                                                                       │
│  [Overview] [Hardware] [Software] [Patches] [CVE Exposure]           │
│  [Deployments] [Audit]                                                │
│  ──────────────────────────────────────────────────────────────       │
│                                                                       │
│                    (Tab content appears here)                         │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### Endpoint Detail Tabs

| Tab | What You'll See |
|-----|----------------|
| **Overview** | Summary card with key metrics and endpoint information |
| **Hardware** | CPU model, cores, RAM size, disk layout, network info |
| **Software** | All installed packages/applications with versions |
| **Patches** | Patches applicable to this endpoint — deployed, pending, and failed |
| **CVE Exposure** | All CVEs affecting this endpoint, with severity, CVSS score, and exploit status |
| **Deployments** | History of all deployments targeting this endpoint |
| **Audit** | Complete activity log for this endpoint |

### Endpoint Actions

| Action | How to Access | What It Does |
|--------|---------------|-------------|
| **Deploy Patches** | Primary button in header | Opens deployment dialog pre-targeted to this endpoint |
| **Scan Now** | Outline button in header | Triggers an immediate inventory scan (shows "Scanning..." with spinner while running) |
| **Export Report** | More menu (...) | Downloads a report for this endpoint |
| **Delete Endpoint** | More menu (...) | Decommissions the endpoint. Shows a confirmation dialog: *"Are you sure you want to delete [hostname]? The endpoint will be marked as decommissioned."* |

---

## 6. Patch Catalog

### Browsing Available Patches

Navigate to **Patches** in the sidebar.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Patches                                                             │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  TOTAL   │ │ CRITICAL │ │   HIGH   │ │  MEDIUM  │ │   LOW    │  │
│  │    89    │ │    23    │ │    41    │ │    18    │ │     7    │  │
│  │          │ │  (red)   │ │ (orange) │ │ (yellow) │ │  (gray)  │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│                                                                       │
│  [Search patches (KB, USN, RHSA...)]     [OS Family ▼] [Status ▼]   │
│                                                                       │
│  ┌─┬───────────────────┬────────┬─────────┬──────┬──────┬─────────┬────────┬──────────┬────────┐
│  │☐│ Patch Name        │Vers/KB │Severity │  OS  │ CVEs │ CVSS Hi │Affected│Remediate%│Released│
│  ├─┼───────────────────┼────────┼─────────┼──────┼──────┼─────────┼────────┼──────────┼────────┤
│  │☐│ curl              │7.88.1  │CRITICAL │ U 22 │  4   │  9.8    │  142   │   12%    │2025-01 │
│  │ │                   │-10+deb │  (red)  │      │      │  ████   │        │  ██░░░░  │   -14  │
│  ├─┼───────────────────┼────────┼─────────┼──────┼──────┼─────────┼────────┼──────────┼────────┤
│  │☐│ KB5034441         │2024-01 │  HIGH   │ W 11 │  2   │  8.1    │   87   │   45%    │2025-01 │
│  │ │                   │        │(orange) │      │      │  ███░   │        │  ████░░  │   -10  │
│  └─┴───────────────────┴────────┴─────────┴──────┴──────┴─────────┴────────┴──────────┴────────┘
│                                                                       │
│  Status column shows: Pending | Deployed | Not Applicable             │
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding Patch Table Columns

| Column | Description |
|--------|-------------|
| **Patch Name** | Name of the patch or security update |
| **Version / KB** | Version number or KB article identifier |
| **Severity** | Risk level: **Critical** (red), **High** (orange), **Medium** (yellow), **Low** (gray) |
| **OS** | Target operating system (letter badge + version) |
| **CVE Count** | Number of CVEs addressed by this patch |
| **CVSS Highest** | Highest CVSS score among linked CVEs (0.0 - 10.0). Visual color bar indicates severity. |
| **Affected Endpoints** | How many of your endpoints need this patch |
| **Remediation %** | Progress bar showing what percentage of affected endpoints have been patched |
| **Released** | Date the vendor released this patch (YYYY-MM-DD) |
| **Status** | **Pending** (needs deployment), **Deployed** (100% remediated), **Not Applicable** (superseded or recalled) |

### Filters

- **Search** — Search by KB number, USN, RHSA, or name. Placeholder: *"Search patches (KB, USN, RHSA...)"*
- **OS Family** — Filter by operating system: Windows, Ubuntu, RHEL, Debian
- **Status** — Filter by status: available, superseded, recalled
- Click **Clear Filters** to reset all filters

### CVSS Score Color Scale

```
  0.0          4.0          7.0          9.0         10.0
   │            │            │            │            │
   ├────────────┼────────────┼────────────┼────────────┤
   │    LOW     │   MEDIUM   │    HIGH    │  CRITICAL  │
   │  (green)   │  (yellow)  │  (orange)  │   (red)    │
```

### Patch Detail Page

Click any patch name to see its full details:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Patches                                                           │
│                                                                       │
│  curl 7.88.1-10+deb12u5                                              │
│  CRITICAL  |  Debian 12  |  Released Jan 14, 2025                    │
│                                                                       │
│                       [Deploy] [Mark Reviewed] [···]                 │
│                                                                       │
│  [Overview] [CVEs (4)] [Affected Endpoints (142)]                    │
│  [Deployment History] [Remediation Metrics]                           │
│  ──────────────────────────────────────────────────────────────       │
│                                                                       │
│  Overview tab content:                                                │
│                                                                       │
│  Description:                                                         │
│  Security update for curl addressing buffer overflow vulnerability    │
│  in HTTP/2 header processing.                                        │
│                                                                       │
│  Highest CVSS Score: 9.8                                             │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  REMEDIATION STATUS                                          │    │
│  │                                                              │    │
│  │  Affected     Patched      Pending      Failed               │    │
│  │    150          80           50           20                  │    │
│  │  ████████   █████░░░░    ████░░░░░░   ██░░░░░░░░            │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  CVEs (4) tab shows linked vulnerabilities                            │
│  Affected Endpoints (142) tab shows endpoint list with patch status  │
│  Deployment History tab shows past rollout attempts                   │
│  Remediation Metrics tab shows detailed remediation analytics        │
└───────────────────────────────────────────────────────────────────────┘
```

### Patch Detail Tabs

| Tab | What You'll See |
|-----|----------------|
| **Overview** | Description, CVSS details, remediation status summary |
| **CVEs** | All CVEs addressed by this patch (count shown in tab label) |
| **Affected Endpoints** | Endpoints that need this patch (count shown in tab label), with hostname, OS, status, and patch status per endpoint |
| **Deployment History** | Timeline of all deployments of this patch with status, target count, success/failure counts |
| **Remediation Metrics** | Detailed remediation analytics |

### Patch Detail Actions

| Action | Description |
|--------|-------------|
| **Deploy** | Opens the deployment dialog with this patch pre-selected |
| **Mark Reviewed** | Marks the patch as reviewed (changes to "Reviewed" with a checkmark once clicked) |
| **Copy Patch ID** | Copies the patch UUID to clipboard (in the more menu) |
| **View in Patches List** | Navigates back to the patches list (in the more menu) |

---

## 7. Deploying Patches

Deploying patches is the core action in Patch Manager. There are multiple ways to initiate a deployment.

### Quick Deploy (From Patch Detail)

The fastest way to deploy a single patch:

1. Go to **Patches** and click a patch name
2. Click the **Deploy** button
3. Fill in the deployment dialog:

```
┌───────────────────────────────────────────────────────────────────┐
│  Create Patch Deployment                                      [X] │
│                                                                   │
│  Deployment Name *                                                │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │ e.g., KB5034441 - Critical Patch                          │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Description                                                      │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │ Optional: deployment notes, approval info...              │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Configuration Type *                                             │
│  (●) Install    ( ) Rollback                                      │
│                                                                   │
│  Target Endpoints *                                               │
│  ┌─────────────────────────────────────┐                          │
│  │ Select endpoints              [▼]   │                          │
│  │ ─────────────────────────────────── │                          │
│  │  All Endpoints                      │                          │
│  │  Windows Only                       │                          │
│  │  Linux Only                         │                          │
│  │  Critical Endpoints                 │                          │
│  └─────────────────────────────────────┘                          │
│                                                                   │
│  Schedule Deployment (Optional)                                   │
│  ┌──────────────────┐  ┌──────────────────┐                      │
│  │ Start Date       │  │ Start Time       │                      │
│  │ 2025-01-20       │  │ 02:00 AM         │                      │
│  └──────────────────┘  └──────────────────┘                      │
│                                                                   │
│  Patches to Deploy:                                               │
│  ┌────────────────────┬──────────────────┬───────────┐           │
│  │ ID                 │ Version          │ Severity  │           │
│  ├────────────────────┼──────────────────┼───────────┤           │
│  │ curl               │ 7.88.1-10+deb12  │ CRITICAL  │           │
│  └────────────────────┴──────────────────┴───────────┘           │
│                                                                   │
│                    [Cancel]  [Save as Draft]  [Publish]           │
└───────────────────────────────────────────────────────────────────┘
```

4. Click **Publish** to start the deployment immediately, or **Save as Draft** to review later

> **Note:** The Deployment Name field is required (marked with *). The Publish button is disabled until a name is provided.

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
2. Patch Manager monitors the success rate
3. If the success rate meets the threshold (e.g., 95%), it waits the configured delay (e.g., 30 minutes)
4. **Wave 2** deploys to the next batch (e.g., 30%)
5. This continues until all waves complete
6. If any wave's failure rate exceeds the maximum error rate, Patch Manager **automatically rolls back** the entire deployment

### What Happens During a Deployment

Behind the scenes, this is what Patch Manager does:

```
YOU click "Publish"
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
│                   │  b) Verifies SHA256 checksum
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
│  ┌───────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │TOTAL DEPLOYMTS│ │ RUNNING  │ │COMPLETED │ │  FAILED  │          │
│  │     53        │ │    3     │ │    47    │ │    2     │          │
│  └───────────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                                                       │
│  Filter: [All] [Running] [Completed] [Failed] [Created]              │
│          [Scheduled] [Cancelled]                                      │
│                                                                       │
│  ┌──────────┬────────────┬──────────────┬──────────┬──────────┬─────┐│
│  │ Name     │ Status     │ Target Count │ Duration │ Created  │ By  ││
│  ├──────────┼────────────┼──────────────┼──────────┼──────────┼─────┤│
│  │ curl     │ ● Running  │ ████░░░░     │ 12 min   │ 10:30 AM │admin││
│  │ deploy   │   (green)  │ 42/100       │          │          │     ││
│  ├──────────┼────────────┼──────────────┼──────────┼──────────┼─────┤│
│  │ KB5034   │ ✓ Completed│ ████████     │ 45 min   │ 09:00 AM │sys  ││
│  │ rollout  │   (green)  │ 87/87        │          │          │     ││
│  ├──────────┼────────────┼──────────────┼──────────┼──────────┼─────┤│
│  │ openssl  │ ✗ Failed   │ ██████░░     │ 20 min   │ 08:15 AM │admin││
│  │ update   │   (red)    │ 28/34 (6fail)│          │          │     ││
│  └──────────┴────────────┴──────────────┴──────────┴──────────┴─────┘│
└───────────────────────────────────────────────────────────────────────┘
```

### Deployment Status Colors

| Status | Color | Indicator | Meaning |
|--------|-------|-----------|---------|
| **Running** | Green | Pulsing dot | Actively deploying to endpoints |
| **Completed** | Green | Static | All waves succeeded |
| **Failed** | Red | Static | Error threshold exceeded |
| **Rollback Failed** | Red | Static | Rollback could not complete |
| **Rolling Back** | Orange | Pulsing dot | Undoing installed patches |
| **Rolled Back** | Muted | Static | Rollback completed successfully |
| **Scheduled** | Muted | Static | Waiting for scheduled time |
| **Created** | Muted | Static | Just created, about to start |
| **Cancelled** | Muted | Static | Manually cancelled by user |

### Target Count Progress Bar

The Target Count column shows a segmented progress bar:
- **Green** = Succeeded endpoints
- **Orange** = Active (currently installing)
- **Red** = Failed endpoints
- **Gray** = Pending (not yet started)

### Deployment Detail Page

Click a deployment to see the full detail view:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← Deployments                                                       │
│                                                                       │
│  curl 7.88.1 - Deployment                                            │
│  ● RUNNING  |  Started 12 min ago  |  By: admin                     │
│                                                                       │
│                    [Cancel] [Retry] [Rollback] [···]                 │
│                                                                       │
│  [Overview] [Progress] [Targets] [Patches] [History]                 │
│  ──────────────────────────────────────────────────────────────       │
│                                                                       │
│  Progress tab — Wave Pipeline:                                        │
│                                                                       │
│  Wave 1 (10%)           Wave 2 (30%)          Wave 3 (60%)           │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐       │
│  │ ✓ Completed  │ ──→  │ ▶ In Progress│ ──→  │ ◼ Waiting    │       │
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
│  Targets tab — Endpoint Status:                                       │
│                                                                       │
│  ┌──────────────┬────────┬──────────────┬──────────────────────┐     │
│  │ Hostname     │ Wave   │ Status       │ Details              │     │
│  ├──────────────┼────────┼──────────────┼──────────────────────┤     │
│  │ web-srv-01   │ Wave 1 │ ✓ Succeeded  │ Installed in 45s     │     │
│  │ web-srv-02   │ Wave 1 │ ✓ Succeeded  │ Installed in 32s     │     │
│  │ api-srv-01   │ Wave 2 │ ✗ Failed     │ Exit code 1: dep err │     │
│  │ api-srv-02   │ Wave 2 │ ⟳ Executing  │ Installing...        │     │
│  │ db-srv-01    │ Wave 3 │ ◼ Pending    │ Waiting for Wave 3   │     │
│  └──────────────┴────────┴──────────────┴──────────────────────┘     │
└───────────────────────────────────────────────────────────────────────┘
```

### Deployment Detail Tabs

| Tab | What You'll See |
|-----|----------------|
| **Overview** | Summary metrics, source policy/patch, scope, configuration |
| **Progress** | Wave pipeline visualization showing wave status and per-endpoint progress |
| **Targets** | Per-endpoint status table with hostname, wave, status, and error details |
| **Patches** | List of patches being deployed |
| **History** | Chronological timeline of deployment events |

### Wave Status Indicators

| Indicator | Meaning |
|-----------|---------|
| ✓ Completed | Wave finished successfully |
| ▶ In Progress | Wave is actively deploying |
| ✗ Failed | Wave failed (error threshold exceeded) |
| ◼ Waiting | Wave hasn't started yet |

### Target Status Indicators

| Indicator | Meaning |
|-----------|---------|
| ◼ Pending | Waiting to be dispatched |
| ⟳ Sent | Command sent to agent |
| ⟳ Executing | Agent is installing (pulsing animation) |
| ✓ Succeeded | Patch installed successfully |
| ✗ Failed | Installation failed |

### Deployment Actions

| Action | When Available | What It Does |
|--------|---------------|-------------|
| **Cancel** | While Running or Created | Immediately stops the deployment. Pending targets are cancelled. Already-installed patches remain. |
| **Retry** | After deployment has Failed | Resets all failed targets back to "pending" and restarts the deployment. Only failed endpoints are retried. |
| **Rollback** | While Running or after Completion/Failure | Sends rollback commands to endpoints that already installed the patch, reverting them to their previous version. |

### Understanding Rollback

When you trigger a rollback (or it happens automatically):

```
ROLLBACK PROCESS
─────────────────

1. Deployment status changes to "Rolling Back"
   (pulsing orange indicator)

2. All remaining waves are CANCELLED
   (endpoints that haven't received the patch yet are safe)

3. All pending commands are CANCELLED

4. For endpoints that already installed the patch:
   → Rollback command sent to each agent
   → Agent reverts to the previous package version
   → Agent reports success/failure

5. If all rollbacks succeed:
   → Status: "Rolled Back" (muted)

6. If any rollback fails:
   → Status: "Rollback Failed" (red)
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
│  Severity:  [Critical] [High] [Medium] [Low]                        │
│  Also:      [Exploitable] [KEV (CISA)] [Has Patches]                │
│  View:      [Table | Cards]                                          │
│                                                                       │
│  ┌──────────────┬──────┬──────────┬────────┬──────┬─────┬─────────┬─────────┬────────┬─────────┐
│  │ CVE ID       │ CVSS │ Severity │ Attack │Explt │ KEV │Aff. Pkgs│Aff Endpt│Patches │Published│
│  ├──────────────┼──────┼──────────┼────────┼──────┼─────┼─────────┼─────────┼────────┼─────────┤
│  │CVE-2024-1234 │  9.8 │ CRITICAL │  N     │ Yes  │ Yes │   4     │  142    │Availble│2024-12  │
│  │CVE-2024-5678 │  9.1 │ CRITICAL │  N     │ No   │ Yes │   2     │   87    │Availble│2024-11  │
│  │CVE-2024-9012 │  8.8 │  HIGH    │  N     │ POC  │ No  │   1     │   34    │  —     │2024-10  │
│  │CVE-2024-3456 │  7.5 │  HIGH    │  L     │ No   │ No  │   1     │   12    │Availble│2024-09  │
│  └──────────────┴──────┴──────────┴────────┴──────┴─────┴─────────┴─────────┴────────┴─────────┘
└───────────────────────────────────────────────────────────────────────┘
```

### Understanding CVE Columns

| Column | Description |
|--------|-------------|
| **CVE ID** | The unique vulnerability identifier (e.g., CVE-2024-1234). Click to view details. |
| **CVSS Score** | Severity score from 0.0-10.0 with a visual color bar |
| **Severity** | Derived from CVSS: Critical (9.0+), High (7.0-8.9), Medium (4.0-6.9), Low (0-3.9) |
| **Attack Vector** | **N** (Network — remote), **A** (Adjacent — local network), **L** (Local — physical access), **P** (Physical) |
| **Exploit** | Whether a known exploit exists: **Yes** (actively exploited), **POC** (proof of concept), or dash |
| **KEV** | Whether CISA lists this as a Known Exploited Vulnerability, with due date if applicable |
| **Affected Pkgs** | Number of affected packages |
| **Affected Endpoints** | Number of your endpoints vulnerable to this CVE |
| **Patches** | **Available** (green badge) if a fix exists, or a dash if no patch yet |
| **Published** | When the CVE was published |

### CVE Filter Options

| Filter Group | Options |
|-------------|---------|
| **Severity** | Critical, High, Medium, Low (each with count and color) |
| **Availability** | Exploitable, KEV (CISA), Has Patches |
| **View** | Table view or Card view toggle |

### CVE Detail Page

Click a CVE ID to see the full breakdown:

```
┌───────────────────────────────────────────────────────────────────────┐
│  ← CVEs                                                              │
│                                                                       │
│  CVE-2024-1234                                                       │
│  CRITICAL  |  CVSS 9.8  |  Published Dec 1, 2024                    │
│                                                                       │
│  Buffer overflow in curl HTTP/2 header processing allows remote      │
│  code execution via crafted HTTP/2 HEADERS frame.                    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  CVSS v3.1 BREAKDOWN                                         │    │
│  │                                                              │    │
│  │  Attack Vector:     Network     │ Scope:       Unchanged    │    │
│  │  Attack Complexity: Low         │ Confidentiality: High     │    │
│  │  Privileges Req:    None        │ Integrity:      High      │    │
│  │  User Interaction:  None        │ Availability:   High      │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  AFFECTED ENDPOINTS                                                   │
│  ┌──────────────┬──────────┬──────────────┬────────────────────┐     │
│  │ Hostname     │ Status   │ Patch Status │ Tags               │     │
│  ├──────────────┼──────────┼──────────────┼────────────────────┤     │
│  │ web-srv-01   │ Online   │ Affected     │ prod, web          │     │
│  │ web-srv-02   │ Online   │ Patched      │ prod, web          │     │
│  │ api-srv-01   │ Offline  │ Affected     │ prod, api          │     │
│  └──────────────┴──────────┴──────────────┴────────────────────┘     │
│                                                                       │
│  RELATED PACKAGES                                                     │
│  ┌──────────────────────┬──────────────┬───────────────────────┐     │
│  │ Package Name         │ Version      │ Available Patches     │     │
│  ├──────────────────────┼──────────────┼───────────────────────┤     │
│  │ curl                 │ 7.87.0       │ curl 7.88.1-10+deb12  │     │
│  └──────────────────────┴──────────────┴───────────────────────┘     │
└───────────────────────────────────────────────────────────────────────┘
```

### Endpoint Remediation Status in CVE Detail

| Status | Color | Meaning |
|--------|-------|---------|
| **Patched** | Green | CVE has been remediated on this endpoint |
| **Affected** | Red | Endpoint is still vulnerable |
| **Mitigated** | Orange | Workaround applied but not fully patched |

---

## 10. Policies - Automated Patch Management

Policies let you define rules for automatic patch deployment. Instead of manually deploying every patch, you create policies that handle it for you.

### Policy Types

Before choosing a mode, you first select the policy type:

| Type | Color | Description |
|------|-------|-------------|
| **Patch Policy** | Accent (blue) | Select patches by severity, CVE, or regex. Evaluate and optionally auto-deploy. |
| **Deploy Policy** | Green | Target specific updates for direct deployment to endpoints. |
| **Compliance Policy** | Muted (gray) | Evaluate patch compliance on a schedule. Report only, no deployments. |

### Policy Modes

Each policy type supports different modes:

| Mode | Available For | Behavior |
|------|-------------|----------|
| **Automatic** | Patch, Deploy | Evaluates on schedule. Matching patches deploy automatically within the maintenance window. |
| **Manual** | Patch, Deploy | Evaluates on schedule. Patches are queued but NOT deployed until you click Deploy. |
| **Advisory** | Patch, Compliance | Evaluates on schedule. Reports compliance status only. No patches are ever deployed. |

### Creating a Policy

Navigate to **Policies** and click **+ New Policy**:

```
STEP 1: POLICY TYPE
═══════════════════════════════════════════════════════════════

  Select the type of policy:

  ┌─────────────────────────────────────────────────────┐
  │ (●) PATCH POLICY                           (blue)   │
  │     Select patches by severity, CVE, or regex.      │
  │     Evaluate and optionally auto-deploy.             │
  ├─────────────────────────────────────────────────────┤
  │ ( ) DEPLOY POLICY                          (green)  │
  │     Target specific updates for direct               │
  │     deployment to endpoints.                         │
  ├─────────────────────────────────────────────────────┤
  │ ( ) COMPLIANCE POLICY                      (gray)   │
  │     Evaluate patch compliance on a schedule.         │
  │     Report only, no deployments.                     │
  └─────────────────────────────────────────────────────┘


STEP 2: BASICS & MODE
═══════════════════════════════════════════════════════════════

  Name *
  ┌──────────────────────────────────────────────────────┐
  │ Critical Linux Security Updates                      │
  └──────────────────────────────────────────────────────┘

  Description
  ┌──────────────────────────────────────────────────────┐
  │ Auto-deploy critical security patches to all         │
  │ production Linux servers.                            │
  └──────────────────────────────────────────────────────┘

  Mode:
  (●) Automatic — Evaluates on schedule. Matching patches
                   deploy automatically within the maintenance
                   window.
  ( ) Manual    — Evaluates on schedule. Patches are queued
                   but NOT deployed until you click Deploy.
  ( ) Advisory  — Reports compliance status only. No patches
                   are ever deployed.


STEP 3: PATCH SELECTION (Patch Policy only)
═══════════════════════════════════════════════════════════════

  How should patches be selected?

  ( ) All Available Patches
  (●) By Severity
      Minimum Severity: [Critical ▼]  (Critical/High/Medium/Low)
  ( ) By CVE List
      CVE IDs: [CVE-2024-1234, CVE-2024-5678, ...]
  ( ) By Package Regex
      Pattern: [^curl.*]
      Exclude: [libcurl-doc, ...]


STEP 4: TARGET ENDPOINTS
═══════════════════════════════════════════════════════════════

  Tag Selector (define which endpoints this policy targets):
  ┌──────────────────────────────────────────────────────┐
  │  environment = "production"  AND  os = "linux"       │
  └──────────────────────────────────────────────────────┘


STEP 5: SCHEDULE
═══════════════════════════════════════════════════════════════

  ( ) Manual — trigger on demand
  (●) Recurring — run on cron schedule

  Preset:  ( ) Daily  (●) Weekly  ( ) Monthly  ( ) Custom

  Day of week: [Sunday ▼]
  Time: [02:00 AM ▼]
  Timezone: [UTC ▼]

  Next 3 runs:
    - Sun, Jan 19, 2025, 2:00 AM UTC
    - Sun, Jan 26, 2025, 2:00 AM UTC
    - Sun, Feb 2, 2025, 2:00 AM UTC

  Maintenance Window:
  [✓] Enable maintenance window
      Start: [01:00 AM]    End: [05:00 AM]

                                              [Create Policy]
```

### Policy Detail Page

After creating a policy, its detail page shows:

| Tab | Content |
|-----|---------|
| **Overview** | Policy name, mode (color-coded), type, created/updated dates, status |
| **Patch Scope** | Severity filters, OS filters, selection criteria, matching patch count |
| **Groups & Endpoints** | Tag selector expression, matched endpoint count, endpoint list |
| **Evaluation History** | When policy was last evaluated, how many endpoints/patches matched |
| **Deployment History** | All deployments triggered by this policy with status and results |
| **Schedule** | Cron expression, next run times, timezone, maintenance window |

### Policy Actions

| Action | Icon | Description |
|--------|------|-------------|
| **Edit** | Pencil | Modify the policy settings |
| **Evaluate** | Refresh | Run the policy evaluation immediately |
| **Deploy** | Rocket | Manually trigger a deployment using this policy's criteria |
| **Copy** | Copy | Duplicate the policy |
| **Delete** | Trash (red) | Remove the policy |

---

## 11. Compliance Management

Patch Manager evaluates your infrastructure against industry security frameworks.

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

Navigate to **Reports > Compliance** or the Compliance section:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Compliance                                                           │
│  Security framework evaluation and tracking                          │
│                                                                       │
│                [Manage Frameworks] [Evaluate All] [Export Report]     │
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
│  │  Score: 82%   │ │  Score: 95%   │ │  Score: 74%   │ │Score:85% │ │
│  │  ████████░░   │ │  █████████░   │ │  ███████░░░   │ │████████░ │ │
│  │  42/50 ctrls  │ │  28/30 ctrls  │ │  37/50 ctrls  │ │34/40 ctl │ │
│  │  Last: 2h ago │ │  Last: 2h ago │ │  Last: 2h ago │ │Last:2h   │ │
│  │  [Evaluate]   │ │  [Evaluate]   │ │  [Evaluate]   │ │[Evaluate]│ │
│  └───────────────┘ └───────────────┘ └───────────────┘ └──────────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

### Compliance Actions

| Action | Description |
|--------|-------------|
| **Manage Frameworks** | Enable/disable frameworks, configure custom frameworks |
| **Evaluate All** | Trigger immediate evaluation across all frameworks (button shows "Evaluating..." with spinner while running) |
| **Export Report** | Download compliance report (coming soon) |
| **Evaluate** (per card) | Evaluate a single framework |

---

## 12. Alerts & Notifications

### Alerts Page

Navigate to **Alerts** in the sidebar. A red or orange badge on the sidebar shows unread alert count.

```
┌───────────────────────────────────────────────────────────────────────┐
│  Alerts                                          [Alert Rules]       │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │  TOTAL   │ │ CRITICAL │ │ WARNING  │ │   INFO   │               │
│  │    14    │ │    2     │ │    5     │ │    7     │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  Status: [Active] [Acknowledged] [Dismissed] [All]                   │
│  Category: [All] [Deployments] [Agents] [CVEs] [Compliance] [System]│
│  Date: [Last 24h ▼]                                                  │
│                                                                       │
│  ┌──────────┬────────────────────────────────────────┬──────────┬────┐│
│  │ Severity │ Title                                  │ Status   │Time││
│  ├──────────┼────────────────────────────────────────┼──────────┼────┤│
│  │ CRITICAL │ Deployment "KB5034" failed on 6        │ Active   │ 5m ││
│  │  (red)   │ endpoints. Error rate exceeded 20%.    │          │ago ││
│  ├──────────┼────────────────────────────────────────┼──────────┼────┤│
│  │ WARNING  │ 3 endpoints offline for >24 hours.     │ Active   │ 2h ││
│  │ (orange) │ Last heartbeat: db-srv-03, web-04...   │          │ago ││
│  ├──────────┼────────────────────────────────────────┼──────────┼────┤│
│  │ INFO     │ Catalog sync completed. 5 new patches. │Dismissed │ 4h ││
│  │ (blue)   │                                        │          │ago ││
│  └──────────┴────────────────────────────────────────┴──────────┴────┘│
└───────────────────────────────────────────────────────────────────────┘
```

### Alert Filters

| Filter | Options |
|--------|---------|
| **Status** | Active, Acknowledged, Dismissed, All |
| **Category** | All, Deployments, Agents, CVEs, Compliance, System |
| **Date Range** | Last 24h, Last 7 days, Last 30 days, Custom Range |

### Alert Actions

- **Mark Read** — Acknowledge you've seen the alert
- **Acknowledge** — Confirm you're working on it
- **Dismiss** — Remove the alert from the active list

### Configuring Notification Channels

Navigate to **Settings > Notifications**:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Notifications                                                        │
│                                                                       │
│  [Preferences]  [History]                                             │
│  ──────────────────────────────────────────────────────────────       │
│                                                                       │
│  Notification Channels                                                │
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
│  Digest Configuration                                                 │
│  Configure when and how digest notifications are delivered            │
│                                                                       │
│  Frequency: [Daily ▼]   Delivery Time (UTC): [08:00 AM ▼]           │
│  Format: [HTML ▼]                             [Send Test Digest]      │
└───────────────────────────────────────────────────────────────────────┘
```

### Notification Channel Setup

Click **Set up** on any unconfigured channel:

| Field | Description |
|-------|-------------|
| **Channel Name** | A display name for this channel |
| **Shoutrrr URL** | The notification URL (SMTP for email, webhook URL for Slack/Discord/Webhook) |

Supported channel types: **Email**, **Slack**, **Webhook**, **Discord**

### Notification Tabs

| Tab | Content |
|-----|---------|
| **Preferences** | Channel configuration, per-event toggle switches, digest settings |
| **History** | Log of all notifications sent — channel, event type, recipient, status, time, and resend option |

---

## 13. Audit Log

The Audit Log is an **immutable record** of every action taken in Patch Manager. It cannot be edited or deleted.

Navigate to **Audit** in the sidebar:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Audit                                                                │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │  TOTAL   │ │  SYSTEM  │ │   USER   │ │  TODAY   │               │
│  │  12,847  │ │  8,234   │ │  4,613   │ │   342    │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  View: [Activity Stream]  [Timeline View]                             │
│                                                                       │
│  Filters:                                                             │
│  Event: [All Event Types ▼]  (Endpoint/Patch/Deployment/Policy/      │
│                                Compliance/Auth/System)                │
│  Actor: [Search by actor...]                                          │
│  Resource: [All Resources ▼]  (Endpoints/Deployments/Policies/       │
│                                 Settings)                             │
│  Date: [Last 24h ▼]  (Last 24h / Last 7 days / Last 30 days /       │
│                        Custom Range)                                  │
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
│  Click any row to expand and see the full event details.             │
│                                                                       │
│  Audit logs retained for 365 days.         [Export as CSV]            │
│  Oldest entry: March 12, 2025.             [Export as JSON]           │
│  Manage Retention Policy                                              │
└───────────────────────────────────────────────────────────────────────┘
```

### Audit View Modes

| Mode | Description |
|------|-------------|
| **Activity Stream** | Chronological list of events with expandable details |
| **Timeline View** | Visual timeline grouped by time period |

### Audit Event Type Filters

| Event Type | What It Covers |
|-----------|----------------|
| **Endpoint** | Created, updated, deleted, enrolled, scanned |
| **Patch** | Discovered, synced from Hub |
| **Deployment** | Created, started, completed, failed, cancelled, rolled back |
| **Policy** | Created, updated, deleted, evaluated, auto-deployed |
| **Compliance** | Framework enabled, evaluation completed, threshold breach |
| **Auth** | Login, logout, user provisioned, role assigned |
| **System** | Settings changed, license events, notification events |

---

## 14. Reports

### Generating Reports

Navigate to **Reports** in the sidebar:

```
┌───────────────────────────────────────────────────────────────────────┐
│  Reports                                         [+ Generate Report] │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │   ALL    │ │COMPLETED │ │GENERATING│ │  FAILED  │ │  TODAY   │  │
│  │    15    │ │    12    │ │    1     │ │    0     │ │    2     │  │
│  │          │ │ (green)  │ │(pulsing) │ │  (red)   │ │          │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│                                                                       │
│  Type: [All Types ▼]    Format: [All Formats ▼]                      │
│                                                                       │
│  ┌──────────────────┬─────────────┬──────┬───────────┬──────┬──────┐ │
│  │ Report Name      │ Type        │Format│ Status    │ Size │ Date │ │
│  ├──────────────────┼─────────────┼──────┼───────────┼──────┼──────┤ │
│  │ Q1 Patch Status  │ PATCHES     │ PDF  │✓ Complete │ 2.4MB│ Jan 1│ │
│  │ HIPAA Compliance │ COMPLIANCE  │ XLSX │✓ Complete │ 1.1MB│ Dec 1│ │
│  │ Monthly Vulns    │ CVES        │ CSV  │● Generating│  — │ Now  │ │
│  └──────────────────┴─────────────┴──────┴───────────┴──────┴──────┘ │
└───────────────────────────────────────────────────────────────────────┘
```

### Report Types

| Type | Description |
|------|-------------|
| **Endpoints** | Endpoint inventory and status report |
| **Patches** | Patch catalog and deployment status |
| **CVEs** | Vulnerability exposure report |
| **Deployments** | Deployment history and results |
| **Compliance** | Compliance framework scores and control status |
| **Executive** | High-level summary for management |

### Report Formats

| Format | Badge Color | Use Case |
|--------|-------------|----------|
| **PDF** | Blue | Formatted reports for sharing with stakeholders |
| **CSV** | Green | Raw data for analysis in spreadsheets |
| **XLSX** | Yellow | Excel format with formatting |

### Generate Report Dialog

Click **+ Generate Report** and configure:

| Field | Description |
|-------|-------------|
| **Report Type** | Select from the 6 types above |
| **Format** | Choose PDF, CSV, or XLSX |
| **Type-specific filters** | Varies by report type: severity, OS family, status, framework, date range, exploit/KEV status |

Click **Generate** and the report appears in the list. Reports download automatically when ready.

---

## 15. Workflows

Workflows let you build custom automation using a visual drag-and-drop canvas.

Navigate to **Workflows** (if available in your sidebar):

```
┌───────────────────────────────────────────────────────────────────────┐
│  Workflows                                       [+ Create Workflow] │
│                                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │  TOTAL   │ │PUBLISHED │ │  DRAFT   │ │ ARCHIVED │               │
│  │    5     │ │    3     │ │    1     │ │    1     │               │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │
│                                                                       │
│  [Search workflows...]                                                │
│  [Published] [Draft] [Archived]                                       │
│                                                                       │
│  ┌──────────────┬──────────┬───────┬──────┬──────────┬────────┬─────┐│
│  │ Name         │ Status   │ Nodes │ Runs │ Last Run │Updated │ Act ││
│  ├──────────────┼──────────┼───────┼──────┼──────────┼────────┼─────┤│
│  │ Auto-patch   │Published │   5   │  12  │ 2h ago   │ Jan 15 │[▶][✎]│
│  │ critical     │          │       │      │          │        │     ││
│  │ CVE response │ Draft    │   3   │   0  │ Never    │ Jan 14 │[✎]  ││
│  └──────────────┴──────────┴───────┴──────┴──────────┴────────┴─────┘│
└───────────────────────────────────────────────────────────────────────┘
```

### Workflow Editor

Click **Create Workflow** or **Edit** to open the visual editor:

```
┌───────────────────────────────────────────────────────────────────────┐
│  My Workflow                    [Saved ✓]  [?] [↩] [↪] [⊞] [Save]  │
│                                                                       │
│  ┌──────────┐  ┌────────────────────────────────────────────────────┐│
│  │ PALETTE  │  │                                                    ││
│  │          │  │     ┌──────────┐                                   ││
│  │ Triggers │  │     │ TRIGGER  │                                   ││
│  │ ├ Event  │  │     │ On CVE   │                                   ││
│  │ ├ Sched  │  │     │ Critical │                                   ││
│  │          │  │     └────┬─────┘                                   ││
│  │ Actions  │  │          │                                         ││
│  │ ├ Deploy │  │     ┌────▼─────┐                                   ││
│  │ ├ Script │  │     │ DECISION │                                   ││
│  │ ├ Notify │  │     │ Is prod? │                                   ││
│  │ ├ Scan   │  │     └──┬────┬──┘                                   ││
│  │          │  │   Yes  │    │  No                                  ││
│  │ Control  │  │   ┌────▼──┐ ┌──▼────┐                             ││
│  │ ├ Branch │  │   │DEPLOY │ │NOTIFY │                             ││
│  │ ├ Delay  │  │   │Waves  │ │Slack  │                             ││
│  │ ├ End    │  │   └───────┘ └───────┘                             ││
│  │          │  │                                                    ││
│  └──────────┘  └────────────────────────────────────────────────────┘│
│  5 nodes, 4 connections                               Valid ✓       │
└───────────────────────────────────────────────────────────────────────┘
```

### Workflow Toolbar

| Button | Shortcut | Action |
|--------|----------|--------|
| **?** | — | Show keyboard shortcuts help |
| **↩** | Ctrl+Z | Undo last change |
| **↪** | Ctrl+Shift+Z | Redo |
| **⊞** | — | Auto-layout nodes |
| **Save** | — | Save workflow |
| **Publish** | — | Publish workflow (makes it active) |

### Workflow Status Bar

The bottom of the editor shows: *"X nodes, Y connections"* and whether the workflow is **Valid** or has **Errors**.

---

## 16. Settings & Administration

### Settings Navigation

Navigate to **Settings** in the sidebar. The settings page has its own sidebar with 4 groups:

```
┌──────────────────────┐
│ SETTINGS             │
│                      │
│ Configuration        │
│ ├ General            │  Organization name, timezone, date format,
│ │                    │  default scan interval
│ ├ Identity & Access  │  OIDC/SSO configuration (Zitadel)
│ │                    │  Shows connection status indicator
│ ├ Patch Sources      │  Repository URLs, credentials, sync schedules
│ ├ Agent Fleet        │  Agent download center, available binaries
│ └ Notifications      │  Email, Slack, Discord, webhook channels
│                      │
│ Account              │
│ ├ My Account         │  Profile, password change, API tokens
│ ├ License            │  License tier, seats, expiration
│ └ Appearance         │  Theme (light/dark), density
│                      │
│ Administration       │
│ ├ Tags               │  Create/manage endpoint tags
│ ├ Roles              │  RBAC role definitions with permission matrix
│ └ User Roles         │  Assign roles to users
│                      │
│ System               │
│ └ About              │  Version info, build details
└──────────────────────┘
```

### General Settings

| Field | Description |
|-------|-------------|
| **Organization Name** | Displayed in reports, notifications, and the dashboard header |
| **Timezone** | System timezone (UTC, America/New_York, Europe/London, Asia/Tokyo, etc.) |
| **Date Format** | YYYY-MM-DD, MM/DD/YYYY, DD/MM/YYYY, DD MMM YYYY |
| **Default Scan Interval** | How often agents scan (1, 2, 4, 6, 12, or 24 hours). Overridden by per-endpoint settings. |

### Role-Based Access Control (RBAC)

Navigate to **Settings > Roles** to manage permissions:

```
Permission Matrix for a Role:
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
│ Alerts            │  [✓]  │  [○]   │  [✓]   │  [○]   │
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
Go to CVEs            →   Filter: Critical      →   Click CVE to see
                          + Exploitable               affected endpoints

STEP 4                    STEP 5                    STEP 6
Go to Patches         →   Find the fix patch    →   Click "Deploy"
(or from CVE detail)       (check "Patches"           on the patch
                          column = "Available")

STEP 7                    STEP 8                    STEP 9
Fill deployment       →   Target: All or select  →   Click "Publish"
name                       specific endpoints

STEP 10                   STEP 11                   STEP 12
Monitor on            →   Check per-endpoint     →   If any fail:
Deployments page          status in detail view       Click "Retry"
```

**Visual flow:**

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  CVEs   │ ──→ │  Find   │ ──→ │ Deploy  │ ──→ │ Monitor │
│  page   │     │  patch  │     │  patch  │     │Progress │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
```

### Workflow 2: Setting Up Automated Patching

**Scenario:** You want critical and high-severity patches deployed automatically every Sunday at 2 AM to production servers.

```
STEP 1                    STEP 2                    STEP 3
Go to Policies        →   Click "+ New Policy"   →   Type: Patch Policy
                                                      Mode: Automatic

STEP 4                    STEP 5                    STEP 6
Name: "Prod           →   Patch Selection:       →   Target Endpoints:
Auto-Patch"                By Severity                Tag selector:
                          Min: Critical               env = production
                          (includes Critical           AND os = linux
                           + High)

STEP 7                    STEP 8                    STEP 9
Schedule:             →   Maintenance Window:    →   Review & click
Recurring, Weekly          Enable                     "Create Policy"
Sunday, 2:00 AM UTC        Start: 1:00 AM
                          End: 5:00 AM

STEP 10
Policy runs automatically every Sunday at 2 AM.
Check Deployments page Monday morning.
```

### Workflow 3: Onboarding New Endpoints

**Scenario:** You're adding 50 new servers to Patch Manager.

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
(select all → Scan)        new endpoints on
                          next scheduled run
```

### Workflow 4: Investigating a Failed Deployment

**Scenario:** A deployment shows "Failed" status. You need to find out why and fix it.

```
STEP 1                    STEP 2                    STEP 3
Go to Deployments     →   Click the failed        →   Go to "Targets" tab
                          deployment

STEP 4                    STEP 5                    STEP 6
Find endpoints with   →   Check error messages    →   Fix the root cause
"✗ Failed" status          (stdout/stderr shown)       on affected endpoints

STEP 7
Go back to deployment
and click "Retry"

                          ┌─────────────────────┐
                          │  Common Causes:      │
                          │                      │
                          │  - Disk full         │
                          │  - Dependency        │
                          │    conflict          │
                          │  - Agent offline     │
                          │  - Maintenance       │
                          │    window closed     │
                          │  - Permission denied │
                          └──────────────────────┘
```

### Workflow 5: Generating a Compliance Report for Auditors

```
STEP 1                    STEP 2                    STEP 3
Go to Compliance      →   Review framework        →   Click "Evaluate All"
                          scores                       to refresh scores

STEP 4                    STEP 5                    STEP 6
Go to Reports         →   Click "+ Generate       →   Configure:
                          Report"                      Type: Compliance
                                                       Framework: HIPAA (*)
                                                       Format: PDF

STEP 7                    STEP 8
Report generates      →   Download and share
(status shows              with auditors
"Generating" with
pulsing indicator)
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
| Has the max concurrent limit been reached? | Check deployment configuration |
| Are commands timing out? | Default timeout is 30 minutes per endpoint. Check agent logs. |

### Patches Not Appearing in Catalog

| Check | Solution |
|-------|----------|
| Is Hub sync working? | Check Settings for last catalog sync time |
| Is the patch for your OS? | Patches are OS-specific. Verify OS family matches. |
| Has the patch been recalled? | Recalled patches have "Not Applicable" status |

### Agent Showing "Stale" Status

The agent hasn't sent a heartbeat recently. Common causes:
- Endpoint is powered off or unreachable
- Agent service crashed — restart it with `patchiq-agent start`
- Network firewall blocking gRPC port 50051
- Agent needs updating to latest version

### Deployment Auto-Rolled Back

If a deployment automatically rolled back, it means the wave's failure rate exceeded the configured error rate maximum. Check the deployment detail for:
- Which endpoints failed and why (Targets tab)
- The wave configuration thresholds (Overview tab)
- Whether the issue is systemic (same error on all failed endpoints) or isolated

---

## 19. Glossary

| Term | Definition |
|------|-----------|
| **Agent** | The Patch Manager software installed on managed endpoints that reports inventory and executes patch commands |
| **Attack Vector** | How a vulnerability can be exploited: Network (N), Adjacent (A), Local (L), Physical (P) |
| **CISA KEV** | Cybersecurity and Infrastructure Security Agency's Known Exploited Vulnerabilities catalog — vulnerabilities confirmed to be actively exploited |
| **Compliance Framework** | A set of security controls and benchmarks (CIS, PCI-DSS, HIPAA, NIST, ISO 27001, SOC 2) |
| **CVE** | Common Vulnerabilities and Exposures — a unique identifier for a publicly known security vulnerability |
| **CVSS** | Common Vulnerability Scoring System — rates vulnerability severity from 0.0 (none) to 10.0 (critical) |
| **Deployment** | The process of distributing and installing patches on one or more endpoints |
| **Deployment Target** | A single endpoint+patch combination within a deployment |
| **Endpoint** | Any device managed by Patch Manager (server, workstation, laptop, virtual machine) |
| **gRPC** | The encrypted communication protocol used between agents and the Patch Manager server |
| **Hub** | The central cloud service that aggregates vulnerability data from global sources |
| **Maintenance Window** | A scheduled time period during which patches are allowed to be installed on endpoints |
| **Patch** | A software update that fixes bugs, vulnerabilities, or adds improvements |
| **Patch Manager** | The on-premises server that orchestrates all patch management operations |
| **Policy** | A set of rules defining which patches should be deployed to which endpoints and when |
| **RBAC** | Role-Based Access Control — permissions system controlling who can do what |
| **Remediation** | The process of fixing a vulnerability by applying the appropriate patch |
| **Rollback** | Reverting an installed patch to the previous version |
| **Severity** | Risk classification: Critical (9.0+ CVSS), High (7.0-8.9), Medium (4.0-6.9), Low (0-3.9) |
| **Shoutrrr** | The notification library used by Patch Manager to send alerts via Email, Slack, Discord, and webhooks |
| **Tag** | A key-value label applied to endpoints for grouping and policy targeting |
| **Tenant** | An organizational unit in Patch Manager (for multi-tenant/MSP deployments) |
| **Wave** | A group of endpoints that receive a patch together during phased deployment. Each wave has its own success threshold and error rate maximum. |

---

*Patch Manager User Guide v1.0*
*For support, contact your system administrator.*
