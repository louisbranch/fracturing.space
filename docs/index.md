---
title: ""
---

Fracturing.Space provides a server-authoritative implementation of the
Duality Dice system, exposed via gRPC and MCP.

It focuses on deterministic resolution of mechanics and campaign state
management for LLM and traditional clients.

## Start here

- [Getting started](running/getting-started.md)
- [Configuration](running/configuration.md)
- [Seeding the database](running/seeding.md)
- [Integration tests](running/integration-tests.md)
- [Architecture](project/architecture.md)
- [Domain language](project/domain-language.md)
- [Event replay and snapshots](project/event-replay.md)
- [Game systems architecture](project/game-systems.md)
- [MCP tools and resources](reference/mcp.md)

## Key concepts

- Deterministic and probabilistic Duality resolution
- Campaign, session, participant, and character management
- MCP integration for AI/LLM clients
- Persistent storage via SQLite
- Reproducible and auditable outcomes via the event journal

## Reference

For the full MCP tool/resource catalog and HTTP endpoint details, see
[MCP tools and resources](reference/mcp.md).
