# Everest — Client Deployment

Everything related to the Everest client deployment lives here.

## Context

Everest is our first client POC. They previously used **Zirozen** for patch management and have built their platform (SOAR, CMDB, ticketing integrations) against Zirozen's REST API. They expect PatchIQ to expose a Zirozen-compatible API layer so their existing automation works without changes.

## Contents

| File | Purpose |
|------|---------|
| [zirozen-api-v1.0.pdf](zirozen-api-v1.0.pdf) | Original Zirozen API documentation (from Everest) |
| [api-compatibility.md](api-compatibility.md) | Zirozen API mapping to PatchIQ — field-by-field analysis |
| [compat-layer-plan.md](compat-layer-plan.md) | Implementation plan for the `/api/compat/zirozen/` adapter |
| [hardening-plan.md](hardening-plan.md) | Production hardening plan for client deployment |

## Timeline

- **Week 1**: Phase 0 — Security & correctness fixes (10 critical items)
- **Week 2**: Phase 0.5 — Zirozen compat layer + Phase 1 (broken UX) + Phase 2 starts (cleanup)
- **Week 3**: Phase 3 — Frontend consistency + Phase 4 starts (testing)
- **Week 4**: Buffer + final QA pass before deployment
