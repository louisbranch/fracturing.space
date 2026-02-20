# Daggerheart no-op event cleanup plan (pass 1)

## Goal
Complete the event-shape migration from removed no-op `*_resolved` and old `IntentAuditOnly`-style events to durable projection events and response-based assertions.

## Status
- [x] Confirm production handlers now no longer emit no-op resolution events.
- [x] Update gRPC daggerheart action tests to remove no-op command/event pairs and stale payload usage.
- [x] Update integration tests that query removed event types.
- [x] Update game runner tests for removed constants/helpers and damaged-roll lookup helper.
 - [x] Run targeted test/lint pass.

## Plan items
1. `internal/services/game/api/grpc/systems/daggerheart/actions_test.go`
   - Replace fake command->event expectations that use removed commands/types:
     - `sys.daggerheart.damage_roll.resolve` now emits `action.roll_resolved` only.
     - `sys.daggerheart.attack.resolve`, `reaction.resolve`, `adversary_roll.resolve`, `adversary_attack.resolve`, `group_action.resolve`, `tag_team.resolve` should be removed from test command maps.
     - `sys.daggerheart.gm_move.apply` should be removed; assert fear change via `sys.daggerheart.gm_fear_changed` + response deltas.
   - Remove `*_ResolvedPayload`/`GMMoveAppliedPayload` fixtures that are no longer emitted.
   - Keep assertions focused on mutable `action.outcome.apply` and domain state mutation events.

2. `internal/test/integration/adversary_attack_flow_test.go`
   - Replace `sys.daggerheart.adversary_attack_resolved` filter with a durable/observable assertion.
   - Use response roll sequence + damage events to assert outcome path executed.

3. `internal/test/integration/gm_mechanics_test.go`
   - Remove `sys.daggerheart.gm_move_applied` requirement and payload parsing helper.
   - Keep `sys.daggerheart.gm_fear_changed` check and response before/after values.

4. `internal/test/integration/group_action_tag_team_test.go`
   - Remove `group_action_resolved` / `tag_team_resolved` query helpers and payload structs.
   - Verify outcomes from flow responses only.

5. `internal/test/integration/reaction_flow_test.go`
   - Remove `reaction_resolved` query helper; keep flow response assertions.

6. `internal/test/game/runner_test.go`
   - Replace removed event constants with current durable events:
     - `EventTypeReactionResolved` -> `EventTypeOutcomeApplied`
     - `EventTypeAttackResolved` / `EventTypeAdversaryAttackResolved` -> `event.TypeOutcomeApplied` as minimum durable marker
     - `EventTypeGroupActionResolved` / `EventTypeTagTeamResolved` -> remove/replace as outcome is now applied via `action.outcome.apply` or response payload.
     - `EventTypeGMMoveApplied` -> remove helper-based checks
     - `EventTypeDamageRollResolved` -> resolve via `action.roll_resolved` + `roll_seq` payload.
   - Remove `findGMMoveAppliedPayload` and switch damage assertions to existing durable payload helper.

## Verification (targeted)
- `go test ./internal/services/game/api/grpc/systems/daggerheart -run TestApply|TestSession`
- `go test ./internal/test/integration -run TestDaggerheart` (or narrower tags as feasible)
- `go test ./internal/test/game -run Runner`
