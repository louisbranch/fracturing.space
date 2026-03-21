// Package daggerheart provides the root Daggerheart gRPC service boundary.
//
// The root package owns only:
//   - the top-level gRPC service type and constructor state,
//   - thin wrappers that preserve the public transport contract,
//   - content and asset service roots,
//   - system-wide dependency assembly that does not belong to one sibling
//     transport package.
//
// Most feature work should start in a sibling package instead of this root:
//   - `contenttransport/`: catalog, content, assets, pagination, localization
//   - `mechanicstransport/`: deterministic dice, explanation, rules metadata
//   - `sessionflowtransport/`: high-level multi-step session gameplay flows
//   - `sessionrolltransport/`: low-level roll execution and roll-side writes
//   - `outcometransport/`: roll outcome application and repair
//   - `damagetransport/`: damage application
//   - `conditiontransport/`: conditions and life-state changes
//   - `gmmovetransport/`: GM fear and GM move writes
//   - `countdowntransport/`: countdown lifecycle writes
//   - `charactermutationtransport/`: inventory, progression, character writes
//   - `recoverytransport/`: rest, downtime, temporary armor, death move, blaze
//   - `adversarytransport/`: adversary CRUD and session-scoped adversary reads
//   - `creationworkflow/`: character-creation workflow provider
//   - `gameplaystores/`: gameplay dependency bundle
//   - `workflowtransport/`, `workflowruntime/`, `workflowwrite/`,
//     `workfloweffects/`: shared gameplay workflow support
//
// Reading order for contributors:
//  1. `service.go` for the root service shape,
//  2. the relevant `workflow_*handler.go`, `content_service.go`, or
//     `asset_service.go` wrapper,
//  3. the sibling package that owns the actual behavior,
//  4. `internal/services/game/domain/systems/daggerheart/...` when the change
//     affects system commands, events, or replay rules.
//
// Non-goals:
//   - root-package ownership of feature transport logic; most handlers and
//     protobuf shaping now live in sibling packages,
//   - shared transport helpers via `api/grpc/game`; cross-system helpers must
//     stay in common internal packages instead.
//
// This package implements `systems.daggerheart.v1.DaggerheartService` from
// `api/proto/systems/daggerheart/v1/service.proto`.
package daggerheart
