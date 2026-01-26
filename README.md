# Duality Engine

![Coverage](../../raw/badges/coverage.svg)

Duality Engine is a server-authoritative implementation of the Duality
Dice system used in Daggerheart. It provides deterministic, auditable
mechanical outcomes via gRPC and an MCP (stdio JSON-RPC) bridge.

This project focuses on rules adjudication and campaign state
management. It does not generate narrative content and does not bundle
Daggerheart SRD material.

Documentation: https://louisbranch.github.io/duality-engine/

------------------------------------------------------------------------

## Quickstart

Run gRPC server and MCP bridge:

    make run

Run individually:

    go run ./cmd/server
    go run ./cmd/mcp

Default endpoints:

-   gRPC: localhost:8080
-   MCP: stdio

------------------------------------------------------------------------

## Capabilities

### Mechanics

-   duality_action_roll
-   duality_outcome
-   duality_explain
-   duality_probability
-   duality_rules_version
-   roll_dice

### Campaign Runtime

Tools:

-   campaign_create
-   participant_create
-   character_create
-   character_control_set
-   session_start

Resources:

-   campaigns://list
-   campaign://{campaign_id}
-   campaign://{campaign_id}/participants
-   campaign://{campaign_id}/characters
-   campaign://{campaign_id}/sessions

### MCP Context

Tools:

-   set_context (in-memory, resets on restart)

Resources:

-   context://current (current MCP execution context; ephemeral, resets on restart)
------------------------------------------------------------------------

## State Model

Persisted (BoltDB):

-   Campaigns
-   Participants
-   Characters
-   Sessions

Ephemeral:

-   MCP execution context

------------------------------------------------------------------------

## Configuration

See: [Configuration](docs/configuration.md)

Environment variables:

-   DUALITY_DB_PATH (default: data/duality.db)
-   DUALITY_GRPC_ADDR (default: localhost:8080)

------------------------------------------------------------------------

## Documentation

-   [Getting started](docs/getting-started.md)
-   [Configuration](docs/configuration.md)
-   [MCP tools and resources](docs/mcp.md)
-   [Integration tests](docs/integration-tests.md)

------------------------------------------------------------------------

## Near-term Roadmap

-   Publish prebuilt binaries
-   Add HTTP transport alongside gRPC
-   Complete campaign lifecycle tools
-   Improve MCP context handling for multi-client use
-   Expand telemetry and request tracing

------------------------------------------------------------------------

## Attribution and Licensing

Duality Engine is an independent, fan-made project and is not affiliated
with Critical Role Productions LLC, Darrington Press, or their partners.

Daggerheart is a trademark of Critical Role Productions LLC.

This project is intended for use under the Darrington Press Community
Gaming License.

Source code is licensed under the MIT License. See [LICENSE](LICENSE).

All trademarks, artwork, and copyrighted material remain the property of
their respective owners.

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

Authors: [AUTHORS](AUTHORS).
