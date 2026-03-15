// Package game exposes the stable system-agnostic gRPC surface for the game
// service.
//
// Ownership map:
//   - `*_service.go` files are thin gRPC entrypoints and constructor seams.
//   - `*_application.go` files own use-case orchestration and dependency wiring.
//   - `<capability>transport/` packages own protobuf mapping for one capability.
//   - `stores*.go` owns startup-time transport dependency construction only.
//
// Capability groups:
//   - campaign, participant, character, invite:
//     campaign governance, roster, and profile lifecycle
//   - session, scene, communication:
//     active play, gates, spotlight, and communication control
//   - fork, snapshot, event, timeline:
//     replay-oriented reads, branching, and audit/history surfaces
//   - authorization:
//     actor/resource policy checks
//
// Reading order for contributors:
//  1. the root `*_service.go` file for the capability,
//  2. the owning `*_application.go` files,
//  3. the capability-local `*transport/` package,
//  4. the owning domain package under `internal/services/game/domain/...`.
//
// Non-goals:
//   - aggregate invariants; those still live in domain packages,
//   - system-specific behavior; that belongs in sibling packages under
//     `api/grpc/systems/<system>/`,
//   - direct projection/storage mutation from write handlers; writes still go
//     through the shared command/event path.
package game
