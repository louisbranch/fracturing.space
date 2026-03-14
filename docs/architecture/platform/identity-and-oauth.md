---
title: "Identity and OAuth"
parent: "Platform surfaces"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
---

# Identity and OAuth

Canonical identity, passkey, recovery, and OAuth architecture for Fracturing.Space.

## Purpose

Define service ownership, security boundaries, and invariants for:

- account identity (`user`, username, passkeys, locale)
- offline account recovery (single-use recovery code)
- first-party OAuth authorization-server behavior
- MCP protected-resource token validation

## Ownership boundaries

- **Auth service** is source of truth for identity and access primitives:
  users, usernames, passkeys, recovery-code state, web sessions, and OAuth
  issuance/introspection.
- **MCP service** is a protected resource that validates bearer tokens through
  auth introspection and exposes OAuth protected-resource metadata.
- **Web service** hosts signup/login/settings UX and delegates passkey and
  recovery verification/storage to auth.
- **Social service** owns optional public profile metadata (display name,
  pronouns, bio, avatar metadata), contact relationships, and authenticated
  people-search read models. It does not own authentication, authorization, or
  username truth.
- **Discovery service** remains the public browsing surface for non-authenticated
  discovery entries. Public profile routing does not imply a separate public
  account type or a discovery-owned username index.

Boundary rules:

1. If a field proves identity or grants/denies access, it belongs to `auth`.
2. If a field is profile metadata, contact ranking state, or authenticated
   people-search projection data, it belongs to `social`.
3. Account preferences (for example locale) are account data and belong to `auth`.
4. If a surface is public browsing or discovery indexing, it belongs to
   `discovery`, but it must consume published public data rather than becoming
   the source of username ownership.

## Identity model

- **User**: canonical identity record keyed by user ID.
- **Username**: immutable auth-owned account locator and public handle.
- **Passkeys**: primary authentication credential; multiple credentials may be
  registered to one account.
- **User locale**: private account preference on the user record.
- **Public profile**: baseline profile exists as soon as the account exists;
  social metadata is optional enrichment, not a prerequisite for profile
  routing.
- **Authenticated people search**: social-owned read model keyed by auth-owned
  usernames and enriched with contact/profile metadata for invite and mention
  UX.

## Passkey and recovery model

Signup and login are username-first WebAuthn ceremonies:

1. `BeginAccountRegistration(username, locale)` reserves the username and
   returns WebAuthn creation options.
2. `FinishAccountRegistration(session_id, credential_response)` creates the
   user, stores the first passkey, creates the web session, and returns the
   recovery code once.
3. `BeginPasskeyLogin(username)` returns assertion options for the account’s
   registered passkeys.
4. `FinishPasskeyLogin(...)` verifies the assertion and attaches any pending
   first-party OAuth authorization handoff.

Recovery is offline and single use:

1. `BeginAccountRecovery(username, recovery_code)` verifies the recovery code
   hash and creates a narrow recovery session.
2. `BeginRecoveryPasskeyRegistration(recovery_session_id)` starts replacement
   passkey enrollment.
3. `FinishRecoveryPasskeyRegistration(...)` stores the replacement passkey,
   rotates the recovery code, revokes prior web sessions, and returns the new
   recovery code once.

Authenticated device enrollment uses `BeginPasskeyRegistration` and
`FinishPasskeyRegistration` to add more passkeys to the same account.

## OAuth surfaces

### OAuth server (auth service)

Auth service acts as the authorization server for first-party clients.

Endpoints:

- `GET /authorize` + `POST /authorize/consent`
- `POST /token`
- `POST /introspect` (protected by `X-Resource-Secret`)
- `GET /.well-known/oauth-authorization-server`

Token model:

- Access tokens are opaque and persisted in auth storage.
- Protected resources (for example MCP HTTP transport) validate via `/introspect`.

### MCP protected resource

Endpoint:

- `GET /.well-known/oauth-protected-resource`

401 responses include `WWW-Authenticate: Bearer resource_metadata=...`.

## Operational invariants

- No email, phone, password, or external social-login provider participates in
  authentication or recovery.
- Public auth pages treat users as authenticated only after passkey or
  recovery-session completion plus web-session validation.
- Username ownership and uniqueness are enforced in `auth`, not `social`.
- Username availability checks are advisory reads from `auth`; successful signup
  remains the only authoritative reservation path.
- A linked user-facing identity always has an auth username; downstream
  services should fall back to that username before rendering anonymous
  placeholders.
- Worker-driven sync may project auth-owned usernames into `social` read models,
  but those projections are derivative and rebuildable.
- Recovery codes are stored only as hashes and are rotated after successful
  recovery.
- If all passkeys and the recovery code are lost, the account is unrecoverable
  by design.
- Protected resource token checks fail closed when introspection is unavailable.
- Identity and OAuth docs should not duplicate environment default inventories.

## Configuration

Authoritative environment defaults and wiring values live in:
[Running configuration](../../running/configuration.md).

Use this page for boundaries and behavior semantics; keep variable inventories in
running docs.
