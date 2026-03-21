---
title: "Game service contributor map"
parent: "Reference"
nav_order: 15
status: canonical
owner: engineering
last_reviewed: "2026-03-12"
---

# Game service contributor map

Reader-first routing guide for contributors changing the game service.

## Start here

Read in this order:

1. [Architecture foundations](../architecture/foundations/index.md)
2. [Event-driven system](../architecture/foundations/event-driven-system.md)
3. [Game systems architecture](../architecture/systems/game-systems.md)
4. This page

Use [Verification commands](../running/verification.md) for the canonical local
check sequence.

## Where to edit

| Change you want | Primary packages/files |
| --- | --- |
| Change startup wiring, runtime registration, or service exposure | `internal/services/game/app/` |
| Change gRPC interceptors or shared transport guards | `internal/services/game/api/grpc/interceptors/`, `internal/services/game/api/grpc/internal/` |
| Change core game transport for campaigns, participants, characters, sessions, scenes, invites, snapshots, forks, events, interaction, or authorization | `internal/services/game/api/grpc/game/` |
| Change core transport protobuf mapping for one capability | `internal/services/game/api/grpc/game/<capability>transport/` |
| Change Daggerheart deterministic read/mechanics endpoints | `internal/services/game/api/grpc/systems/daggerheart/mechanicstransport/` |
| Change Daggerheart gameplay mutation/read transport | `internal/services/game/api/grpc/systems/daggerheart/*transport/` and thin root wrappers in `internal/services/game/api/grpc/systems/daggerheart/` |
| Change Daggerheart content/catalog or asset endpoints | `internal/services/game/api/grpc/systems/daggerheart/contenttransport/`, `internal/services/game/api/grpc/systems/daggerheart/content_service.go`, `internal/services/game/api/grpc/systems/daggerheart/asset_service.go` |
| Change Daggerheart character-creation workflow provider | `internal/services/game/api/grpc/systems/daggerheart/creationworkflow/` |
| Change command dispatch, replay, or cross-aggregate engine policy | `internal/services/game/domain/command/`, `internal/services/game/domain/engine/`, `internal/services/game/domain/readiness/` |
| Change aggregate rules for campaigns, sessions, characters, participants, invites, scenes, forks, or readiness | `internal/services/game/domain/<aggregate>/` plus the owning workflow-local `decider_*.go` or workflow package listed below |
| Change system-owned domain behavior for Daggerheart | `internal/services/game/domain/systems/daggerheart/` |
| Change manifest/module/adapter/metadata registration for a game system | `internal/services/game/domain/systems/manifest/`, `internal/services/game/domain/module/`, `internal/services/game/domain/systems/` |
| Change projection apply logic or read-model contracts | `internal/services/game/projection/`, `internal/services/game/storage/` |
| Change SQLite persistence backends | `internal/services/game/storage/sqlite/` and the owning backend subpackage such as `coreprojection/`, `eventjournal/`, `integrationoutbox/`, `daggerheartcontent/`, or `daggerheartprojection/` |
| Change integration harnesses or cross-service behavior checks | `internal/test/integration/`, `internal/services/game/integration/` |

## Package reading order

Use this order when orienting yourself in the game service:

1. `internal/services/game/app/`
   Why: startup shows what actually gets wired and exposed.
2. `internal/services/game/api/grpc/game/` or `internal/services/game/api/grpc/systems/daggerheart/`
   Why: transport packages show the capability boundary and the application seam.
3. `internal/services/game/domain/...`
   Why: domain packages own invariants, command decisions, and replay state.
4. `internal/services/game/projection/` plus `internal/services/game/storage/...`
   Why: read models and persistence explain how emitted events become query state.

## Aggregate workflow routing

For aggregate changes, start with the owning workflow file rather than scanning
the whole package.

| Aggregate/policy | Start here | Then read |
| --- | --- | --- |
| `campaign` | `decider_create.go`, `decider_update.go`, `decider_lifecycle.go`, `decider_ai.go`, `decider_fork.go` | `decider.go`, `policy.go`, `registry.go`, `fold.go` |
| `campaignbootstrap` | `workflow.go` | `workflow_test.go`, then `campaign/` and `participant/` as needed |
| `participant` | `decider_join.go`, `decider_update.go`, `decider_lifecycle.go`, `decider_binding.go` | `decider.go`, `decider_shared.go`, `registry.go`, `fold.go` |
| `session` | `decider_lifecycle.go`, `decider_gate.go`, `decider_spotlight.go`, `decider_interaction.go` | `decider.go`, the `gate_workflow_*.go`, `gate_progress_*.go`, and `gate_projection_*.go` families, then `registry.go` and `fold.go` |
| `scene` | `decider_lifecycle.go`, `decider_character.go`, `decider_gate.go`, `decider_spotlight.go` | `decider.go`, the `registry_*.go` family, then `fold.go` |
| `character` | `decider_create.go`, `decider_update.go`, `decider_lifecycle.go` | `decider.go`, `decider_shared.go`, `registry.go`, `fold.go` |
| `invite` | `decider_create.go`, `decider_claim.go`, `decider_lifecycle.go` | `decider.go`, `registry.go`, `fold.go`, `decline_test.go` for the dedicated decline path |
| `action` | `decider_roll.go`, `decider_outcome.go`, `decider_note.go` | `decider.go`, `decider_shared.go`, `registry.go`, `fold.go` |
| `readiness` | `session_start_workflow.go`, `session_start_boundary.go`, `session_start_blockers.go`, `session_start_actions.go` | `session_start.go`, `session_start_indexes.go`, `action.go` |
| `fork` | `fork.go` | integration callers in `api/grpc/game/fork_*` when behavior crosses the transport seam |
| `replay` | `replay.go` | engine callers when changing replay semantics or event filtering |

