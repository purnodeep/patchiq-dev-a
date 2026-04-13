# Foundational Architecture Gaps — Must Be Baked In From Day 1

> These 7 design areas must be implemented in Phase 1 scaffolding. Retrofitting any of them later
> would require touching nearly every file in the codebase. They are prerequisites for the platform
> to evolve from patch management into UEM (Unified Endpoint Management).

---

## 1. Multi-Tenancy Data Isolation

### Problem

Blueprint V2 mentions tenant scopes in RBAC and an MSP portal in Phase 3, but there is no schema-level isolation strategy. If we build single-tenant tables now and bolt on tenancy later, we'd need to:
- Add a `tenant_id` column to every table
- Rewrite every query
- Backfill data
- Audit every API endpoint for tenant leakage

### Design: Row-Level Isolation with PostgreSQL RLS

We use a **single database, shared schema** model with `tenant_id` on every table and PostgreSQL Row-Level Security (RLS) as a safety net.

**Why not schema-per-tenant or database-per-tenant:**
- Schema-per-tenant makes cross-tenant reporting (Hub Manager dashboards) extremely painful
- Database-per-tenant makes migrations a nightmare (run migration N times)
- Row-level isolation with RLS is what GitLab, Notion, and most modern SaaS products use
- It scales to thousands of tenants without operational overhead

#### Schema Pattern

Every table that holds tenant-scoped data includes `tenant_id` as the first column after the primary key:

```sql
-- The tenant table itself
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    license_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Example: endpoints table
CREATE TABLE endpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    hostname    TEXT NOT NULL,
    os_family   TEXT NOT NULL,
    os_version  TEXT NOT NULL,
    agent_version TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Composite index: tenant_id is ALWAYS the leading column
CREATE INDEX idx_endpoints_tenant ON endpoints(tenant_id);
CREATE INDEX idx_endpoints_tenant_status ON endpoints(tenant_id, status);

-- Row-Level Security as a safety net
ALTER TABLE endpoints ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON endpoints
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

#### Middleware: Tenant Context Injection

Every incoming request sets the tenant context before any database query executes:

```go
// Middleware sets tenant_id from JWT claims or API key
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantID(r) // from JWT, API key, or subdomain
        ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Every database transaction sets the PostgreSQL session variable
func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    tenantID := TenantIDFromContext(ctx)
    _, err = tx.ExecContext(ctx,
        "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
    if err != nil {
        tx.Rollback()
        return nil, fmt.Errorf("set tenant context: %w", err)
    }
    return tx, nil
}
```

#### Store Layer: Explicit tenant_id in All Queries

RLS is the safety net, but the application layer always filters explicitly:

```go
func (r *EndpointRepo) List(ctx context.Context, filter EndpointFilter) ([]Endpoint, error) {
    tenantID := TenantIDFromContext(ctx)
    query := `SELECT id, hostname, os_family, status, last_seen
              FROM endpoints
              WHERE tenant_id = $1`
    args := []any{tenantID}

    if filter.Status != "" {
        query += " AND status = $2"
        args = append(args, filter.Status)
    }
    // ...
}
```

**Double protection**: Even if a developer forgets the `WHERE tenant_id =` clause, RLS prevents cross-tenant data leakage.

#### What Stays Global (No tenant_id)

Some tables are shared across all tenants and live outside RLS:

- `tenants` — the tenant registry itself
- `patch_catalog` — the shared software catalog from Hub
- `cve_feeds` — vulnerability data from NVD/CISA
- `agent_binaries` — agent release artifacts

#### Day 1 Rule

**Every new table must include `tenant_id` unless it's in the global list above.** The migration tool should lint for this.

#### Single-Tenant Deployments

For customers running a single-tenant Patch Manager (most Phase 1 customers), a default tenant is auto-created during installation. The system works identically — the tenant_id column just always has the same value. Zero overhead, full forward-compatibility.

---

## 2. Domain Events & Audit System

### Problem

Blueprint V2 mentions "audit trail for every action" but doesn't define how events are captured, stored, or consumed. Without a domain event system from day 1:
- Adding audit logging later means inserting log calls into every existing handler
- Building integrations (webhooks, SIEM export) requires ad-hoc plumbing each time
- Debugging production issues requires reconstructing sequences from scattered logs

### Design: Domain Event Bus with Append-Only Audit Store

#### Event Schema

Every significant action in the system emits a domain event:

```go
type DomainEvent struct {
    ID        string    `json:"id"`         // ULID (sortable, unique)
    Type      string    `json:"type"`       // e.g., "endpoint.registered", "deployment.started"
    TenantID  string    `json:"tenant_id"`
    ActorID   string    `json:"actor_id"`   // User ID, system, or agent ID
    ActorType string    `json:"actor_type"` // "user", "agent", "system", "ai_assistant"
    Resource  string    `json:"resource"`   // Resource type: "endpoint", "deployment", etc.
    ResourceID string   `json:"resource_id"`
    Action    string    `json:"action"`     // "created", "updated", "deleted", "executed"
    Payload   any       `json:"payload"`    // Event-specific structured data
    Metadata  EventMeta `json:"metadata"`
    Timestamp time.Time `json:"timestamp"`
}

