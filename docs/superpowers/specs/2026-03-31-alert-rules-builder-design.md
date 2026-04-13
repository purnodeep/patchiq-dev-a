# Alert Rules Builder — Design Spec

> **Date:** 2026-03-31
> **Scope:** Frontend-only — no backend/API/DB changes
> **Files:** `event-catalog.ts` (new), `AlertRulesSheet.tsx` (rewrite)

## Problem

The current alert rules sheet exposes raw template fields (`{{.field_name}}`) that require knowing event type strings and payload field names. Client admins can't configure rules without developer help.

## Solution: Hybrid Preset Gallery + Guided Builder

### View 1: Preset Gallery (default)

- Category tabs: All | Deployments | CVEs | Agents | Compliance | System (with counts)
- Search bar for filtering by event name
- Compact rule cards: severity dot + event type (mono) + human label + enabled toggle
- Hover reveals "Edit" link
- "Create Custom Rule" button in header

### View 2: Guided Builder (edit/create)

Slides in when clicking Edit or Create:

1. **Event Type** — categorized dropdown with human labels and descriptions
2. **Classification** — severity pills (pre-selected, editable) + auto-derived category (locked)
3. **Alert Content** — title + description fields with:
   - Clickable field chips above inputs (insert `{{.field}}` at cursor)
   - "No dynamic fields available" for nil-payload events
   - Field chip tooltips showing type + sample value
4. **Preview** — rendered with sample data
5. **Reset to Default** button (restores preset template)

### Event Type Catalog (frontend-only registry)

Static TypeScript file mapping ~100 event types to metadata: label, description, category, default severity, payload fields (name, type, sample value), default templates.

Only events with known payload fields get field metadata. Others show "No dynamic fields."

### Duplicate Detection

When creating a new rule, if the event type already has a rule, show warning and link to edit existing.

## Files

| File | Action |
|------|--------|
| `web/src/pages/alerts/event-catalog.ts` | Create — static event type registry |
| `web/src/pages/alerts/AlertRulesSheet.tsx` | Rewrite — preset gallery + guided builder |
