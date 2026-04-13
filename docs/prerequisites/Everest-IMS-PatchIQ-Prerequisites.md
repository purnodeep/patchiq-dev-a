
---

<p align="center">
  <strong>PATCHIQ</strong><br/>
  Enterprise Patch Management Platform
</p>

---

# Infrastructure Prerequisite Document

| | |
|---|---|
| **Prepared For** | Everest IMS |
| **Product** | PatchIQ Patch Manager |
| **Deployment** | On-Premise (Containerized) |
| **Scale** | 8,000 Managed Endpoints |
| **Version** | 1.0 |
| **Date** | April 14, 2026 |
| **Classification** | Confidential — Everest IMS Internal Use Only |

---

## Table of Contents

1. [Purpose of This Document](#1-purpose-of-this-document)
2. [Solution Overview](#2-solution-overview)
3. [Server Requirements](#3-server-requirements)
   - 3.1 [Operating System](#31-operating-system)
   - 3.2 [Hardware Specifications](#32-hardware-specifications)
   - 3.3 [Disk Layout & Partitioning](#33-disk-layout--partitioning)
   - 3.4 [RAID Configuration](#34-raid-configuration)
   - 3.5 [Software to be Pre-Installed](#35-software-to-be-pre-installed)
4. [Network Requirements](#4-network-requirements)
   - 4.1 [Ports — Inbound (External Access)](#41-ports--inbound-external-access)
   - 4.2 [Ports — Internal (No Firewall Changes Needed)](#42-ports--internal-no-firewall-changes-needed)
   - 4.3 [Ports — Outbound (Server to Internet)](#43-ports--outbound-server-to-internet)
5. [URL & Domain Whitelisting](#5-url--domain-whitelisting)
   - 5.1 [Security Intelligence Feeds](#51-security-intelligence-feeds)
   - 5.2 [Patch Repository Sources](#52-patch-repository-sources)
   - 5.3 [Container Image Registries](#53-container-image-registries)
   - 5.4 [Complete Whitelist (Copy-Paste Ready)](#54-complete-whitelist-copy-paste-ready)
6. [Endpoint (Agent) Requirements](#6-endpoint-agent-requirements)
7. [Pre-Deployment Checklist](#7-pre-deployment-checklist)
8. [Connectivity Verification Commands](#8-connectivity-verification-commands)
9. [Support & Contact](#9-support--contact)

---

## 1. Purpose of This Document

This document outlines the infrastructure that Everest IMS must have in place **before** the PatchIQ deployment team begins installation.

Completing every item in this document ensures a smooth, on-schedule deployment. Items marked with **Action Required** need to be completed by the Everest IMS IT team prior to the deployment date.

---

## 2. Solution Overview

PatchIQ Patch Manager provides centralized patch management for your entire endpoint fleet. Here is how the solution works at a high level:

```
  +-------------------+          +-------------------+
  |                   |  HTTPS   |                   |
  |   Admin / Users   |--------->|   PatchIQ Web UI  |
  |   (Browsers)      |  :443    |   (Dashboard)     |
  +-------------------+          +---------+---------+
                                           |
                                           v
                              +------------------------+
                              |                        |
                              |   PatchIQ Patch        |
                              |   Manager Server       |
                              |                        |
                              |   - Patch Deployment   |
                              |   - CVE Correlation    |
                              |   - Compliance Engine  |
                              |   - Policy Engine      |
                              +-----+------------+-----+
                                    |            |
                          gRPC+mTLS |            | HTTPS
                          :50051    |            | (outbound)
                                    v            v
                      +-------------+--+    +----+--------+
                      |                |    |             |
                      |  8,000 Agents  |    |  NVD / CISA |
                      |  (Endpoints)   |    |  (Vuln Data)|
                      +----------------+    +-------------+
```

**What runs on your server (all containerized):**

| Component | What It Does |
|-----------|-------------|
| **PatchIQ Server** | Core engine — manages patches, deployments, policies, and compliance |
| **Web Dashboard** | Browser-based console for your admins to manage everything |
| **Database** | Stores endpoint inventory, deployment history, audit logs |
| **Cache** | Speeds up operations and manages user sessions |
| **Identity Manager** | Handles user login, roles, and permissions (SSO-ready) |
| **File Storage** | Stores reports, agent installers, and compliance exports |

**What runs on each endpoint:**

| Component | What It Does |
|-----------|-------------|
| **PatchIQ Agent** | Lightweight service that reports inventory, receives patch commands, and executes deployments |

---

## 3. Server Requirements

### 3.1 Operating System

> **Action Required:** Install the operating system with the settings below before deployment day.

| Setting | Requirement |
|---------|-------------|
| **OS** | Ubuntu 24.04 LTS (Noble Numbat) — Server Edition |
| **Architecture** | 64-bit (x86_64 / amd64) |
| **Installation Type** | Minimal server (no graphical desktop) |
| **Time Sync (NTP)** | Must be enabled — critical for security certificate validity |
| **Timezone** | Set to your local timezone |
| **Admin User** | Create a dedicated user (e.g., `patchiq-admin`) with sudo privileges |

---

### 3.2 Hardware Specifications

> **Action Required:** Procure and rack the server hardware before deployment day.

The following specifications are sized for **8,000 managed endpoints**.

#### Option A — Recommended

| Resource | Specification | Why |
|----------|--------------|-----|
| **CPU** | 24 cores | Handles 8,000 concurrent agent connections + patch deployment orchestration |
| **RAM** | 96 GB | Database caching, concurrent operations, and connection pools |
| **OS Disk** | 2 x 200 GB SSD (RAID 1) | Operating system and container images |
| **Data Disk** | 4 x 1 TB NVMe (RAID 10) | Database, reports, backups — high-speed writes for real-time agent data |
| **Network** | 10 Gbps | Fast agent communication during large-scale deployments |

#### Option B — Minimum Viable

| Resource | Specification | Notes |
|----------|--------------|-------|
| **CPU** | 16 cores | Adequate for steady-state, may slow during peak deployment waves |
| **RAM** | 64 GB | Functional but limited headroom |
| **OS Disk** | 2 x 100 GB SSD (RAID 1) | Minimum for OS + Docker |
| **Data Disk** | 3 x 1 TB NVMe (RAID 5) | ~30% slower writes than RAID 10; acceptable under 5,000 endpoints |
| **Network** | 1 Gbps | Sufficient for normal operations |

---

### 3.3 Disk Layout & Partitioning

> **Action Required:** Partition disks according to the layout below during OS installation.

#### OS Disk (RAID 1 — 200 GB usable)

| Partition | Size | Purpose |
|-----------|------|---------|
| `/boot/efi` | 512 MB | Boot loader (UEFI systems) |
| `/boot` | 1 GB | Kernel and boot files |
| `/` (root) | 30 GB | Operating system |
| `/var` | 40 GB | Docker images and container logs |
| `/tmp` | 10 GB | Temporary working files |
| `/home` | 10 GB | Admin user home directories |
| `swap` | 8 GB | System swap space |

#### Data Disk (RAID 10 — 2 TB usable)

| Directory | Recommended Size | What's Stored |
|-----------|-----------------|---------------|
| `/data/postgres` | 500 GB | All PatchIQ data — endpoints, patches, deployments, audit trail |
| `/data/minio` | 200 GB | Compliance reports, agent installers, exported data |
| `/data/backups` | 200 GB | Automated database backups |
| `/data/repo-cache` | 20 GB | Cached patch repository metadata |
| `/data/agent-binaries` | 5 GB | Agent installer packages (Windows, Linux, macOS) |
| `/data/valkey` | 10 GB | Cache persistence |
| *Free space* | ~1 TB | Growth headroom for audit logs and deployment history |

> **Important:** The data disk should be mounted with the `noatime` flag for optimal database performance.

---

### 3.4 RAID Configuration

> **Action Required:** Configure RAID before OS installation, either via hardware RAID controller or during Ubuntu setup.

| Disk Group | RAID Level | Disks Required | Usable Space | Purpose |
|------------|-----------|----------------|-------------|---------|
| **OS** | RAID 1 (Mirror) | 2 disks | 200 GB | Protects against OS disk failure — server boots even if one disk dies |
| **Data** | RAID 10 (Stripe + Mirror) | 4 disks | 2 TB | Best write speed for real-time agent data + protection against disk failure |

**Why RAID 10 for data?**
With 8,000 agents reporting every 30 seconds, the database handles ~267 writes per second continuously. RAID 10 delivers the fastest write performance of any redundant RAID level while surviving a disk failure.

**Budget alternative:** RAID 5 with 3 disks is acceptable for deployments under 5,000 endpoints, but writes are ~30% slower.

> A hardware RAID controller with battery-backed write cache is strongly recommended.

---

### 3.5 Software to be Pre-Installed

> **Action Required:** Install Docker Engine on the server before deployment day.

| Software | Version | How to Verify |
|----------|---------|---------------|
| **Docker Engine** | 27.0 or newer | Run: `docker --version` |
| **Docker Compose** | 2.29 or newer (plugin) | Run: `docker compose version` |

Installation guide: https://docs.docker.com/engine/install/ubuntu/

After installation, confirm Docker is running:
```
sudo systemctl status docker
```

> All other software (database, cache, identity manager, etc.) is included in the PatchIQ deployment package as Docker containers. You do **not** need to install them separately.

---

## 4. Network Requirements

### 4.1 Ports — Inbound (External Access)

> **Action Required:** Open these ports on your firewall for inbound traffic to the PatchIQ server.

| Port | Protocol | Who Connects | Purpose |
|------|----------|-------------|---------|
| **443** | HTTPS | Admin users (browsers) | Access to the PatchIQ web dashboard |
| **50051** | gRPC (encrypted) | All 8,000 managed endpoints | Agent communication — enrollment, status updates, patch commands |
| **8080** | HTTP | Agents / internal tools | REST API and agent installer downloads |
| **22** | SSH | IT administrators only | Server management (restrict to admin IPs) |

### 4.2 Ports — Internal (No Firewall Changes Needed)

These ports are used **only between containers** inside the server. They are not exposed to your network and require no action from your team.

| Port | Used By | Purpose |
|------|---------|---------|
| 5432 | Database (PostgreSQL) | Data storage |
| 6379 | Cache (Valkey) | Session and performance cache |
| 8080 | Identity Manager (Zitadel) | User authentication |
| 9000 | File Storage (MinIO) | Report and binary storage |

### 4.3 Ports — Outbound (Server to Internet)

> **Action Required:** Ensure the PatchIQ server can reach the internet on these ports.

| Port | Protocol | Why |
|------|----------|-----|
| **443** | HTTPS | Fetching vulnerability intelligence data (NVD, CISA) |
| **80** | HTTP | Downloading patch repository metadata (Ubuntu mirrors) |

See Section 5 for the exact domains.

---

## 5. URL & Domain Whitelisting

> **Action Required:** If your network uses a web proxy or URL-filtering firewall, whitelist all domains listed below.

### 5.1 Security Intelligence Feeds

PatchIQ automatically downloads vulnerability data from official government sources to identify which patches are critical for your environment.

| Domain | Port | What It Provides | How Often |
|--------|------|-----------------|-----------|
| `services.nvd.nist.gov` | 443 (HTTPS) | CVE vulnerability database with severity scores (NIST) | Every 24 hours |
| `www.cisa.gov` | 443 (HTTPS) | List of vulnerabilities actively exploited in the wild (CISA KEV) | Every 24 hours |

### 5.2 Patch Repository Sources

PatchIQ scans official OS repositories to build its patch catalog, so it knows what patches are available for your endpoints.

| Domain | Port | What It Provides | How Often |
|--------|------|-----------------|-----------|
| `security.ubuntu.com` | 80 (HTTP) | Ubuntu security patch listings | Every 60 minutes |
| `archive.ubuntu.com` | 80 (HTTP) | Ubuntu general patch listings | Every 60 minutes |

> **Note:** If your managed endpoints include RHEL, Windows, or macOS machines, additional repository domains may need to be whitelisted. The PatchIQ team will advise during deployment based on your endpoint inventory.

### 5.3 Container Image Registries

These are needed **only during initial setup and software updates**. They can be blocked after deployment if your security policy requires it.

| Domain | Port | What It Provides |
|--------|------|-----------------|
| `download.docker.com` | 443 (HTTPS) | Docker Engine software repository |
| `registry-1.docker.io` | 443 (HTTPS) | Container images (database, cache, web server) |
| `auth.docker.io` | 443 (HTTPS) | Docker registry authentication |
| `production.cloudflare.docker.com` | 443 (HTTPS) | Docker image download CDN |
| `ghcr.io` | 443 (HTTPS) | Identity manager container image |
| `gcr.io` | 443 (HTTPS) | Secure base container image |

### 5.4 Complete Whitelist (Copy-Paste Ready)

Provide this list to your network/firewall team:

```
# ── Required (Always) ──────────────────────────────

# Vulnerability intelligence feeds
services.nvd.nist.gov          port 443 (HTTPS)
www.cisa.gov                   port 443 (HTTPS)

# Patch repository metadata
security.ubuntu.com            port 80  (HTTP)
archive.ubuntu.com             port 80  (HTTP)

# ── Required (Setup & Updates Only) ────────────────

# Container image registries
download.docker.com            port 443 (HTTPS)
registry-1.docker.io           port 443 (HTTPS)
auth.docker.io                 port 443 (HTTPS)
production.cloudflare.docker.com  port 443 (HTTPS)
ghcr.io                        port 443 (HTTPS)
gcr.io                         port 443 (HTTPS)
```

---

## 6. Endpoint (Agent) Requirements

A small, lightweight agent is installed on each machine you want to manage. Here is what it needs:

### Supported Operating Systems

| Operating System | Supported Versions |
|------------------|--------------------|
| **Ubuntu / Debian** | 18.04, 20.04, 22.04, 24.04 |
| **RHEL / CentOS** | 7, 8, 9 |
| **Windows** | 10, 11, Server 2016, 2019, 2022 |
| **macOS** | 12 (Monterey) and newer |

### Resource Usage (Per Endpoint)

| Resource | Usage |
|----------|-------|
| **Disk space** | Less than 30 MB |
| **Memory (RAM)** | Less than 50 MB |
| **CPU** | Less than 1% at idle |

> The agent has a negligible impact on endpoint performance. End users will not notice it.

### Network Requirements (Per Endpoint)

| Port | Direction | Purpose |
|------|-----------|---------|
| **50051** | Endpoint **to** PatchIQ Server | Agent communication (encrypted) |
| **8080** | Endpoint **to** PatchIQ Server | Agent installer download |

> **Key point:** Agents connect **outbound** to the server. You do **not** need to open any inbound ports on your endpoints.

> **Action Required:** Ensure all 8,000 endpoints can reach the PatchIQ server IP on ports 50051 and 8080.

---

## 7. Pre-Deployment Checklist

Please complete every item below and confirm with the PatchIQ deployment team **at least 3 business days** before the scheduled installation date.

### Server Hardware & OS

| # | Task | Status |
|---|------|--------|
| 1 | Server hardware procured and racked (see Section 3.2) | [ ] Done |
| 2 | RAID configured (see Section 3.4) | [ ] Done |
| 3 | Ubuntu 24.04 LTS Server installed (minimal, no desktop) | [ ] Done |
| 4 | Disk partitions configured (see Section 3.3) | [ ] Done |
| 5 | OS fully updated (`sudo apt update && sudo apt upgrade -y`) | [ ] Done |
| 6 | NTP time synchronization enabled and verified | [ ] Done |
| 7 | Dedicated admin user created with sudo access | [ ] Done |
| 8 | Static IP address assigned to the server | [ ] Done |

### Software

| # | Task | Status |
|---|------|--------|
| 9 | Docker Engine 27.x+ installed and running | [ ] Done |
| 10 | Docker Compose v2.29+ available | [ ] Done |

### Network & Firewall

| # | Task | Status |
|---|------|--------|
| 11 | Inbound port 443 (HTTPS) open — for admin web access | [ ] Done |
| 12 | Inbound port 50051 (gRPC) open — for agent communication | [ ] Done |
| 13 | Inbound port 8080 (HTTP) open — for REST API and agent downloads | [ ] Done |
| 14 | Inbound port 22 (SSH) open — restricted to admin IPs | [ ] Done |
| 15 | Outbound access to all domains in Section 5 verified | [ ] Done |
| 16 | DNS record created for server (e.g., `patchiq.everestims.local`) | [ ] Done |

### Endpoints

| # | Task | Status |
|---|------|--------|
| 17 | Endpoint inventory list provided (hostname, OS, IP address) | [ ] Done |
| 18 | All endpoints can reach server IP on port 50051 and 8080 | [ ] Done |
| 19 | Admin/root access available on endpoints for agent installation | [ ] Done |

---

## 8. Connectivity Verification Commands

> Run these commands from the PatchIQ server to confirm everything is reachable. Share the output with the PatchIQ team if any fail.

**Test vulnerability feed access:**
```bash
curl -sI https://services.nvd.nist.gov/rest/json/cves/2.0 | head -1
# Expected: HTTP/2 200

curl -sI https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json | head -1
# Expected: HTTP/2 200
```

**Test patch repository access:**
```bash
curl -sI http://security.ubuntu.com/ubuntu/dists/noble-security/main/binary-amd64/Packages.gz | head -1
# Expected: HTTP/1.1 200 OK

curl -sI http://archive.ubuntu.com/ubuntu/dists/noble-updates/main/binary-amd64/Packages.gz | head -1
# Expected: HTTP/1.1 200 OK
```

**Test container registry access:**
```bash
curl -sI https://registry-1.docker.io/v2/ | head -1
# Expected: HTTP/1.1 401 Unauthorized (this is normal — it means the registry is reachable)

curl -sI https://ghcr.io/v2/ | head -1
# Expected: HTTP/1.1 401 Unauthorized (same — reachable is what matters)
```

**Test Docker installation:**
```bash
docker --version
# Expected: Docker version 27.x.x

docker compose version
# Expected: Docker Compose version v2.29.x+
```

---

## 9. Support & Contact

For questions about this document or assistance with preparation, contact the PatchIQ deployment team.

| | |
|---|---|
| **Deployment Lead** | *To be assigned* |
| **Email** | *To be provided* |
| **Deployment Date** | *To be confirmed* |

---

<p align="center">
  <em>PatchIQ — Prerequisite Document v1.0 — Prepared for Everest IMS — April 2026</em><br/>
  <em>Confidential</em>
</p>