type EventMeta struct {
    TraceID   string `json:"trace_id"`
    RequestID string `json:"request_id"`
    IPAddress string `json:"ip_address,omitempty"` // For user actions
    UserAgent string `json:"user_agent,omitempty"`
}
```

#### Event Types (Non-Exhaustive)

```
# Agent lifecycle
agent.enrolled
agent.heartbeat_missed
agent.disconnected
agent.updated

# Endpoint management
endpoint.registered
endpoint.group_changed
endpoint.decommissioned

# Patch operations
patch.discovered
patch.approved
patch.rejected

# Deployment lifecycle
deployment.created
deployment.wave_started
deployment.wave_completed
deployment.succeeded
deployment.failed
deployment.rolled_back

# Policy changes
policy.created
policy.updated
policy.deleted
policy.evaluated

# Auth & access
user.login
user.login_failed
user.permission_denied
role.created
role.updated

# System
license.validated
license.expired
license.grace_period_entered
config.changed
```

#### Event Bus Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Application Layer                     │
│                                                           │
│  API Handler / gRPC Handler / Engine / Scheduler          │
│         │                                                 │
│         │ Emit(event)                                     │
│         ▼                                                 │
│  ┌──────────────────┐                                     │
│  │   Event Bus       │ (in-process, synchronous fanout)   │
│  │   (EventEmitter)  │                                    │
│  └──────┬────────────┘                                    │
│         │                                                 │
│    ┌────┼────────────┬────────────┬────────────┐          │
│    │    │            │            │            │          │
│    ▼    ▼            ▼            ▼            ▼          │
│  Audit  Webhook     Redis       Metrics     Notification │
│  Store  Dispatcher  Pub/Sub     Counter     Trigger      │
│  (PG)   (async)     (for other  (OTel)      (email,      │
│                     services)               slack)       │
└──────────────────────────────────────────────────────────┘
```

#### Implementation

```go
// EventBus interface — all event emission goes through this
type EventBus interface {
    // Emit sends an event to all registered subscribers
    Emit(ctx context.Context, event DomainEvent) error

    // Subscribe registers a handler for specific event types
    // Pattern supports wildcards: "deployment.*", "*"
    Subscribe(pattern string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event DomainEvent) error

// In-process implementation using channels + goroutine pool
type inProcessBus struct {
    handlers map[string][]EventHandler
    mu       sync.RWMutex
}

// Usage in a handler:
func (h *DeploymentHandler) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) error {
    deployment, err := h.engine.CreateDeployment(ctx, req)
    if err != nil {
        return err
    }

    // Emit event — all subscribers (audit, webhooks, metrics) fire automatically
    h.events.Emit(ctx, DomainEvent{
        ID:         ulid.New(),
        Type:       "deployment.created",
        TenantID:   TenantIDFromContext(ctx),
        ActorID:    UserIDFromContext(ctx),
        ActorType:  "user",
        Resource:   "deployment",
        ResourceID: deployment.ID,
        Action:     "created",
        Payload: map[string]any{
            "policy_id":      req.PolicyID,
            "endpoint_count": len(req.EndpointIDs),
            "wave_count":     req.WaveCount,
        },
        Metadata: EventMeta{
            TraceID:   trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
            RequestID: RequestIDFromContext(ctx),
        },
        Timestamp: time.Now(),
    })

    return nil
}
```

#### Audit Store (PostgreSQL)

```sql
-- Append-only audit log table
CREATE TABLE audit_events (
    id          TEXT PRIMARY KEY,   -- ULID
    type        TEXT NOT NULL,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    actor_id    TEXT NOT NULL,
    actor_type  TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    action      TEXT NOT NULL,
    payload     JSONB,
    metadata    JSONB,
    timestamp   TIMESTAMPTZ NOT NULL,
    -- No updated_at: audit events are immutable
    -- No DELETE permission granted to application role
    CONSTRAINT audit_events_immutable CHECK (true)  -- placeholder for trigger
);

-- Partition by month for efficient retention management
CREATE TABLE audit_events_2026_01 PARTITION OF audit_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- Indexes for common queries
CREATE INDEX idx_audit_tenant_time ON audit_events(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_resource ON audit_events(tenant_id, resource, resource_id);
CREATE INDEX idx_audit_actor ON audit_events(tenant_id, actor_id);
CREATE INDEX idx_audit_type ON audit_events(tenant_id, type);
```

**Immutability enforcement**: The database role used by the application has `INSERT` and `SELECT` only on the audit table — no `UPDATE` or `DELETE`. A separate retention job (run as superuser) drops old partitions.

#### Why Not a Separate Event Store (Kafka, NATS)?

For Phase 1, an in-process event bus + PostgreSQL audit table is sufficient. It avoids operational complexity. If scale demands it later:
- Swap the in-process bus for Redis Pub/Sub (already in the stack)
- Or NATS (lightweight, embeddable in Go)
- The `EventBus` interface abstracts this — subscribers don't change

