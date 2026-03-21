// Package game provides root transport concern builders, domain-layer adapters,
// and a small set of infrastructure-level gRPC services (integration,
// statistics, system) for the game service.
//
// Entity-scoped gRPC services live in subpackages:
//
//	campaigntransport/        campaign governance and AI binding
//	participanttransport/     roster and social profile lifecycle
//	charactertransport/       character CRUD and profile management
//	sessiontransport/         session lifecycle, gates, spotlight, communication
//	scenetransport/           scene CRUD, character membership
//	forktransport/            campaign fork management
//	snapshottransport/        character snapshot CRUD
//	eventtransport/           event timeline, replay, append
//	authorizationtransport/   actor/resource policy checks
//
// Shared foundations:
//
//	authz/      authorization policy enforcement and evaluation
//	handler/    domain write helpers, pagination, mapping utilities
//	gametest/   shared test infrastructure (fakes, fixtures, runtime)
//
// Root files:
//   - stores*.go: explicit projection/infrastructure/content/runtime concern
//     builders plus root validation for startup wiring
//   - campaign_ai_orchestration_*.go: explicit AI GM turn orchestration
//     boundary using root-game deps instead of a broad root bag
//   - domain_adapter.go: domain/system bridge adapters
//   - integration_service.go, statistics_service.go, system_service.go:
//     infrastructure services without entity-specific application layers
//   - write_path_architecture_test.go: architectural governance tests
package game
