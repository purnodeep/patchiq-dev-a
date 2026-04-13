# ADR-003: MCP Protocol for AI Integration

## Status

Accepted

## Context

PatchIQ needs an AI assistant that can operate the platform through natural language. We needed to choose an integration pattern for connecting the LLM to platform functionality.

## Decision

Use the Model Context Protocol (MCP) with Claude API. TypeScript MCP server wrapping Go API calls, using Streamable HTTP transport.

## Consequences

- **Positive**: Standard protocol (not locked to one LLM provider); human-in-the-loop built into MCP design; tool annotations (readOnlyHint, destructiveHint) enable safety patterns; MCP Resources provide context without tool calls
- **Negative**: MCP is relatively new; TypeScript MCP server adds a language boundary; streaming adds complexity

## Alternatives Considered

- **Direct Claude API function calling**: Simpler but proprietary — rejected because MCP provides a standard protocol layer
- **LangChain/LlamaIndex**: Framework-based — rejected because too much abstraction for what is essentially tool calling
- **Custom REST-to-LLM bridge**: Build our own — rejected because MCP already solves this well