#### Day 1 Rule

**Every write operation (create, update, delete, execute) must emit a domain event.** This is enforced by code review and the PR checklist: "Does this write operation emit an event?"

---

## 3. Offline-First Agent Design

### Problem

Blueprint V2 says agents are "offline-resilient" with SQLite queuing, but provides no detail on queue design, sync protocol, or conflict resolution. Without a proper offline-first design:
- Agents in branch offices with flaky WAN will lose data
- Reconnection storms after network recovery will overwhelm the server
- There's no clear contract for what the agent can do while disconnected

### Design: Local-First with Store-and-Forward

#### Agent Architecture (Offline-First)

```
┌────────────────────────────────────────────────────────────┐
│                        PatchIQ Agent                        │
│                                                             │
│  ┌─────────────┐    ┌──────────────┐    ┌───────────────┐  │
│  │  Inventory   │    │  Patcher     │    │  Command      │  │
│  │  Collector   │    │  (installer) │    │  Executor     │  │
│  └──────┬──────┘    └──────┬───────┘    └──────┬────────┘  │
│         │                  │                   │            │
│         ▼                  ▼                   ▼            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Local State Manager                      │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────┐  │  │
│  │  │ Outbox     │  │ Command    │  │ Local          │  │  │
│  │  │ Queue      │  │ Inbox      │  │ Inventory DB   │  │  │
│  │  │ (pending   │  │ (pending   │  │ (last known    │  │  │
│  │  │  uploads)  │  │  commands) │  │  state)        │  │  │
│  │  └────────────┘  └────────────┘  └────────────────┘  │  │
│  │                    SQLite                             │  │
│  └──────────────────────┬───────────────────────────────┘  │
│                         │                                   │
│  ┌──────────────────────┴───────────────────────────────┐  │
│  │             Connection Manager                        │  │
│  │  - gRPC client with mTLS                              │  │
│  │  - Connection state: CONNECTED / DISCONNECTED         │  │
│  │  - Exponential backoff reconnection                   │  │
│  │  - Sync orchestrator (flush outbox, pull inbox)       │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

#### SQLite Schema (Agent-Side)

```sql
-- Agent's local database

-- Outbox: data waiting to be sent to the server
CREATE TABLE outbox (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    message_type TEXT NOT NULL,     -- 'inventory', 'patch_result', 'heartbeat', 'event'
    payload     BLOB NOT NULL,      -- protobuf-encoded message
    created_at  TEXT NOT NULL,       -- ISO 8601
    attempts    INTEGER DEFAULT 0,
    last_error  TEXT,
    status      TEXT DEFAULT 'pending' -- 'pending', 'sending', 'sent', 'failed'
);

CREATE INDEX idx_outbox_status ON outbox(status, created_at);

-- Inbox: commands received from server, pending execution
CREATE TABLE inbox (
    id          TEXT PRIMARY KEY,    -- server-assigned command ID
    command_type TEXT NOT NULL,       -- 'install_patch', 'run_scan', 'update_config', 'reboot'
    payload     BLOB NOT NULL,       -- protobuf-encoded command
    priority    INTEGER DEFAULT 0,   -- higher = more urgent
    received_at TEXT NOT NULL,
    execute_at  TEXT,                 -- scheduled execution time (maintenance window)
    status      TEXT DEFAULT 'pending', -- 'pending', 'executing', 'completed', 'failed'
    result      BLOB,                -- protobuf-encoded result
    completed_at TEXT
);

CREATE INDEX idx_inbox_status ON inbox(status, priority DESC, execute_at);

-- Local inventory cache
CREATE TABLE local_inventory (
    package_name  TEXT NOT NULL,
    version       TEXT NOT NULL,
    os_family     TEXT NOT NULL,
    source        TEXT,              -- 'apt', 'yum', 'winupdate', 'homebrew', etc.
    installed_at  TEXT,
    scanned_at    TEXT NOT NULL,
    PRIMARY KEY (package_name, os_family)
);

