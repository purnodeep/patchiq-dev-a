---
description: "Initialize OmniProd — auto-detect project stack, set up standards, configure perspectives"
argument-hint: "[--interview]"
allowed-tools: ["Read", "Write", "Glob", "Grep", "Bash"]
---

# OmniProd — Initialize

Set up OmniProd for this project. Auto-detects the tech stack, creates the persistence directory, and optionally runs an interactive interview to customize standards.

## Parse Arguments

Arguments: $ARGUMENTS

- `--interview`: Run the interactive customization interview (optional)

## Execution

### Step 1: Create Directory Structure

```
.omniprod/
├── config.json
├── reviews/
├── findings/
├── screenshots/
│   └── current/
└── standards-overrides/
```

### Step 2: Auto-Detect Project Context

Read these files to understand the project:
- `package.json` or `go.mod` — tech stack
- `CLAUDE.md` — project conventions, structure, identity
- `tailwind.config.*` or CSS theme files — design system
- `packages/ui/` — component library
- Any existing `tsconfig.json` — TypeScript setup

From these, extract:
- **Project name** and description
- **Tech stack** (React, Go, Tailwind, etc.)
- **Design system** details (color tokens, spacing, component library)
- **Port configuration** (for URL defaults)
- **Frontend apps** and their routes

### Step 3: Create Config

Write `.omniprod/config.json`:
```json
{
  "project_name": "{detected}",
  "tech_stack": "{detected}",
  "default_base_url": "http://localhost:3001",
  "perspectives": {
    "essential": ["ux-designer", "enterprise-buyer", "qa-engineer"],
    "optional": ["product-manager", "end-user"],
    "inactive": ["accessibility-expert", "cto-architect", "sales-engineer"]
  },
  "model_selection": {
    "intelligence_gathering": "opus",
    "exploration": "opus",
    "perspectives_critical": "opus",
    "perspectives_standard": "sonnet",
    "smoke_test": "sonnet",
    "correlation": "opus"
  },
  "severity_levels": ["critical", "major", "minor", "nitpick"],
  "pass_threshold": "unanimous",
  "detected_context": {
    "design_system": "{details}",
    "color_tokens": "{if found}",
    "spacing_scale": "{if found}",
    "component_library": "{if found}",
    "apps": ["{list of frontend apps with ports}"]
  }
}
```

### Step 4: Interactive Interview (if --interview)

Ask the user:

1. "What's your product and who are your target buyers/users?"
2. "I detected {design_system}. Is this your primary design system? Any brand guidelines I should know?"
3. "What's your accessibility target? (WCAG 2.1 AA is default)"
4. "Are there specific quality bars beyond the defaults? (e.g., page load time targets, specific competitors to benchmark against)"
5. "Which perspectives matter most for your product? (3 essential + 2 optional perspectives are active by default, 3 inactive)"

Based on answers, create project-specific standard overrides in `.omniprod/standards-overrides/`.

### Step 5: Report

Print:
```
✅ OmniProd initialized for {project_name}

Detected:
- Tech stack: {stack}
- Design system: {system}
- Frontend apps: {list}

Active perspectives: {count} ({list})
Standards: 6 base + {N} overrides

Next steps:
- Run /product-review <url> to review a page
- Run /product-check <url> for a quick check
- Run /product-config to customize perspectives or standards
```

**For development agents:**
- Read `.omniprod-plugin/omniprod/references/dev-expectations.md` before implementing UI changes
- This condensed checklist prevents 80% of findings from `/product-review`

### Step 6: Save to Memory

Save a project memory:
- Title: "OmniProd initialized for {project_name}"  
- Content: "OmniProd product quality system active. {N} perspectives, {N} base standards. Run /product-review <url> to audit pages. Dev agents should read .omniprod-plugin/omniprod/references/dev-expectations.md before UI work."
