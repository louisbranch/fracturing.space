---
title: "Identity and OAuth"
parent: "Platform surfaces"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Identity and OAuth

Canonical identity, recovery, and OAuth architecture for Fracturing.Space.

## Purpose

Define service ownership, security boundaries, and invariants for:

- account identity (`user`, email, passkeys, locale)
- account recovery (magic links)
- OAuth authorization server behavior (first-party clients)
- OAuth external provider login/linking (Google/GitHub)
- MCP protected-resource token validation

## Ownership boundaries

- **Auth service** is source of truth for identity and access primitives:
  users, emails, passkeys, magic links, sessions, OAuth issuance/introspection,
  and external identity linking.
- **MCP service** is a protected resource that validates bearer tokens through
  auth introspection and exposes OAuth protected-resource metadata.
- **Web service** hosts login/recovery UX and delegates verification/storage to auth.
- **Social service** owns discovery/profile metadata (display identity), not
  authentication or authorization.

Boundary rules:

1. If a field proves identity or grants/denies access, it belongs to `auth`.
2. If a field is social/discovery profile metadata, it belongs to `social`.
3. Account preferences (for example locale) are account data and belong to `auth`.

## Identity model

- **User**: canonical identity record keyed by user ID.
- **Primary email**: required in this release for account creation and recovery.
- **Passkeys**: primary authentication credential (multiple per user allowed).
- **User locale**: private account preference on the user record.
- **Additional emails**: planned extension; out of scope for current contract.

## Recovery model

- **Magic links** are single-use tokens with expiration and used-at tracking.
- Consuming a magic link verifies email and may attach to pending OAuth authorization.
- Deny-by-default: expired/used/invalid tokens cannot establish authenticated state.

## OAuth surfaces

### OAuth server (auth service)

Auth service acts as authorization server for first-party clients.

Endpoints:

- `GET /authorize` + `POST /authorize/consent`
- `POST /token`
- `POST /introspect` (protected by `X-Resource-Secret`)
- `GET /.well-known/oauth-authorization-server`

Token model:

- Access tokens are opaque and persisted in auth storage.
- Protected resources (for example MCP HTTP transport) validate via `/introspect`.

### OAuth client (external providers)

Auth service may act as OAuth client for provider login and account linking.

Endpoints:

- `GET /oauth/providers/{provider}/start`
- `GET /oauth/providers/{provider}/callback`

External identities are linked to internal users in auth storage.

### MCP protected resource

Endpoint:

- `GET /.well-known/oauth-protected-resource`

401 responses include `WWW-Authenticate: Bearer resource_metadata=...`.

## Operational invariants

- Public auth pages treat users as authenticated only after auth-session validation.
- Protected resource token checks fail closed when introspection is unavailable.
- OAuth override/privileged operations require explicit telemetry and reason codes.
- Identity and OAuth docs should not duplicate environment default inventories.

## Configuration

Authoritative environment defaults and wiring values live in:
[Running configuration](../../running/configuration.md).

Use this page for boundaries and behavior semantics; keep variable inventories in
running docs.
