# ADR-002: React Flow for Visual Workflow Builder

## Status

Accepted

## Context

PatchIQ needs a visual policy and deployment pipeline builder. Admins should be able to construct workflows by dragging and connecting nodes rather than filling out forms.

## Decision

Use React Flow + ELK.js for auto-layout to build the visual workflow builder.

## Consequences

- **Positive**: No competitor offers this — key differentiator; proven library (Stripe, Typeform use it); natural fit for policy DAGs; also reusable for topology maps and dependency graphs
- **Negative**: Complex component; requires custom node types; ELK.js adds bundle size; learning curve for the team

## Alternatives Considered

- **Form-based wizards**: Standard approach (ManageEngine, WSUS) — rejected because it's the status quo and not differentiating
- **D3.js custom canvas**: More flexibility — rejected because significantly more effort to build interactive editing
- **Rete.js**: Another visual programming library — rejected because smaller community and less mature than React Flow
