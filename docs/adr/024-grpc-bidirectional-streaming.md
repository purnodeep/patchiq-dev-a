# ADR-024: gRPC Bidirectional Streaming for Agent-Server Communication

## Status

Accepted

## Context

PatchIQ agents communicate with the Patch Manager server for enrollment, heartbeats, outbox message sync, and command reception. The protocol must handle thousands of concurrent agents, support bidirectional communication (server pushes commands/config to agents), work efficiently over unreliable networks, minimize bandwidth for periodic heartbeats, and integrate with the agent's offline-first outbox pattern (ADR-021). The agent binary must remain minimal.

## Decision

### gRPC with Protocol Buffers

Use gRPC (over HTTP/2) with Protocol Buffers for all agent-server communication. The service definition lives in `proto/patchiq/v1/agent.proto`. The `AgentService` provides four RPCs with three distinct communication patterns:

### 1. Enroll — Unary RPC

```protobuf
rpc Enroll(EnrollRequest) returns (EnrollResponse);
```

One-shot request/response for agent registration. The agent sends its `AgentInfo` (version, protocol version, capabilities), an `enrollment_token`, and `EndpointInfo` (hostname, OS). The server validates the token, creates an endpoint record, negotiates protocol version, and returns the assigned `agent_id` and `AgentConfig`.

**Protocol version negotiation**: The server computes `min(agent_protocol_version, server_max_protocol_version)` during enrollment (`internal/server/grpc/enroll.go:60-73`). If the agent's version is below `ServerMinProtocolVersion`, enrollment fails with `ENROLLMENT_ERROR_CODE_PROTOCOL_VERSION_INCOMPATIBLE`. The negotiated version is persisted in the agent's SQLite `agent_state` table and sent in subsequent heartbeat/outbox messages.

**Idempotent re-enrollment**: If an endpoint with the same hostname and OS already exists for the tenant, the server returns the existing `agent_id` (`enroll.go:87-105`). The agent also skips the RPC entirely if `agent_id` is already stored locally (`internal/agent/comms/enroll.go:30-47`).

### 2. Heartbeat — Bidirectional Streaming

```protobuf
rpc Heartbeat(stream HeartbeatRequest) returns (stream HeartbeatResponse);
```

Long-lived bidirectional stream for agent liveness and server-to-agent control. The agent sends periodic `HeartbeatRequest` messages containing status, resource usage (`runtime.MemStats`), uptime, and `offline_queue_depth` (outbox pending count). The server responds with `HeartbeatResponse` messages carrying:

- **`commands_pending`**: When > 0, triggers `OnCommandsPending` callback which calls `SyncRunner.Trigger()` for immediate outbox drain and `FetchInbox()` for command retrieval (`internal/agent/comms/heartbeat.go:183-188`).
- **`HeartbeatDirective`**: Server-pushed control instructions:
  - `RE_ENROLL` — agent must clear its `agent_id` and re-register
  - `SHUTDOWN` — orderly agent shutdown
  - `UPDATE_REQUIRED` — mandatory agent binary update
  - `PROTOCOL_UNSUPPORTED` — triggers re-enrollment

The heartbeat uses concurrent sender/receiver goroutines (`internal/agent/comms/heartbeat.go:75-82`) with a child context for coordinated cancellation. The heartbeat interval is server-configurable via `AgentConfig.heartbeat_interval_seconds` returned during enrollment (default 60 seconds, applied in `internal/agent/comms/client.go:267-269`).

### 3. SyncOutbox — Bidirectional Streaming

```protobuf
rpc SyncOutbox(stream OutboxMessage) returns (stream OutboxAck);
```

Message-by-message synchronization of the agent's outbox queue. The agent sends an `OutboxMessage` (with `message_id`, `protocol_version`, `type`, `payload`, `timestamp`), waits for an `OutboxAck`, then processes the next item. The server identifies the agent via `x-agent-id` gRPC metadata (`internal/server/grpc/sync_outbox.go:31-39`).

Message types: `INVENTORY` (parsed and persisted), `COMMAND_RESULT` (emitted as domain event for deployment state machine), `HEARTBEAT` (offline snapshot), `EVENT`.

Rejection codes control retry behavior: `UNSPECIFIED` = accepted, `SERVER_OVERLOADED` = transient (retry with backoff), all others = permanent (discard). See ADR-021 for dead-lettering details.

### 4. SyncInbox — Server Streaming

```protobuf
rpc SyncInbox(InboxRequest) returns (stream CommandRequest);
```

Server-to-agent command delivery. The agent sends a single `InboxRequest` with its `agent_id` and `last_received_id` for cursor-based pagination. The server streams pending `CommandRequest` messages.

### Connection Lifecycle

The full agent lifecycle (`internal/agent/comms/client.go:231-337`): generate TLS cert, connect with retry, enroll with retry, then enter the heartbeat loop. On heartbeat stream failure, the agent reconnects with exponential backoff (ADR-021 parameters). On `RE_ENROLL` directive, the agent clears its `agent_id` and re-enters enrollment. The sync runner runs as a child goroutine tied to the heartbeat lifecycle — when the heartbeat fails, the sync runner is cancelled and recreated on reconnect.

## Consequences

- **Positive**: HTTP/2 multiplexing allows heartbeat and outbox sync over a single TCP connection; bidirectional streaming enables server-push without polling; Protocol Buffers provide compact binary serialization (critical for thousands of agents); strong typing via protobuf prevents schema drift between agent and server; HeartbeatDirective system enables server-side fleet control without separate management channel; protocol version negotiation allows independent agent/server upgrades
- **Negative**: gRPC adds complexity vs simple REST (protoc toolchain, code generation); bidirectional streams are harder to debug than request/response; HTTP/2 requirement may complicate some proxy/load balancer configurations; stream reconnection logic adds code complexity in the agent

## Alternatives Considered

- **REST/HTTP polling**: Simplest — rejected because polling for commands wastes bandwidth and adds latency; no server-push capability; separate endpoint per operation vs multiplexed streams
- **WebSocket**: Bidirectional — rejected because no built-in schema enforcement (Protocol Buffers); manual message framing; weaker ecosystem for Go server/client code generation
- **MQTT**: IoT standard — rejected because adds a message broker dependency (Mosquitto/EMQX); agent must be self-contained; MQTT QoS levels add complexity without matching gRPC's type safety
- **gRPC server-side streaming only**: Simpler — rejected because outbox sync requires client-to-server streaming with per-message acknowledgment; heartbeat requires bidirectional communication for directives
