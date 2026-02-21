---
title: "Web Session Persistence Across Restarts"
nav_order: 80
---

# Web Session Persistence Across Restarts

This page captures the web session restart behavior introduced for session continuity.

## Decision

- Web sessions are now persisted in `web_sessions` in the web cache SQLite DB.
- Persistence stores only a SHA-256 hash of the raw access token (`access_token_hash`), not the raw token.
- In-memory sessions remain authoritative while the process is running.
- On restart, session lookup follows `fs_session` + `fs_token` cookies:
  - `fs_session` resolves the session ID.
  - `fs_token` is hashed and validated against the persisted hash before acceptance.
  - A missing or mismatched token hash causes the persisted row to be removed.

## Operational Notes

- Expired rows are pruned during `SaveSession`.
- Session save/update uses upsert and preserves original `created_at`.
- Debug logging is added when persistence save/load/delete fails to make restart recovery issues easier to diagnose.

## Side Effects

- Existing sessions are not invalidated in memory while the process is live.
- Sessions restored from persistence require both cookies to pass validation.
