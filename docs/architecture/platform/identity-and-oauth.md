---
title: "Identity and OAuth"
parent: "Platform surfaces"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Identity and OAuth

Canonical identity, passkey, recovery, and OAuth architecture for Fracturing.Space.

This page defines service ownership, security boundaries, and invariants for
account identity, offline recovery, and first-party OAuth.

## Ownership boundaries

- **Auth service** owns identity and access primitives: users, usernames,
  passkeys, recovery-code state, web sessions, and OAuth issuance/introspection.
- **Web service** hosts signup, login, recovery, and settings UX and delegates
  credential verification and storage to auth.
- **Social service** owns optional profile metadata, contacts, and authenticated
  people-search read models. It does not own auth or usernames.
- **Discovery service** remains the public browsing surface and consumes
  published public data rather than owning usernames.

Boundary rules:

1. If a field proves identity or grants access, it belongs to `auth`.
2. If a field is profile metadata, contact ranking state, or people-search
   projection data, it belongs to `social`.
3. Account preferences such as locale belong to `auth`.
4. Discovery surfaces consume published public data; they do not own usernames.

## Identity model

- **User**: canonical identity record keyed by user ID.
- **Username**: immutable auth-owned account locator and public handle.
- **Passkeys**: primary authentication credential; multiple may be registered.
- **User locale**: private account preference on the user record.
- **Public profile**: baseline profile exists as soon as the account exists.
- **Authenticated people search**: social-owned read model keyed by auth-owned
  usernames and enriched for invite and mention UX.

## Passkey and recovery model

Signup and login are username-first WebAuthn ceremonies:

1. `BeginAccountRegistration(username, locale)` reserves the username and
   returns WebAuthn creation options for a short signup-only ceremony window.
2. `FinishAccountRegistration(session_id, credential_response)` verifies the
   first passkey, stages the recovery-code hash plus credential data in the
   registration session, extends that staged signup to the pending-signup TTL,
   and returns the recovery code once. It does not create an active user or
   web session.
3. `AcknowledgeAccountRegistration(session_id, pending_id)` activates the
   account only after the browser confirms the recovery code was saved. This
   step creates the user, stores the first passkey, creates the web session,
   emits `auth.signup_completed`, and attaches any pending OAuth handoff.
4. `BeginPasskeyLogin(username)` returns assertion options for the account’s
   registered passkeys.
5. `FinishPasskeyLogin(...)` verifies the assertion and attaches any pending
   first-party OAuth authorization handoff.

Signup reservations therefore have two TTL phases: an initial short signup-only
WebAuthn reservation before passkey creation, then a longer pending-signup
reservation after passkey success while the user acknowledges the recovery code.
If either expires before activation, the username becomes available for a fresh
signup attempt and the abandoned signup becomes unusable.

Recovery is offline and single use:

1. `BeginAccountRecovery(username, recovery_code)` verifies the recovery code
   hash and creates a narrow recovery session.
2. `BeginRecoveryPasskeyRegistration(recovery_session_id)` starts replacement
   passkey enrollment.
3. `FinishRecoveryPasskeyRegistration(...)` stores the replacement passkey,
   rotates the recovery code, revokes prior web sessions, and returns the new
   recovery code once.

Authenticated device enrollment uses `BeginPasskeyRegistration` and
`FinishPasskeyRegistration` to add more passkeys.

## OAuth surfaces

### OAuth server (auth service)

Auth service acts as the authorization server for first-party clients.

- `GET /authorize` + `POST /authorize/consent`
- `POST /token`
- `POST /introspect` (protected by `X-Resource-Secret`)
- `GET /.well-known/oauth-authorization-server`

Access tokens are opaque and persisted in auth storage. Protected resources,
including first-party web/admin surfaces, validate them through `/introspect`.

## Operational invariants

- No email, phone, password, or external social-login provider participates in
  authentication or recovery.
- Public auth pages treat users as authenticated only after passkey or recovery
  completion plus web-session validation.
- Username ownership and uniqueness are enforced in `auth`, not `social`.
- Username availability checks are advisory reads; successful signup remains the
  authoritative reservation path.
- A staged signup does not count as an active user. Until recovery-code
  acknowledgement succeeds, the username is only temporarily reserved.
- A linked user-facing identity always has an auth username; downstream
  services should fall back to it before rendering anonymous placeholders.
- Worker-driven sync may project auth-owned usernames into `social`, but those
  projections are derivative and rebuildable.
- Recovery codes are stored only as hashes and rotate after successful recovery.
- `auth.signup_completed` is emitted only when staged signup acknowledgement
  activates the user.
- If all passkeys and the recovery code are lost, the account is unrecoverable
  by design.
- Protected resource token checks fail closed when introspection is unavailable.

## Configuration

Authoritative defaults and wiring values live in
[Running configuration](../../running/configuration.md). Keep this page focused
on boundaries and behavior semantics, not variable inventories.
