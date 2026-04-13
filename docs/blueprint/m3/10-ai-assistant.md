# AI Assistant (MCP + Claude API)

**Status**: Planned
**Wave**: 3 — AI (last, builds on everything above)
**Dependencies**: All M3 Wave 1-2 features (AI needs a complete platform to be useful)

---

## Vision

Natural language interface to the entire PatchIQ platform. An AI assistant that can query endpoints, search CVEs, explain compliance gaps, create deployments, and take action — all filtered through the user's RBAC permissions.

## Deliverables

### MCP Server
- [ ] MCP server embedded in Patch Manager (Go SDK, Streamable HTTP transport)
- [ ] Tool registry matching platform capabilities
- [ ] Session management (conversation history per user)

### AI Tools (13)
| Tool | Type | Description |
|------|------|-------------|
| `list_endpoints` | Read | List/filter endpoints by status, OS, tags |
| `get_endpoint_detail` | Read | Full endpoint info including CVEs, patches, compliance |
| `list_patches` | Read | List/filter patches by severity, OS, status |
| `search_cves` | Read | Search CVEs by ID, severity, KEV status, affected package |
| `get_compliance_report` | Read | Framework scores, non-compliant controls, exceptions |
| `get_audit_log` | Read | Query audit events by type, actor, date range |
| `create_policy` | Write | Create a policy with targets, patches, schedule |
| `create_deployment` | Write | Create a deployment targeting endpoints |
| `approve_deployment` | Write | Approve a pending deployment |
| `cancel_deployment` | Destructive | Cancel a running deployment |
| `trigger_scan` | Write | Trigger scan on endpoint(s) |
| `create_group` | Write | Create/modify tag assignments |
| `modify_policy` | Write | Update policy configuration |

### Human-in-the-Loop
- [ ] Read-only tools: execute immediately, show results
- [ ] Write tools: show plan, require confirmation ("Create deployment targeting 5 endpoints?")
- [ ] Destructive tools: require explicit approval with warning
- [ ] Confirmation UI: inline approval buttons in chat

### RBAC Integration
- [ ] AI tool calls filtered to user's permissions
- [ ] If user can't view CVEs, AI can't search CVEs
- [ ] Tool availability shown in chat context

### Audit Logging
- [ ] All AI actions logged: `actor: ai_assistant, confirmed_by: <user_id>`
- [ ] Conversation history stored (30-day retention)
- [ ] AI actions visible in audit log with conversation link

### Chat Panel UI
- [ ] Sidebar panel in Patch Manager (collapsible)
- [ ] Conversation history
- [ ] Context awareness: AI sees current page (which endpoint, deployment, CVE)
- [ ] Suggested prompts: "What CVEs affect this endpoint?", "Create a deployment for critical patches"
- [ ] Streaming responses

## License Gating
- AI Assistant: ENTERPRISE only
- All AI tools: ENTERPRISE
