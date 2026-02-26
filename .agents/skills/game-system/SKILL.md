---
name: game-system
description: Workflow and checks for adding a new game system
user-invocable: true
---

# Adding a New Game System

Use `docs/project/game-systems.md` for the comprehensive guide, checklist, and
Daggerheart reference implementation.

## Quick-start workflow

1. **Start at manifest.go** — open
   `internal/services/game/domain/bridge/manifest/manifest.go` first. It shows
   all three registry surfaces (`BuildModule`, `BuildMetadataSystem`,
   `BuildAdapter`) wired from one `SystemDescriptor`. Read an existing entry
   (Daggerheart) to understand the shape, then add yours.

2. **Pick the right DecideFunc helper** — see the decision tree in
   `docs/project/game-systems.md` under "DecideFunc decision tree":
   - `DecideFunc[P]` — no state needed, same payload for command/event
   - `DecideFuncWithState[S, P]` — needs state for validation
   - `DecideFuncTransform[S, PIn, POut]` — event payload differs from command
   - Raw `Decide` switch — multi-event or custom routing

3. **Follow the numbered steps** in `docs/project/game-systems.md` § "Adding a
   new system (current flow)" (steps 1–10).

4. **Run conformance tests** after wiring:
   ```bash
   go test ./internal/services/game/domain/module/testkit/...
   ```
   `testkit.ValidateSystemConformance` composes all startup validators for
   module + adapter pairs and fails loudly on missing registrations.

5. **Run full verification**:
   ```bash
   make test         # unit tests
   make integration  # gRPC + MCP + storage + event-catalog-check
   ```

## Startup validator troubleshooting

| Validator | What it checks | Fix |
|---|---|---|
| `ValidateDeciderCommandCoverage` | Every registered system command has a decider case | Add missing command type to decider's `DeciderHandledCommands()` and `Decide()` |
| `ValidateSystemFoldCoverage` | Every emittable event with replay intent has a fold handler | Register a `HandleFold` for the missing event type |
| `ValidateAdapterEventCoverage` | Every emittable event with projection intent has an adapter handler | Register a `HandleAdapter` for the missing event type |
| `ValidateCommandTypeNamespace` | System command types match `sys.<system_id>.*` | Fix the command type constant string |
| `ValidateEventTypeNamespace` | System event types match `sys.<system_id>.*` | Fix the event type constant string |

## Key reference files

| Concern | Location |
|---|---|
| Manifest registry | `domain/bridge/manifest/manifest.go` |
| Module interface | `domain/module/registry.go` |
| Adapter interface | `domain/bridge/adapter_registry.go` |
| ProfileAdapter | `domain/bridge/adapter_registry.go` |
| DecideFunc helpers | `domain/module/decide_func.go` |
| FoldRouter | `domain/module/fold_router.go` |
| Conformance tests | `domain/module/testkit/` |
| Daggerheart example | `domain/bridge/daggerheart/` |
