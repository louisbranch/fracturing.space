---
title: "Web passkey recovery and device enrollment"
parent: "Platform surfaces"
nav_order: 10
status: implemented
owner: engineering
last_reviewed: "2026-03-11"
---

# Web passkey recovery and device enrollment

Canonical UX and transport reference for staged web signup, account recovery,
and authenticated passkey enrollment. Auth remains the backend authority; the
web tier composes it into browser flows.

## Scope

- staged public signup activation
- recovery-code-driven account restoration
- authenticated add-passkey flow in settings
- success, failure, and state-transition rules for those UX paths

## Signup activation UX

`BeginAccountRegistration` creates a short-lived username reservation for the
active WebAuthn ceremony. That reservation uses a shorter signup-specific TTL
than the general WebAuthn session TTL used by login, recovery, and authenticated
passkey enrollment. If the browser never completes passkey creation, the user
can retry signup with the same username after that short reservation expires.

`FinishAccountRegistration` does not sign the user in. It stages the first
credential plus the recovery-code hash and extends the registration session to
the pending-signup TTL used only for the recovery-code confirmation step.

Signup completion flow:

1. Browser posts `/passkeys/register/start`.
2. Web returns `session_id` and WebAuthn creation options.
3. Browser completes the WebAuthn creation ceremony.
4. Browser posts `/passkeys/register/finish` with `session_id`, credential
   JSON, optional `pending_id`, and optional `next`.
5. Web stores one-time recovery reveal state in an `HttpOnly`, path-scoped
   cookie and redirects to `/login/recovery-code`.
6. `/login/recovery-code` renders signup-specific copy and keeps the reveal
   cookie available across refreshes in the same browser.
7. Continuing posts `/login/recovery-code/acknowledge`.
8. Web calls `AcknowledgeAccountRegistration`, then writes the normal web
   session cookie and redirects to the resolved post-auth destination.

Signup failure handling:

- The username stays unavailable only until the staged registration expires or
  is acknowledged.
- If signup crashes, the passkey provider fails, or the browser closes before
  acknowledgement, the user waits for the staged-signup TTL to expire and then
  signs up again with the same username.
- If acknowledgement is attempted after expiry or after a newer signup has
  superseded the reservation, web clears the reveal cookie, redirects to
  `/login`, and shows a retry message.
- Web must not write the normal authenticated session cookie before signup
  acknowledgement succeeds.

## Recovery UX

- Public route `/login/recovery`, linked from `/login`.
- User supplies `username` and `recovery_code`.
- Web calls `BeginAccountRecovery`, then immediately starts replacement
  passkey enrollment via `BeginRecoveryPasskeyRegistration`.
- A successful `BeginAccountRecovery` returns `recovery_session_id`, which web
  must treat as narrow-scope state that can only continue replacement passkey
  enrollment.
- The recovery flow must not create a normal logged-in app session until
  recovery passkey registration finishes successfully.

Replacement passkey enrollment:

1. Browser posts `/passkeys/recovery/start`.
2. Web returns `recovery_session_id`, passkey `session_id`, and WebAuthn
   creation options in one response.
3. Browser completes the WebAuthn creation ceremony.
4. Browser posts `/passkeys/recovery/finish` with `recovery_session_id`,
   `session_id`, credential JSON, and optional `pending_id`.
5. On success, web stores the returned web session, writes one-time
   recovery-code reveal state, and redirects to `/login/recovery-code`.

Recovery failure and success rules:

- Invalid username/recovery-code combinations render a generic failure state;
  do not reveal whether the username exists.
- Expired or consumed recovery sessions restart from `/login/recovery`.
- WebAuthn ceremony failures stay on the replacement-passkey step and allow
  retry while the recovery session remains valid.
- Explain that prior web sessions were revoked and prior passkeys were replaced.
- Require explicit acknowledgement or download of the new recovery code before
  leaving the success state.
- Never place the recovery code in URLs, flash notices, or persistent browser
  storage.

## Shared recovery-code reveal

- Successful signup and successful account recovery both redirect to
  `/login/recovery-code`.
- Reveal state lives in a dedicated `HttpOnly`, path-scoped cookie.
- Signup mode preserves the reveal cookie across GET refreshes until
  acknowledgement or expiry so a same-browser refresh does not strand the user.
- Recovery mode also keeps the reveal cookie until the acknowledgement POST.
- Continuing posts to `/login/recovery-code/acknowledge`.
- With a first-party OAuth `pending_id`, acknowledgement redirects to
  `${FRACTURING_SPACE_WEB_AUTH_BASE_URL}/authorize/consent?pending_id=...`.
- Without a `pending_id`, acknowledgement redirects to `/app/dashboard`.

## Authenticated add-passkey UX

- Authenticated route `/app/settings/security`.
- Web may optionally list existing passkeys using `ListPasskeys`, but the core
  enrollment flow must not depend on edit/delete controls.

Enrollment flow:

1. Browser posts `/app/settings/security/passkeys/start`.
2. Web calls `BeginPasskeyRegistration(user_id)` and returns WebAuthn creation
   options.
3. Browser completes the WebAuthn creation ceremony.
4. Browser posts `/app/settings/security/passkeys/finish`.
5. Web calls `FinishPasskeyRegistration(session_id, credential_response_json)`.
6. UI redirects back to `/app/settings/security` with a success notice and a
   refreshed passkey list.

Failure and presentation rules:

- Ceremony cancellation or client-side WebAuthn errors stay on the settings
  page and surface a retryable inline error.
- Backend errors must fail closed and not imply partial enrollment.
- Settings passkey mutation endpoints require authenticated user context and
  same-origin proof.
- The security page lists passkeys read-only, sorted by `last_used_at`
  descending then `created_at` descending.
- UI labels rows as `Passkey 1`, `Passkey 2`, and so on.
- Raw credential IDs are never rendered in the browser.

## Security and UX invariants

- Web never handles private key material; it only transports WebAuthn JSON.
- Recovery flow must not fall back to email, SMS, or support-mediated proofing.
- Username is display-only throughout these flows.
- Public profile routing is independent from whether optional social metadata
  exists.
- Pending first-party OAuth authorizations may complete after successful
  passkey login or recovery, but the web flow must not invent its own
  token/session semantics.