-- Agent state
CREATE TABLE agent_state (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Stores: agent_id, enrollment_token, server_address, last_sync_time,
--         config_version, assigned_profile_version
```

#### Offline Behavior Rules

| Scenario | Agent Behavior |
|----------|---------------|
| **Scheduled scan while offline** | Runs the scan, stores results in outbox. Uploads when connected. |
| **Patch install command received before disconnect** | Executes from inbox if binary is already cached. Stores result in outbox. |
| **Patch install command but binary not cached** | Stays in inbox as `pending`. Agent cannot download binary while offline. Executes after reconnection and binary download. |
| **Maintenance window arrives while offline** | Executes cached commands that are due. Honors the schedule even without server contact. |
| **Server pushes new config while agent is offline** | Agent continues with last known config. Receives new config on reconnection. |
| **Agent restarts while offline** | Reads state from SQLite. Resumes pending inbox commands. Outbox items survive restart. |

#### Sync Protocol (On Reconnection)

```
Agent reconnects to server:

1. AUTHENTICATE
   → Agent presents mTLS cert + agent_id
   ← Server confirms identity

2. SYNC OUTBOX (agent → server)
   → Agent sends oldest pending outbox items (batch of up to 100)
   → For each: message_type + payload + original timestamp
   ← Server ACKs each item by outbox ID
   → Agent marks ACKed items as 'sent', deletes after confirmation
   → Repeat until outbox is empty

3. SYNC INBOX (server → agent)
   → Agent sends its last_sync_timestamp
   ← Server sends all pending commands since that timestamp
   → Agent stores in inbox, updates last_sync_timestamp

4. RESUME NORMAL OPERATION
   → Heartbeat stream resumes
   → Real-time command push resumes
```

#### Reconnection Strategy

```go
// Exponential backoff with jitter to prevent reconnection storms
type ReconnectConfig struct {
    InitialDelay  time.Duration // 1 second
    MaxDelay      time.Duration // 5 minutes
    Multiplier    float64       // 2.0
    JitterFactor  float64       // 0.2 (±20%)
}

// When 10,000 agents reconnect after a network outage, jitter
// spreads the reconnections over the MaxDelay window instead
// of all hitting the server simultaneously.
```

#### Conflict Resolution

| Conflict | Resolution |
|----------|-----------|
| Agent sends inventory from 2 hours ago, server has newer data from another source | **Server wins.** Agent inventory is timestamped; server keeps the latest. |
| Agent completed a patch install offline, but server already cancelled that deployment | **Agent result is recorded** but deployment status reflects the cancellation. The install happened — that's a fact — but the deployment is marked as cancelled with a note that some agents completed before cancellation. |
| Agent has old config, server has new config | **Server wins.** On reconnection, server pushes latest config. Agent applies it. |
| Two commands for the same package in inbox (install v1, then install v2) | **Execute in order.** Commands have sequence numbers. Agent processes inbox in order. |

#### Day 1 Rule

**The agent must never assume connectivity.** Every operation follows the pattern: do the work → write result to outbox → send when possible. The outbox is the source of truth for "what happened on this endpoint."

---

## 4. Agent Protocol Versioning

### Problem

Once agents are deployed to customer endpoints, they can't be force-updated instantly. There will always be a mix of agent versions in the field. Without protocol versioning:
- A server upgrade that changes the protobuf schema breaks all existing agents
- Rolling out new agent features requires all agents to update simultaneously (impossible)
- There's no way to deprecate old behavior gracefully

### Design: Version Negotiation + Backward Compatibility

#### Protocol Version Header

Every gRPC call includes protocol version metadata:

```protobuf
// proto/patchiq/v1/common.proto

message AgentInfo {
    string agent_id = 1;
    string agent_version = 2;       // Semantic version: "1.2.3"
    uint32 protocol_version = 3;    // Integer, monotonically increasing: 1, 2, 3...
    string os_family = 4;           // "linux", "windows", "darwin"
    string os_version = 5;
    repeated string capabilities = 6; // ["inventory", "patching", "scripting", "profiling"]
}
```

#### Version Negotiation (During Enrollment & Reconnection)

```protobuf
// Agent enrollment includes version negotiation
service AgentService {
    rpc Enroll(EnrollRequest) returns (EnrollResponse);
    rpc Heartbeat(stream HeartbeatRequest) returns (stream HeartbeatResponse);
    rpc SyncOutbox(stream OutboxMessage) returns (stream OutboxAck);
    rpc SyncInbox(InboxRequest) returns (stream InboxCommand);
}

message EnrollRequest {
    AgentInfo agent_info = 1;
    string enrollment_token = 2;
}

message EnrollResponse {
    string agent_id = 1;
    bytes  mtls_certificate = 2;
    uint32 negotiated_protocol_version = 3;  // Server picks min(agent, server)
    AgentConfig config = 4;
    UpdateInfo update_available = 5;          // Nudge to update if old
}
```

#### Compatibility Rules

```
Server protocol version: 5
Agent protocol version:  3

Negotiated version: 3 (minimum of both)

Server must support: current version AND (current - 2) at minimum
So server v5 supports protocols: 3, 4, 5
Agent on protocol 2 would be rejected with an "upgrade required" error
```

#### Capability-Based Feature Detection

Instead of only relying on version numbers, agents declare their capabilities:

```go
// Server checks capabilities before sending commands
func (s *Server) canSendCommand(agent *AgentInfo, cmd CommandType) bool {
    switch cmd {
    case CmdInstallPatch:
        return hasCapability(agent, "patching")
    case CmdRunScript:
        return hasCapability(agent, "scripting")
    case CmdApplyProfile:
        return hasCapability(agent, "profiling") // Added in agent v2.0
    case CmdCollectDiagnostics:
        return hasCapability(agent, "diagnostics") // Added in agent v2.1
    default:
        return false
    }
}
```

This is critical for UEM expansion: when you add device management capabilities to the agent, old agents simply won't declare those capabilities. The server knows not to send device management commands to them.

#### Protobuf Evolution Rules

1. **Never remove or renumber existing fields** — only deprecate with `[deprecated = true]`
2. **New fields are always optional** — old agents that don't send them get zero values
3. **New RPC methods are additive** — old agents never call them, server never sends them to old agents
4. **Breaking changes = new protocol version number** — bump the integer, add server-side handler for both old and new

#### Deprecation & Forced Updates

```
Protocol version lifecycle:
  v3: SUPPORTED (current - 2)
  v4: SUPPORTED (current - 1)
  v5: CURRENT
  v6: DEVELOPMENT (not yet released)

When server upgrades to protocol v6:
  v3: DEPRECATED — agents get "update required" warning in heartbeat response
  v4: SUPPORTED
  v5: SUPPORTED
  v6: CURRENT

90 days after deprecation:
  v3: REJECTED — agents get hard error, must update to connect
```

The server never silently drops old agents. It warns them via the heartbeat response, giving admins time to update. The timeline is configurable per deployment.

#### Day 1 Rule

**The `protocol_version` and `capabilities` fields exist in the very first protobuf definition.** Even if protocol version is 1 and capabilities is `["inventory", "patching"]` in Phase 1, the negotiation mechanism is in place for future expansion.

---

## 5. Plugin / Extension Architecture

### Problem

This is the most critical gap for UEM expansion. Currently the agent and server are monolithic — all functionality is compiled into a single binary. To evolve into UEM, we need to add:
- Device compliance checks
- Application deployment (not just patching)
- Remote control / remote terminal
- Software metering
- Endpoint encryption management
- Mobile device management (eventually)

Without extension points, each new capability requires modifying core code, increasing complexity and risk.

### Design: Module System with Registry Pattern

#### Core Concept

The agent and server are thin orchestrators. All domain functionality lives in **modules** that register themselves with the core:

```
┌────────────────────────────────────────────────────────┐
│                    Agent Core                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │Connection │  │ Scheduler│  │  Local   │             │
│  │ Manager   │  │          │  │  State   │             │
│  └──────────┘  └──────────┘  └──────────┘             │
│                                                        │
│  Module Registry                                       │
│  ┌────────────────────────────────────────────────┐    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐       │    │
│  │  │Inventory │ │ Patcher  │ │ Scripts  │       │    │
│  │  │ Module   │ │ Module   │ │ Module   │       │    │
│  │  └──────────┘ └──────────┘ └──────────┘       │    │
│  │                                                │    │
│  │  Future (UEM):                                 │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐       │    │
│  │  │ Device   │ │ App      │ │ Remote   │       │    │
│  │  │Compliance│ │ Deploy   │ │ Control  │       │    │
│  │  └──────────┘ └──────────┘ └──────────┘       │    │
│  └────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────┘
```

#### Module Interface (Agent-Side)

```go
// Every agent module implements this interface
type AgentModule interface {
    // Identity
    Name() string              // "inventory", "patcher", "scripts"
    Version() string           // "1.0.0"
    Capabilities() []string    // Reported to server during enrollment

    // Lifecycle
    Init(ctx context.Context, deps ModuleDeps) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // Command handling
    HandleCommand(ctx context.Context, cmd Command) (Result, error)
    SupportedCommands() []string  // "install_patch", "run_scan", etc.

    // Data collection (periodic)
    Collect(ctx context.Context) ([]OutboxMessage, error)
    CollectInterval() time.Duration  // How often to run Collect()

    // Health
    HealthCheck(ctx context.Context) error
}

// Dependencies injected by the core
type ModuleDeps struct {
    Logger     *slog.Logger
    LocalDB    *sql.DB          // SQLite handle
    Outbox     OutboxWriter     // Write to outbox queue
    Config     ConfigProvider   // Read module-specific config
    EventBus   EventEmitter     // Emit local events
    FileCache  FileCache        // Download and cache files from server
}
```

#### Module Registration (Compile-Time)

For Phase 1, modules are compiled into the binary (not dynamically loaded). Registration happens in `main.go`:

```go
// cmd/agent/main.go
func main() {
    core := agent.NewCore(config)

    // Register modules — this is the extension point
    core.RegisterModule(inventory.New())
    core.RegisterModule(patcher.New())
    core.RegisterModule(scripts.New())

    // Future UEM modules would be added here:
    // core.RegisterModule(compliance.New())
    // core.RegisterModule(appdeploy.New())
    // core.RegisterModule(remotecontrol.New())

    core.Run(ctx)
}
```

**Why compile-time, not dynamic plugins?**
- Go's `plugin` package is fragile (same Go version, same build flags, Linux-only)
- Compile-time registration is simple, testable, and type-safe
- Agent binary is already cross-compiled per OS — adding modules is just adding imports
- License gating controls which modules are active, not which are compiled in

#### Module Interface (Server-Side)

```go
// Server modules handle the server-side logic for each domain
type ServerModule interface {
    // Identity
    Name() string
    Version() string

    // Lifecycle
    Init(ctx context.Context, deps ServerModuleDeps) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // API routes — module registers its own HTTP/gRPC handlers
    RegisterRoutes(router *http.ServeMux)
    RegisterGRPCServices(server *grpc.Server)

    // Database — module owns its own tables
    MigrationSource() fs.FS  // Embedded SQL migrations for this module's tables

    // Event handling — module subscribes to events it cares about
    EventSubscriptions() map[string]EventHandler
}

// Server-side dependencies
type ServerModuleDeps struct {
    Logger     *slog.Logger
    DB         *sql.DB
    Redis      *redis.Client
    EventBus   EventBus
    License    LicenseChecker   // Check if this module's features are licensed
    RBAC       RBACChecker      // Check permissions for this module's resources
    Config     ConfigProvider
    Telemetry  TelemetryProvider
}
```

#### Module Isolation Rules

1. **Modules don't import each other directly.** They communicate through the event bus or through well-defined interfaces in `internal/common/`.
2. **Each module owns its database tables.** The patcher module owns `deployments`, `deployment_waves`, `patch_results`. The inventory module owns `endpoint_inventory`, `software_catalog_cache`. Modules don't query each other's tables directly.
3. **Modules declare their RBAC resources.** The patcher module registers `deployments` as a resource with actions `create`, `approve`, `cancel`. The RBAC system discovers these at startup.
4. **Modules declare their license features.** The patcher module requires feature `patching`. If the license doesn't include it, the module's routes return 403 and its agent commands are not sent.

#### Why This Matters for UEM

When you're ready to add device compliance:

1. Create `internal/agent/compliance/` — implements `AgentModule`
2. Create `internal/server/compliance/` — implements `ServerModule`
3. Add `compliance.New()` to both agent and server registration
4. Add new protobuf commands: `check_compliance`, `remediate`
5. Add `compliance` to the capabilities list and license features
6. The module gets its own DB tables, API routes, events, RBAC resources — all isolated

**Zero changes to existing patch management code.** That's the goal.

#### Day 1 Rule

**Agent core and server core must be thin.** Business logic lives in modules. The core handles: connection, scheduling, config, module lifecycle, event routing, and outbox/inbox management. Nothing else.

---

## 6. Idempotent Operations

### Problem

In distributed systems, messages get delivered more than once (network retries, agent reconnection, queue replay). Without idempotency:
- A patch install command retried after a timeout might install the same patch twice (usually harmless but wastes time and creates confusing audit trails)
- A deployment creation API call retried by a flaky frontend creates duplicate deployments
- An approval webhook retried by the ITSM tool approves twice, potentially advancing waves prematurely

### Design: Idempotency Keys + Idempotent Receivers

#### API-Level Idempotency (HTTP)

All mutating API endpoints accept an `Idempotency-Key` header:

```go
// Middleware for idempotent POST/PUT/PATCH requests
func IdempotencyMiddleware(store IdempotencyStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method == "GET" || r.Method == "DELETE" {
                next.ServeHTTP(w, r) // GET is naturally idempotent, DELETE is idempotent
                return
            }

            key := r.Header.Get("Idempotency-Key")
            if key == "" {
                next.ServeHTTP(w, r) // No key = no idempotency guarantee
                return
            }

            // Check if we've already processed this key
            if cached, found := store.Get(r.Context(), key); found {
                // Return the cached response
                w.WriteHeader(cached.StatusCode)
                w.Write(cached.Body)
                return
            }

            // Capture the response
            rec := &responseRecorder{ResponseWriter: w}
            next.ServeHTTP(rec, r)

            // Cache the response for this key (TTL: 24 hours)
            store.Set(r.Context(), key, CachedResponse{
                StatusCode: rec.statusCode,
                Body:       rec.body.Bytes(),
            }, 24*time.Hour)
        })
    }
}
```

Storage: Redis with TTL. Key format: `idempotency:{tenant_id}:{key}`.

#### Command-Level Idempotency (Agent)

Every command sent to an agent has a unique `command_id`. The agent tracks executed command IDs:

```go
// Agent-side: before executing any command
func (e *CommandExecutor) Execute(ctx context.Context, cmd Command) (Result, error) {
    // Check if already executed
    if result, found := e.localDB.GetCommandResult(cmd.ID); found {
        slog.InfoContext(ctx, "command already executed, returning cached result",
            "command_id", cmd.ID,
            "command_type", cmd.Type,
        )
        return result, nil
    }

    // Execute
    result, err := e.dispatch(ctx, cmd)

    // Store result (even if failed — a failure is a valid result for idempotency)
    e.localDB.StoreCommandResult(cmd.ID, result, err)

    return result, err
}
```

#### Patch Installation Idempotency

Patch installs are naturally idempotent if designed correctly:

```go
func (p *Patcher) InstallPatch(ctx context.Context, patch PatchSpec) (PatchResult, error) {
    // Step 1: Check if already at target version (idempotent check)
    currentVersion, err := p.detector.GetInstalledVersion(patch.PackageName)
    if err == nil && currentVersion == patch.TargetVersion {
        return PatchResult{
            Status:  "already_installed",
            Message: fmt.Sprintf("%s is already at version %s", patch.PackageName, patch.TargetVersion),
        }, nil
    }

    // Step 2: Install
    result, err := p.installer.Install(ctx, patch)
    if err != nil {
        return PatchResult{Status: "failed", Error: err.Error()}, err
    }

    // Step 3: Verify
    newVersion, err := p.detector.GetInstalledVersion(patch.PackageName)
    if err != nil || newVersion != patch.TargetVersion {
        return PatchResult{Status: "verification_failed"}, fmt.Errorf("version mismatch after install")
    }

    return PatchResult{Status: "success", InstalledVersion: newVersion}, nil
}
```

The key: **check desired state before acting.** If the system is already in the desired state, report success without doing anything. This makes every retry safe.

#### Deployment State Machine

Deployments follow a strict state machine. Each transition is guarded:

```
CREATED → PENDING_APPROVAL → APPROVED → WAVE_1_RUNNING → WAVE_1_COMPLETE
    → WAVE_2_RUNNING → ... → COMPLETED

