# M4 — Scale

> **Goal**: Enterprise tier. High availability, multi-site distribution, full MSP support, AI-powered intelligence, and infrastructure automation for the largest organizations.

## Features

| # | Feature | Status | Doc |
|---|---------|--------|-----|
| 1 | [HA/DR](01-ha-dr.md) | Planned | Active-passive + active-active, Patroni, Valkey Cluster, automated failover |
| 2 | [Multi-Site Distribution](02-multi-site.md) | Planned | Hub-spoke (5K agents/DS), DMZ mode, federated multi-HQ, content caching |
| 3 | [MSP Portal (Full)](03-msp-portal-full.md) | Planned | White-label, billing metrics, tenant provisioning (builds on M3 foundations) |
| 4 | [AI Patch Pipeline v2](04-ai-patch-pipeline-v2.md) | Planned | LLM crawler, AI installer analysis, sandbox VM testing, confidence scoring |
| 5 | [AI v2](05-ai-v2.md) | Planned | Patch risk prediction, optimal scheduling, anomaly detection, NL reports |
| 6 | [Remote Access](06-remote-access.md) | Planned | Browser terminal, remote desktop, file transfer, session RBAC, recording |
| 7 | [Development Compliance](07-development-compliance.md) | Planned | License auditing, HIPAA data handling, toolchain enforcement |
| 8 | [Infrastructure Automation](08-infrastructure-automation.md) | Planned | Terraform, Ansible, GitOps, webhook triggers |
| 9 | [Kubernetes + Air-Gapped](09-kubernetes-airgapped.md) | Planned | Helm chart, horizontal scaling, OVA appliance, offline catalog |
| 10 | [Support System](10-support-system.md) | Planned | Client health scoring, remote diagnostics, ticket routing |

## Work Streams (3 devs)

| Dev | Track | Focus |
|-----|-------|-------|
| Dev 1 | Platform | HA/DR, multi-site, K8s Helm chart, air-gapped mode, load testing |
| Dev 2 | Engine | AI Pipeline v2, AI v2, MSP backend, IaC providers (Terraform, Ansible) |
| Dev 3 | Surface | MSP portal UI, topology management, HA monitoring dashboard, support dashboard |

## Exit Criteria

- [ ] Patroni failover within RTO targets (automated test)
- [ ] Distribution server caches and serves to 5K agents
- [ ] MSP admin manages 10+ tenants from single Hub dashboard
- [ ] AI pipeline processes vendor advisory → verified catalog entry (E2E)
- [ ] Sandbox VM tests installer and detects failure
- [ ] Terraform provider creates resources via `terraform apply`
- [ ] Helm chart deploys all 3 platforms on Kubernetes
- [ ] Air-gapped appliance operates without internet
- [ ] Load test passes at 5K+ concurrent agents
