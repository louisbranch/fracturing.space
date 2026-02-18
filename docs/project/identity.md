---
title: "Identity and Recovery"
parent: "Project"
nav_order: 9
---

# Identity and Recovery

This document captures the identity model and recovery flows for authentication.

## Identity Model

- **User**: Core identity record. A user is defined by an email.
- **Email**: Canonical identity value used across auth, admin, and game surfaces.
- **Additional emails**: Planned as a future extension, but out of scope for this change.
- **Passkey**: Primary authentication credential. Users can register multiple passkeys.

## Recovery Model

- **Magic link**: Single-use token issued to an email address for recovery or login.
- Magic links are stored with an expiration time and a used-at timestamp.
- Consuming a magic link verifies the email and can attach to a pending OAuth authorization.

## Service Boundaries

- **Auth service**: Source of truth for users, emails, passkeys, magic links, and OAuth issuance.
- **Web service**: Hosts login and recovery UX, calls auth gRPC for verification and storage.
- **Admin service**: Generates magic links for operators (display-only, not emailed).

## UX Flow (Web)

1) Begin passkey login (or registration) via web endpoints.
2) WebAuthn ceremony runs in the browser.
3) Auth service verifies responses and persists credentials.
4) OAuth authorization proceeds via pending authorization + consent.

## Notes

- Email is the canonical identity during this release.
- Additional email support is planned but out of scope.
