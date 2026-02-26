---
title: "OAuth System"
parent: "Architecture"
nav_order: 15
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# OAuth System

This document describes the OAuth surface owned by the auth service and the MCP protected resource integration.

## Goals

- Auth service acts as an OAuth authorization server for first-party clients (MCP, web tools).
- Auth service can act as an OAuth client to external providers (Google, GitHub) to bootstrap or link identities.
- MCP HTTP transport advertises resource metadata and validates access tokens via introspection.

## Boundaries

- **Auth service**: owns OAuth authorization, token issuance, and external provider login.
- **MCP service**: a protected resource that validates bearer tokens and advertises `.well-known` metadata.
- **Game service**: unchanged; consumes auth identity for join grants and permissions.

## OAuth Server (Auth Service)

Endpoints:

- `GET /authorize` + `POST /authorize/consent`
- `POST /token`
- `POST /introspect` (opaque tokens, protected by `X-Resource-Secret`)
- `GET /.well-known/oauth-authorization-server`

Token model:

- Access tokens are opaque and stored in SQLite.
- MCP validates tokens using `/introspect`.

## OAuth Client (External Providers)

Endpoints:

- `GET /oauth/providers/{provider}/start`
- `GET /oauth/providers/{provider}/callback`

External identities are stored in the auth database and linked to internal users.

## MCP Protected Resource Metadata

Endpoint:

- `GET /.well-known/oauth-protected-resource`

The MCP HTTP server advertises its authorization server and includes
`WWW-Authenticate: Bearer resource_metadata=...` on 401 responses.

## Configuration (Env)

Authoritative defaults live in
[running/configuration.md](../running/configuration.md).

Use this page for OAuth behavior and ownership semantics, not duplicated default
inventories.

Critical wiring values to verify in all OAuth deployments:

- `FRACTURING_SPACE_OAUTH_ISSUER` (auth-server issuer metadata)
- `FRACTURING_SPACE_OAUTH_RESOURCE_SECRET` (auth introspection secret)
- `FRACTURING_SPACE_MCP_OAUTH_ISSUER` (MCP introspection issuer target)
- `FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET` (MCP introspection secret)

For provider login, passkey, and magic-link variables, use the canonical
configuration page above.

## Example OAuth Client Config (JSON)

```json
[
  {
    "client_id": "claude-desktop",
    "redirect_uris": ["http://localhost:8081/oauth/callback"],
    "client_name": "Claude Desktop",
    "token_endpoint_auth_method": "none"
  }
]
```
