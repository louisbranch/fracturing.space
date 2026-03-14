---
title: "Web passkey recovery and device enrollment"
parent: "Platform surfaces"
nav_order: 10
status: implemented
owner: engineering
last_reviewed: "2026-03-09"
---

# Web passkey recovery and device enrollment

Canonical UX and transport reference for web account recovery and authenticated
passkey enrollment. The auth RPC surface remains the backend authority; the
web tier composes it into dedicated browser flows.

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

- Public route `/login/recovery`, linked from `/login`.
- User supplies `username` and `recovery_code`.
- Web calls `BeginAccountRecovery`, then immediately starts replacement
  passkey enrollment via `BeginRecoveryPasskeyRegistration`.

### Recovery session rules

- A successful `BeginAccountRecovery` returns `recovery_session_id`.
- Web must treat that session as narrow-scope state that can only continue
  replacement passkey enrollment.
- The recovery flow must not create a normal logged-in app session until
  recovery passkey registration finishes successfully.

### Replacement passkey enrollment

1. Browser posts `/passkeys/recovery/start`.
2. Web returns `recovery_session_id`, passkey `session_id`, and WebAuthn
   creation options in one response.
3. Browser completes the WebAuthn creation ceremony.
4. Browser posts `/passkeys/recovery/finish` with
   `recovery_session_id`, `session_id`, credential JSON, and optional
   `pending_id`.
5. On success, web stores the returned web session, writes one-time
   recovery-code reveal state, and redirects to `/login/recovery-code`.

### Recovery failure handling

- Invalid username/recovery-code combinations render a generic failure state;
  do not reveal whether the username exists.
- Expired or consumed recovery sessions restart from `/login/recovery`.
- WebAuthn ceremony failures stay on the replacement-passkey step and allow
  retry while the recovery session remains valid.

### Recovery success messaging

- Explain that prior web sessions were revoked and prior passkeys were replaced.
- Require an explicit acknowledgement/download affordance for the new recovery
  code before leaving the success state.
- Persist reveal state only in a short-lived, `HttpOnly`, path-scoped cookie.
- Never place the recovery code in URLs, flash notices, or persistent browser
  storage.

## Shared recovery-code reveal

- Successful signup and successful account recovery both redirect to
  `/login/recovery-code`.
- The reveal page reads one-time state from a dedicated cookie, renders
  mode-specific copy, then consumes that cookie.
- Continuing from the reveal page posts to
  `/login/recovery-code/acknowledge`.
- When a first-party OAuth `pending_id` is present, acknowledgement redirects
  to `${FRACTURING_SPACE_WEB_AUTH_BASE_URL}/authorize/consent?pending_id=...`.
- Without a `pending_id`, acknowledgement redirects to `/app/dashboard`.

## Authenticated add-passkey UX

### Entry point

- Authenticated route `/app/settings/security`.
- Web may optionally list existing passkeys using `ListPasskeys`, but the core
  enrollment flow must not depend on edit/delete controls.

### Enrollment flow

1. Browser posts `/app/settings/security/passkeys/start`.
2. Web calls `BeginPasskeyRegistration(user_id)` and returns WebAuthn creation
   options.
3. Browser completes the WebAuthn creation ceremony.
4. Browser posts `/app/settings/security/passkeys/finish`.
5. Web calls `FinishPasskeyRegistration(session_id, credential_response_json)`.
6. UI redirects back to `/app/settings/security` with a success notice and a
   refreshed passkey list.

### Failure handling

- Ceremony cancellation or client-side WebAuthn errors stay on the settings
  page and surface a retryable inline error.
- Backend errors must fail closed and not imply partial enrollment.
- Settings passkey mutation endpoints require authenticated user context and
  same-origin proof.

## Passkey list presentation

- The security page lists passkeys read-only.
- Rows are sorted by `last_used_at` descending, then `created_at` descending.
- UI labels rows as `Passkey 1`, `Passkey 2`, and so on.
- Raw credential IDs are never rendered in the browser.

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
- Passkey login finish and recovery finish both preserve `pending_id` and
  resolve post-auth redirects from the configured auth base URL.