Transitions:
  CREATED → PENDING_APPROVAL: only if currently CREATED
  PENDING_APPROVAL → APPROVED: only if currently PENDING_APPROVAL
  APPROVED → WAVE_1_RUNNING: only if currently APPROVED
```

```go
// State transition with optimistic locking
func (s *DeploymentStore) Transition(ctx context.Context, id string, from, to DeploymentStatus) error {
    result, err := s.db.ExecContext(ctx,
        `UPDATE deployments SET status = $1, updated_at = now()
         WHERE id = $2 AND status = $3`, // WHERE status = from prevents double-transition
        to, id, from,
    )
    if rows, _ := result.RowsAffected(); rows == 0 {
        return ErrStateTransitionConflict // Someone already transitioned it
    }
    return err
}
```

If two concurrent requests both try to approve a deployment, only one succeeds. The second gets a conflict error, which is safe to retry (it'll see the deployment is already approved).

#### Day 1 Rule

**All write operations must be safe to execute twice.** Pattern: check current state → act only if needed → verify result. Every command has an ID. Every state transition uses optimistic locking.

---

## 7. Configuration Hierarchy

### Problem

Without a configuration hierarchy, every setting is flat — either global or per-endpoint with no inheritance. This fails for real enterprises:
- "All endpoints scan daily, except the finance group which scans hourly, except this one PCI server which scans every 15 minutes"
- "Default maintenance window is Sunday 2am, but the EU tenant uses Saturday 3am, and the London office uses Saturday 1am"
- MSPs need tenant-level defaults that clients can partially override

### Design: 4-Level Config Hierarchy with Merge Semantics

#### Hierarchy Levels

```
Level 0: SYSTEM DEFAULTS     (compiled into the binary)
    ↓ overridden by
