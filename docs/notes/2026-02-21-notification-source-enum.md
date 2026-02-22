---
title: "2026-02 Notification Source Enum"
parent: "Notes"
nav_order: 2
---

# 2026-02 Notification Source Enum

## Status

Accepted and implemented.

## Context

Notification API `source` was a free-form string, which allowed drift and made
internal message classification inconsistent.

Internal messages (for example, onboarding welcome notifications) needed a
single catch-all source classification.

## Decisions

1. Introduce `NotificationSource` enum in notifications API.
2. Set `NOTIFICATION_SOURCE_SYSTEM = 1` as the internal catch-all source.
3. Update onboarding welcome notification creation to emit `SYSTEM`.
4. Keep domain/storage source as string for now, with gRPC boundary mapping to
   avoid a storage migration in this change.

## Consequences

- API callers now use typed enum values instead of raw strings.
- Existing persisted records remain compatible with current storage schema.
- Future source categories can be added as explicit enum values without
  re-introducing free-form string drift at the API boundary.
