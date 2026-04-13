---
description: "Add/remove perspectives, customize product standards, configure review settings"
argument-hint: "<action> [args] — actions: add-perspective, remove-perspective, list-perspectives, edit-standards, show-standards"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash"]
---

# OmniProd — Configuration

Manage perspectives, standards, and settings for OmniProd.

## Parse Arguments

Arguments: $ARGUMENTS

Actions:
- `add-perspective <role-name>` — Interview and create a new perspective
- `remove-perspective <name>` — Remove a perspective from the active list
- `list-perspectives` — Show all available perspectives and their status
- `edit-standards [standard-name]` — Modify a standard or create an override
- `show-standards` — Display current standards with any overrides
- `set <key> <value>` — Update config.json (e.g., `set default_model haiku`)

## Actions

### add-perspective

Interview the user:
1. "What role is this stakeholder? (e.g., Data Privacy Officer, DevOps Engineer)"
2. "What do they care about most when evaluating software?"
3. "What would make them reject the product?"
4. "What delights them?"
5. "What's their perspective prefix? (e.g., DPO, OPS — 2-4 chars)"
6. "Which tier should this perspective start in? (essential — always runs, optional — critical pages only, inactive — disabled)"

Generate a perspective `.md` file following the format of existing perspectives in the plugin's `perspectives/` directory. Structure:
- `## Who You Are` — 2-3 sentences, second person, strong opinions
- `## What You Care About` — detailed bullets
- `## Your Quality Bar` — what PASS and FAIL mean
- `## Severity Calibration` — examples per severity level

Save to: the plugin's `perspectives/` directory (find via Glob for `.omniprod-plugin/omniprod/perspectives/`)
Add the name to the appropriate tier array (`essential`, `optional`, or `inactive`) in `.omniprod/config.json` under `perspectives`.

### remove-perspective

Move the perspective to the `inactive` tier in `.omniprod/config.json` rather than deleting it. This preserves the perspective file so it can be re-activated later.

If the user passes `--delete`, remove it from all tiers in config.json entirely. Do not delete the perspective `.md` file.

### list-perspectives

Read config.json. Show perspectives grouped by tier:

```
Essential (always run):
  - ux-designer
  - enterprise-buyer
  - qa-engineer

Optional (critical pages only):
  - product-manager
  - end-user

Inactive:
  - accessibility-expert
  - cto-architect
  - sales-engineer
```

For each perspective, also show its brief description (from the first line of the perspective file).

### edit-standards

If a standard name is given, read both:
1. The base standard from the plugin's `standards/` directory
2. Any existing override from `.omniprod/standards-overrides/`

Show the current rules and ask the user what they want to change. Save changes to `.omniprod/standards-overrides/{standard-name}.md`.

If no standard name given, list available standards and ask which to edit.

### show-standards

Read all base standards and any overrides. Present a summary of each standard category with its key rules. Note which have project-specific overrides.

### set

Update the specified key in `.omniprod/config.json`. Valid keys:
- `default_model` — sonnet, opus, haiku
- `orchestrator_model` — sonnet, opus
- `pass_threshold` — unanimous, majority
- `default_base_url` — URL prefix for relative paths
