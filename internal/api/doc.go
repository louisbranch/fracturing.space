// Package api contains API service implementations.
//
// This package organizes API handlers by transport and concern:
//
// # gRPC Services
//
// The grpc subpackage contains gRPC service implementations organized by domain:
//
//   - grpc/state/: System-agnostic state management services
//     (CampaignService, SessionService, SnapshotService, ParticipantService, CharacterService)
//   - grpc/systems/daggerheart/: Daggerheart-specific mechanics service
//
// # Service Boundaries
//
// State services (under grpc/state/) are system-agnostic - they work the same
// regardless of whether the campaign uses Daggerheart, D&D 5e, or any other system.
//
// System services (under grpc/systems/{name}/) provide system-specific mechanics.
// They call state services to persist changes to snapshot and session state.
//
// # MCP Integration
//
// MCP tools call these gRPC services. The tool layer (internal/mcp/tool/) handles
// MCP-specific concerns like tool schemas and context, while the gRPC layer
// handles business logic.
package api