Level 1: TENANT DEFAULTS      (set by MSP or platform admin)
    ↓ overridden by
Level 2: GROUP SETTINGS        (set by client admin per endpoint group)
    ↓ overridden by
Level 3: ENDPOINT OVERRIDES    (set per specific endpoint)
```

The effective configuration for any endpoint is computed by merging all four levels, with lower levels overriding higher ones.

#### Config Schema

```go
// Configuration is a structured map, not arbitrary key-value
type PatchConfig struct {
    ScanSchedule      *string        `json:"scan_schedule,omitempty"`       // cron expression
    MaintenanceWindow *TimeWindow    `json:"maintenance_window,omitempty"`
    AutoReboot        *bool          `json:"auto_reboot,omitempty"`
    RebootDelay       *Duration      `json:"reboot_delay,omitempty"`
    MaxConcurrent     *int           `json:"max_concurrent_installs,omitempty"`
    WaveStrategy      *WaveStrategy  `json:"wave_strategy,omitempty"`
    NotifyUser        *bool          `json:"notify_user_before_reboot,omitempty"`
    ExcludedPackages  []string       `json:"excluded_packages,omitempty"`   // additive across levels
    PreScript         *string        `json:"pre_script,omitempty"`
    PostScript        *string        `json:"post_script,omitempty"`
    BandwidthLimit    *string        `json:"bandwidth_limit,omitempty"`     // e.g., "10mbps"
}

