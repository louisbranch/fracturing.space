// Package game provides the transport dependency injection container (Stores),
// domain-layer adapters, and a small set of infrastructure-level gRPC services
// (integration, statistics, system) for the game service.
//
// Entity-scoped gRPC services live in subpackages:
//
//	campaigntransport/        campaign governance and AI binding
//	participanttransport/     roster and social profile lifecycle
//	charactertransport/       character CRUD and profile management
//	invitetransport/          invite lifecycle, claim, revoke
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
//   - stores*.go: transport dependency container and startup construction
//   - domain_adapter.go, system_adapters.go: domain/system bridge adapters
//   - integration_service.go, statistics_service.go, system_service.go:
//     infrastructure services without entity-specific application layers
//   - write_path_architecture_test.go: architectural governance tests
package game
