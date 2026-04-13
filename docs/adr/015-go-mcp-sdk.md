# ADR-015: Official Go MCP SDK over TypeScript MCP Server

## Status

Accepted

## Context

PatchIQ's AI assistant uses the Model Context Protocol (MCP) to connect Claude to platform operations. The original plan was a TypeScript MCP server wrapping Go API calls. This introduces a language boundary (TypeScript sidecar alongside Go backend), complicates deployment (Node.js runtime required on-prem), and increases operational surface area.

Two Go MCP SDKs now exist:
- `mark3labs/mcp-go` (community, v0.44.0, ~8.2k stars)
- `modelcontextprotocol/go-sdk` (official, backed by Google + MCP org)

## Decision

Use the official `modelcontextprotocol/go-sdk` to build MCP servers in Go. This eliminates the TypeScript↔Go language boundary entirely. The MCP server runs in-process within the Go backend — no separate sidecar or Node.js runtime.

## Consequences

- **Positive**: Single language for entire platform; compiled binary deployment simplifies on-prem; no Node.js runtime dependency; lower resource consumption than TypeScript MCP server; official SDK tracks MCP spec evolution; uses Go team's battle-tested JSON-RPC implementation from gopls
- **Negative**: Both Go SDKs are pre-1.0; MCP specification is still evolving; fewer examples and community resources compared to TypeScript SDK; may need to wrap SDK behind an interface for future swapability

## Alternatives Considered

- **TypeScript MCP server (original plan)**: Most examples, largest community — rejected because adds Node.js as a deployment dependency; language boundary complicates debugging and deployment
- **mark3labs/mcp-go (community SDK)**: More mature, more stars — rejected in favor of official SDK because official SDK will be the long-term maintained option and tracks the spec more closely