// Using pointer fields (*string, *bool, *int) so we can distinguish
// "not set at this level" (nil) from "explicitly set to zero/false/empty"
```

#### Database Storage

```sql
CREATE TABLE config_overrides (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    scope_type  TEXT NOT NULL,  -- 'tenant', 'group', 'endpoint'
    scope_id    UUID NOT NULL,  -- tenant_id, group_id, or endpoint_id
    module      TEXT NOT NULL,  -- 'patching', 'inventory', 'agent', etc.
    config      JSONB NOT NULL, -- partial config (only overridden fields)
    updated_by  UUID,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, scope_type, scope_id, module)
);

CREATE INDEX idx_config_scope ON config_overrides(tenant_id, scope_type, scope_id);
```

#### Resolution Algorithm

```go
// Resolve effective config for an endpoint
func (s *ConfigService) ResolveConfig(
    ctx context.Context,
    tenantID, groupID, endpointID string,
    module string,
) (*PatchConfig, error) {
    // Start with compiled defaults
    effective := DefaultPatchConfig()

    // Layer 1: Tenant-level overrides
    if tenantCfg, err := s.store.GetConfig(ctx, "tenant", tenantID, module); err == nil {
        mergeConfig(effective, tenantCfg)
    }

    // Layer 2: Group-level overrides
    if groupCfg, err := s.store.GetConfig(ctx, "group", groupID, module); err == nil {
        mergeConfig(effective, groupCfg)
    }

    // Layer 3: Endpoint-level overrides
    if endpointCfg, err := s.store.GetConfig(ctx, "endpoint", endpointID, module); err == nil {
        mergeConfig(effective, endpointCfg)
    }

    return effective, nil
}

