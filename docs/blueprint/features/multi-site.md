# Feature: Multi-Site Deployment Topologies

> Status: Proposed | Phase: 3 | License: Enterprise/MSP

---

## Overview

Four deployment topology patterns for organizations of different sizes and network architectures — from single-site SMBs to globally federated enterprises.

## Problem Statement

Enterprises have branch offices, DMZs, air-gapped networks, and multiple autonomous HQs. A single-server architecture doesn't scale to these real-world network topologies. Agents at branch offices can't download multi-gigabyte patches over slow WAN links.

---

## Topology 1: Single Site (Standard)

```
┌─────────────────────────────────────┐
│            Client HQ                │
│                                     │
│  ┌─────────────────────────────┐    │
│  │      Patch Manager          │    │
│  │   + PostgreSQL + Redis      │    │
│  │   + MinIO                   │    │
│  └──────────────┬──────────────┘    │
│                 │                    │
│     ┌───────────┼───────────┐       │
│     │           │           │       │
│   Agent       Agent       Agent     │
└─────────────────────────────────────┘
```

---

## Topology 2: Hub-Spoke (Parent-Child) for Multi-Branch

```
┌────────────────────────────────────────────────────┐
│                    HQ (Parent)                      │
│  ┌──────────────────────────────┐                  │
│  │  Patch Manager (Primary)     │                  │
│  │  - Central policies          │                  │
│  │  - Aggregated reporting      │                  │
│  │  - Master database           │                  │
│  └──────────────┬───────────────┘                  │
└─────────────────┼──────────────────────────────────┘
                  │ WAN (encrypted)
        ┌─────────┼─────────┐
        │                   │
┌───────┴────────┐  ┌──────┴─────────┐
│  Branch A      │  │  Branch B      │
│  (Child)       │  │  (Child)       │
│                │  │                │
│ ┌────────────┐ │  │ ┌────────────┐ │
│ │Distribution│ │  │ │Distribution│ │
│ │  Server    │ │  │ │  Server    │ │
│ │ (PM Lite)  │ │  │ │ (PM Lite)  │ │
│ └─────┬──────┘ │  │ └─────┬──────┘ │
│       │        │  │       │        │
│  Agents (200)  │  │  Agents (500)  │
└────────────────┘  └────────────────┘
```

**Distribution Server** (a lightweight Patch Manager component):
- Caches patch binaries locally (avoids re-downloading over WAN for every agent)
- Relays agent heartbeats and inventory to the parent
- Executes deployments locally using policies pushed from parent
- Stores local SQLite cache for offline resilience if WAN goes down
- Can manage up to 5,000 agents per distribution server

**Parent-Child synchronization:**
- Policies pushed from parent to children (centralized control)
- Inventory and patch results rolled up from children to parent (consolidated reporting)
- Patch binaries synced from parent (or from Hub) to distribution servers on schedule
- Bandwidth throttling configurable per site link

---

## Topology 3: DMZ/MZ Deployment

For environments where agents are in a secure internal network and the Patch Manager must be accessible from a DMZ:

```
┌─────────────────────────────────────────────────────┐
│                    DMZ                               │
│  ┌──────────────────────────────────┐               │
│  │      Patch Manager Proxy         │               │
│  │  (Reverse proxy + TLS term)      │               │
│  │  - API Gateway only              │               │
│  │  - No data storage               │               │
│  │  - mTLS termination for agents   │               │
│  └──────────────┬───────────────────┘               │
└─────────────────┼───────────────────────────────────┘
                  │ Internal firewall (port 443 only)
┌─────────────────┼───────────────────────────────────┐
│              Internal Network (MZ)                   │
│  ┌──────────────┴───────────────────┐               │
│  │      Patch Manager (Core)         │               │
│  │  + PostgreSQL + Redis + MinIO     │               │
│  └──────────────┬───────────────────┘               │
│                 │                                    │
│            Internal Agents                           │
└─────────────────────────────────────────────────────┘
```

---

## Topology 4: Multi-HQ (Federated)

For large enterprises with multiple independent HQs that need their own autonomy but unified reporting:

```
┌──────────────────────────────────────────┐
│          PatchIQ Hub Manager             │
│  (Global view, license management)       │
└──────┬───────────────────────┬───────────┘
       │                       │
┌──────┴──────────┐   ┌───────┴─────────┐
│     HQ Europe    │   │    HQ Americas   │
│  Patch Manager   │   │  Patch Manager   │
│  (Independent)   │   │  (Independent)   │
│  Own policies    │   │  Own policies    │
│  Own database    │   │  Own database    │
│  2,000 endpoints │   │  5,000 endpoints │
└─────────────────┘   └─────────────────┘
```

Each HQ runs a fully independent Patch Manager with its own database. The Hub Manager provides:
- Unified compliance reporting across all HQs
- Centralized license management
- Shared patch catalog and feed
- Cross-HQ policy templates (optional adoption by each HQ)

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **HA/DR** | Each topology tier can layer HA on top |
| **Agent** | Agents connect to their nearest server (distribution or primary) |
| **Compliance** | Parent aggregates compliance data from all children |
| **License** | License covers all sites under one deployment |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Distribution server | `internal/server/distribution/` |
| Agent multi-server config | `internal/agent/comms/` |
| Hub sync | `internal/hub/sync/` |
