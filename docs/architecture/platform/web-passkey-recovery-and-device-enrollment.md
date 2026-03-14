---
title: "Web passkey recovery and device enrollment"
parent: "Platform surfaces"
nav_order: 10
status: proposed
owner: engineering
last_reviewed: "2026-03-08"
---

# Web passkey recovery and device enrollment

Canonical next-wave UX spec for web account recovery and authenticated
passkey enrollment. The auth RPC surface already exists; this document defines
the intended web flow and boundaries before implementation.

## Status

Implementation pending. This document is design authority for the next web
wave; it does not imply that the routes or handlers exist today.

## Scope

- recovery-code-driven account restoration in web
- authenticated add-passkey flow in web settings
- success, failure, and state-transition rules for those UX paths

Out of scope:

- auth backend contract changes
- passkey deletion UI
- proactive recovery-code rotation outside successful recovery
- account identity mutation (username remains immutable)

## Recovery UX

### Entry point

- Public route from the auth/login surface.
- User supplies `username` and `recovery_code`.
- Web calls `BeginAccountRecovery`.

### Recovery session rules

- A successful `BeginAccountRecovery` returns `recovery_session_id`.
- Web must treat that session as narrow-scope state that can only continue
  replacement passkey enrollment.
- The recovery flow must not create a normal logged-in app session until
  recovery passkey registration finishes successfully.

### Replacement passkey enrollment

1. Web calls `BeginRecoveryPasskeyRegistration(recovery_session_id)`.
2. Browser completes the WebAuthn creation ceremony.
3. Web submits `FinishRecoveryPasskeyRegistration(recovery_session_id, session_id, credential_response_json, pending_id?)`.
4. On success, web stores the returned web session, shows the replacement
   recovery code exactly once, and redirects into the authenticated app.

### Recovery failure handling

- Invalid username/recovery-code combinations render a generic failure state;
  do not reveal whether the username exists.
- Expired or consumed recovery sessions restart from the recovery start page.
- WebAuthn ceremony failures stay on the replacement-passkey step and allow
  retry while the recovery session remains valid.

### Recovery success messaging

- Explain that prior web sessions were revoked and prior passkeys were replaced.
- Require an explicit acknowledgement/download affordance for the new recovery
  code before leaving the success state.

## Authenticated add-passkey UX

### Entry point

- Settings security area for an already authenticated user.
- Web may optionally list existing passkeys using `ListPasskeys`, but the core
  enrollment flow must not depend on edit/delete controls.

### Enrollment flow

1. Web calls `BeginPasskeyRegistration(user_id)`.
2. Browser completes the WebAuthn creation ceremony.
3. Web calls `FinishPasskeyRegistration(session_id, credential_response_json)`.
4. UI returns to settings with a success notice and refreshed passkey list.

### Failure handling

- Ceremony cancellation or client-side WebAuthn errors stay on the settings
  page and surface a retryable inline error.
- Backend errors must fail closed and not imply partial enrollment.

## Security and UX invariants

- Web never handles private key material; it only transports WebAuthn JSON.
- Recovery flow must not fall back to email, SMS, or support-mediated identity
  proofing.
- Username is display-only throughout these flows.
- Public profile routing is independent from whether optional social metadata
  exists.
- Pending first-party OAuth authorizations may be completed after successful
  passkey login or successful recovery, but the web flow must not invent its
  own token/session semantics.