// Merge: non-nil fields in override replace fields in base
func mergeConfig(base, override *PatchConfig) {
    if override.ScanSchedule != nil {
        base.ScanSchedule = override.ScanSchedule
    }
    if override.AutoReboot != nil {
        base.AutoReboot = override.AutoReboot
    }
    if override.MaintenanceWindow != nil {
        base.MaintenanceWindow = override.MaintenanceWindow
    }
    // Additive fields: append, don't replace
    if len(override.ExcludedPackages) > 0 {
        base.ExcludedPackages = append(base.ExcludedPackages, override.ExcludedPackages...)
    }
    // ... all fields
}
```

#### Merge Semantics

| Field Type | Merge Behavior |
|-----------|---------------|
| Scalar (`*string`, `*bool`, `*int`) | Lower level replaces higher level. `nil` means "inherit from parent." |
| Slice (`[]string`) | **Additive.** Group-level excluded packages are added to tenant-level excluded packages. To remove an inherited value, use a "deny" mechanism (e.g., `excluded_packages_remove`). |
| Struct (e.g., `TimeWindow`) | Replaced as a whole. If group sets a maintenance window, it fully replaces the tenant's window. |

#### API: View Effective Config + Source

The admin UI should show not just the effective value, but **where each value comes from**:

```json
{
  "scan_schedule": {
    "effective_value": "0 */6 * * *",
    "source": "group:finance-servers",
    "inherited_from": null
  },
  "auto_reboot": {
    "effective_value": false,
    "source": "tenant:acme-corp",
    "inherited_from": "system_default was true, overridden at tenant level"
  },
  "maintenance_window": {
    "effective_value": {"day": "sunday", "start": "02:00", "end": "06:00"},
    "source": "system_default",
    "inherited_from": null
  },
  "excluded_packages": {
    "effective_value": ["kernel", "glibc", "openssl"],
    "sources": [
      {"value": "kernel", "source": "tenant:acme-corp"},
      {"value": "glibc", "source": "group:finance-servers"},
      {"value": "openssl", "source": "endpoint:srv-fin-01"}
    ]
  }
}
```

This transparency is critical — admins need to understand why a setting has a particular value and which level to change it at.

#### Config Push to Agents

When a config changes at any level, the server:
1. Computes the new effective config for all affected endpoints
2. Pushes the effective config to affected agents via the gRPC command stream
3. Agent stores the new config in SQLite (for offline use) and applies it

Config changes emit a `config.changed` domain event with before/after diff for the audit trail.

#### Module-Scoped Config

Each module owns its own config namespace. The config hierarchy applies independently per module:

```
patching.scan_schedule = "0 */6 * * *"   (from group)
patching.auto_reboot = false             (from tenant)
inventory.collect_interval = "1h"         (from system default)
agent.log_level = "debug"                 (from endpoint override)
```

This aligns with the module system (Gap #5) — when you add a UEM device compliance module, it gets its own config namespace (`compliance.check_interval`, `compliance.remediate_on_failure`, etc.) with full hierarchy support automatically.

#### Day 1 Rule

**No flat config lookups.** Every config read goes through `ResolveConfig()` which walks the hierarchy. Even in Phase 1 with single-tenant deployments, the hierarchy exists (system defaults → tenant defaults → group → endpoint). The tenant and endpoint levels may be empty, but the resolution path is exercised.

---

## Summary: Implementation Priority

All 7 gaps must be scaffolded in Phase 1 (Month 1). Here's the suggested order:

| Priority | Gap | Why This Order |
|----------|-----|----------------|
| 1 | **Multi-tenancy** | Every table depends on `tenant_id`. Must be in the first migration. |
| 2 | **Event bus + Audit** | Every handler emits events. Must exist before writing handlers. |
| 3 | **Module system** | Agent core and server core structure depends on this. Must be in place before writing business logic. |
| 4 | **Config hierarchy** | Modules need config. Config resolution must exist before modules read config. |
| 5 | **Protocol versioning** | Protobuf definitions are the first thing written. Version fields must be there from proto v1. |
| 6 | **Idempotency** | Middleware + patterns applied as handlers are written. |
| 7 | **Offline-first agent** | Agent SQLite schema + outbox/inbox designed when agent skeleton is built. |

These are not features. They are **infrastructure that all features build on.** Skipping any of them creates technical debt that compounds with every feature added.
