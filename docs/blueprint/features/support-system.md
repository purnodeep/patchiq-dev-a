# Feature: Support System

> Status: Proposed | Phase: 2-4 | License: Tiered (see below)

---

## Overview

A multi-tier support system with in-product diagnostics, knowledge base, health scoring, and remote diagnostics — enabling proactive support before customers file tickets.

## Problem Statement

Enterprise patch management is infrastructure-critical. When issues arise, customers need fast resolution. Without built-in diagnostics, support interactions start with lengthy back-and-forth to gather system state.

---

## In-Product Support

| Feature | Description |
|---------|-------------|
| **Support Bundle Generator** | One-click diagnostic package: logs, config (sanitized), system info, agent status, DB health — compressed and ready to upload to support |
| **In-App Knowledge Base** | Searchable documentation integrated into the Patch Manager UI, context-aware (shows relevant articles based on current page) |
| **Health Dashboard** | `/health` endpoint with structured status of all components (DB, Redis, MinIO, agents) — visible to admins and support team |
| **Remote Diagnostics API** | With customer consent, support can query anonymized system health metrics from the Hub |
| **Guided Troubleshooting** | AI assistant can diagnose common issues: "Why did deployment DEP-123 fail?" and walk through resolution |

---

## Support Tiers

| Tier | Response SLA | Channels | Features |
|------|-------------|----------|----------|
| **Community** | Best effort | GitHub Discussions | Community docs, public forum |
| **Professional** | 48h (business) | Email + Portal | Support portal, knowledge base |
| **Enterprise** | 4h (critical, 24/7) | Email + Portal + Phone | Dedicated engineer, quarterly reviews |
| **MSP** | 2h (critical, 24/7) | Dedicated Slack + Phone | Named account team, custom SLAs |

---

## Customer Health Score

Automated scoring based on:
- Agent connectivity rate (% of agents checking in regularly)
- Patch compliance percentage
- License utilization (endpoints used vs. licensed)
- Support ticket frequency and severity
- Feature adoption breadth
- Platform version currency (how up-to-date they are)

This feeds into proactive outreach — if a customer's health drops, support reaches out before they file a ticket.

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **Observability** | Support bundles pull from OTel metrics and structured logs |
| **AI Assistant** | Guided troubleshooting uses the AI to diagnose issues |
| **License** | Support tier is tied to license tier |
| **Hub Manager** | Remote diagnostics API and health score live in the Hub |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| Health endpoint | `internal/server/apm/` |
| Support bundle | `internal/server/support/` |
| Hub diagnostics | `internal/hub/support/` |
