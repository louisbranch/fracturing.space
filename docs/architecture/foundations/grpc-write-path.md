---
title: "gRPC write path"
parent: "Foundations"
nav_order: 6
status: canonical
owner: engineering
last_reviewed: "2026-03-04"
---

# gRPC Write Path

How gRPC handlers execute domain commands, with error handling boundaries and helper conventions.

Prerequisite: [Event-driven system](event-driven-system.md) for the core lifecycle.

## Handler to domain — execution flow

```
gRPC handler
  │
  ├─ build command (commandbuild.Core / commandbuild.System)
  ├─ choose Options (empty or preset)
  │
  └─ executeAndApplyDomainCommand(ctx, stores, applier, cmd, options)
       │
       ├─ normalizeGRPCDefaults(&options)     ← inject gRPC-aware error handlers
       │
       └─ WriteRuntime.ExecuteAndApply(ctx, domain, applier, cmd, options)
            │
            ├─ domain.Execute(cmd)            ← engine: validate → gate → load → decide → append → fold
            ├─ intent filter (ShouldApply)    ← skip audit-only/replay-only events
            └─ applier.Apply(event)           ← inline projection (if enabled)
```

## Two execution helpers

| Helper | Inline projection | Use when |
|---|---|---|
| `executeAndApplyDomainCommand` | Yes | Default. Handler needs read-after-write consistency |
| `executeDomainCommandWithoutInlineApply` | No | Outbox pattern or fire-and-forget writes |

Both call `normalizeGRPCDefaults` and `ensureGRPCStatus` identically. The only difference is whether events are applied to projections inline.

## Error handling boundaries

The design keeps domain logic transport-agnostic. Error mapping happens at two boundaries:

### 1. `normalizeGRPCDefaults` — injected error handlers

Sets three error handlers on `Options` if the caller didn't provide custom ones:

| Handler | Wraps | Default gRPC code |
|---|---|---|
| `ExecuteErr` | Engine execution failures | `codes.Internal` |
| `ApplyErr` | Projection apply failures | `codes.Internal` |
| `RejectErr` | Domain rejections (business rule violations) | `codes.FailedPrecondition` |

These fire inside `WriteRuntime.ExecuteAndApply` at the appropriate points.

### 2. `ensureGRPCStatus` — final error wrapper

Catches any error that escapes without a gRPC status:

1. Already a gRPC status → pass through.
2. Domain error (`apperrors.GetCode != CodeUnknown`) → `handleDomainError` maps to semantic gRPC code (NotFound, InvalidArgument, FailedPrecondition, etc.).
3. Unknown error → `codes.Internal`.

### 3. `handleDomainError` — domain code mapping

Delegates to `apperrors.HandleError(err, apperrors.DefaultLocale)`, which maps domain error codes to gRPC codes with i18n-ready structured error details.

Both the game service and daggerheart packages use the same pattern.

## Options type

```go
type Options struct {
    RequireEvents      bool              // Reject if no events emitted
    MissingEventMsg    string            // Error message when RequireEvents fails
    ExecuteErr         func(error) error // Custom executor error wrapper
    ApplyErr           func(error) error // Custom applier error wrapper
    RejectErr          func(string) error // Custom rejection wrapper
    ExecuteErrMessage  string            // Fallback message for ExecuteErr
    ApplyErrMessage    string            // Fallback message for ApplyErr
}
```

### Presets

- `domainwrite.RequireEvents(msg)` — command must emit at least one event.
- `domainwrite.RequireEventsWithDiagnostics(msg, applyMsg)` — same, with custom diagnostic messages.
- `domainwrite.Options{}` (empty) — zero events allowed, default messages.

## Intent filtering

`WriteRuntime` holds an intent filter built from the event registry. During inline apply, each event is checked:

- `IntentProjectionAndReplay` → applied to projections.
- `IntentReplayOnly` → skipped (affects aggregate state only).
- `IntentAuditOnly` → skipped (journal-only).

This ensures projection appliers only process events they are responsible for.

## Typical handler pattern

```go
func (a *application) DoSomething(ctx context.Context, in *pb.Request) (*pb.Response, error) {
    if in == nil {
        return nil, status.Error(codes.InvalidArgument, "request is required")
    }
    campaignID := in.GetCampaignId()
    if campaignID == "" {
        return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
    }

    // Build domain command
    cmd := commandbuild.Core(commandbuild.CoreInput{
        CampaignID:  campaignID,
        Type:        commandType,
        PayloadJSON: payloadJSON,
        // ...
    })

    // Execute and apply
    _, err := executeAndApplyDomainCommand(ctx, a.stores, applier, cmd, domainwrite.Options{})
    if err != nil {
        return nil, err // already gRPC-wrapped
    }

    return &pb.Response{}, nil
}
```

Key conventions:
- Validate request fields before building commands (return `codes.InvalidArgument`).
- The returned error from `executeAndApplyDomainCommand` is always gRPC-status-wrapped — don't double-wrap.
- Use `handleDomainError` for errors from store lookups or other domain operations outside the command path.

## Adding a new write handler

1. Define your command type and event types in the domain layer.
2. Write a handler following the pattern above.
3. Choose `executeAndApplyDomainCommand` (default) or `executeDomainCommandWithoutInlineApply`.
4. Use `domainwrite.Options{}` or a preset — custom error handlers are rarely needed.
5. Errors from domain operations flow through `handleDomainError`; errors from the command path flow through the options handlers and `ensureGRPCStatus`.
