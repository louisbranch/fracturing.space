// Package api contains service API implementations.
//
// API handlers are organized by transport. Today, the gRPC transport is the
// canonical surface area for game services.
//
// Subpackages:
//   - grpc/game: system-agnostic game services (campaigns, sessions, participants,
//     characters, snapshots, forks, events, invites, statistics)
//   - grpc/systems/daggerheart: Daggerheart-specific mechanics services
//   - grpc/metadata: request metadata helpers and interceptors
//   - grpc/interceptors: cross-cutting gRPC middleware
//
// MCP tools call these gRPC services through internal/services/mcp/service.
package api
