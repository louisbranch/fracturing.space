# Duality Engine Documentation

Duality Engine provides a server-authoritative implementation of the
Duality Dice system, exposed via gRPC and MCP.

It focuses on deterministic resolution of mechanics and campaign state
management for LLM and traditional clients.

------------------------------------------------------------------------

## Overview

The engine supports:

-   Deterministic and probabilistic Duality resolution
-   Campaign, session, participant, and actor management
-   MCP integration for AI/LLM clients
-   Persistent storage via BoltDB
-   Reproducible and auditable outcomes

------------------------------------------------------------------------

## API Surface

### Mechanics

-   duality_action_roll
-   duality_outcome
-   duality_explain
-   duality_probability
-   duality_rules_version
-   roll_dice

### Campaign Runtime

Resources:

-   campaigns://list
-   campaign://{campaign_id}
-   campaign://{campaign_id}/participants
-   campaign://{campaign_id}/actors
-   campaign://{campaign_id}/sessions

Tools:

-   campaign_create
-   participant_create
-   actor_create
-   actor_control_set
-   session_start

### MCP Context

Resources:

-   context://current

Tools:
-   set_context

------------------------------------------------------------------------

## Documentation

-   [Getting started](getting-started.md)
-   [Configuration](configuration.md)
-   [MCP tools and resources](mcp.md)
-   [Integration tests](integration-tests.md)