Use `registry.go` when changing payload contracts or command/event metadata.
Use `fold.go` when changing replay state instead of write-path validation.

## Game transport routing

### Core game transport

Use `internal/services/game/api/grpc/game/` when the change is system-agnostic.

- Campaigns: `campaign_*application.go`, `campaign_service_*.go`, `campaigntransport/`
- Participants: `participant_*application.go`, `participant_service_*.go`, `participanttransport/`
- Characters: `character_*application.go`, `character_service_*.go`, `charactertransport/`, `characterworkflow/`
- Sessions/scenes/interaction: `session_*`, `scene_*`, `interaction_*`, `sessiontransport/`
- Invites/forks/events/snapshots/authorization: matching `*_application.go` and `*_service*.go` groups

### Daggerheart transport

Use `internal/services/game/api/grpc/systems/daggerheart/` only for thin service
roots, constructor seams, and system-wide helpers that genuinely remain shared.
Most feature work should go straight to the sibling package that owns the
behavior:

- `contenttransport/`: catalog/content/asset endpoint behavior
- `mechanicstransport/`: deterministic dice and rules metadata
- `sessionflowtransport/`: high-level multi-step session flows
- `sessionrolltransport/`: roll execution and low-level roll mutation paths
- `outcometransport/`: roll outcome application
- `damagetransport/`: damage mutation paths
- `conditiontransport/`: conditions and life-state changes
- `gmmovetransport/`: GM fear and GM move writes
- `countdowntransport/`: countdown lifecycle writes
- `charactermutationtransport/`: progression, inventory, and other character-scoped writes
- `recoverytransport/`: rest, downtime, temporary armor, death move, blaze
- `adversarytransport/`: adversary CRUD/read paths
- `workflowtransport/`, `workflowruntime/`, `workfloweffects/`: shared gameplay workflow seams
- `creationworkflow/`: character-creation workflow provider

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Domain invariants, deciders, replay folds, or registry metadata | `internal/services/game/domain/<area>/*_test.go` next to the owning workflow file | Protect durable business rules at the owning package seam without routing back through package-wide hubs. |
| Core game gRPC handler orchestration | `internal/services/game/api/grpc/game/*_test.go` or the owning `<capability>transport/` package | Keep transport assertions near the application seam and protobuf mapping boundary. |
| Daggerheart transport behavior | The owning Daggerheart sibling package such as `*transport/*_test.go`, `workflowruntime/*_test.go`, or `creationworkflow/*_test.go` | New transport splits are intended to carry their own seam coverage instead of root-package regression tests. |
| Projection apply behavior or event dispatch ordering | `internal/services/game/projection/*_test.go` plus integration coverage when ordering matters | Projection rules are durability-sensitive and should be tested where apply happens. |
| Shared storage contracts | `internal/services/game/storage/*_test.go` | Contract packages should validate store expectations independent of one backend. |
| SQLite persistence behavior | `internal/services/game/storage/sqlite/.../*_test.go` in the owning backend package | Backend-specific SQL, migrations, and persistence invariants belong with that backend. In `coreprojection/`, start with the matching `store_projection_*.go`, `store_conversion_*.go`, or runtime helper file and keep the test near that concern. |
| Startup wiring or service registration | `internal/services/game/app/*_test.go` | Composition-root tests should stay in the composition root. |
| Cross-package workflows or system registration parity | `internal/test/integration/` and targeted package tests | Use integration tests only when the contract spans domain, transport, and persistence together. |
| Architecture guardrails on handler boundaries | existing `write_path_arch_test.go` / package guardrail tests in the owning transport package | Keep anti-regression checks at the package boundary they protect. |

## Verification

Baseline contributor workflow for game-service changes:

- `make test`
- `make smoke`
- `make check`

Add these when applicable:

- `make game-architecture-check` for `internal/services/game/**` boundary changes
- `make cover` when production behavior changes
- `make cover-critical-domain` when changing `internal/services/game/domain/**`
- `make docs-check` for docs-heavy batches

## Related docs

- [Contributor map](contributor-map.md)
- [Game systems architecture](../architecture/systems/game-systems.md)
- [Testing policy](../architecture/policy/testing-policy.md)
- [Verification commands](../running/verification.md)
