// Package grpc contains gRPC service implementations.
//
// This package is organized by concern:
//
//   - state/: System-agnostic state management (campaigns, sessions, participants, characters, snapshots)
//   - systems/daggerheart/: Daggerheart-specific mechanics
//
// Future game systems will add their own subpackages under systems/.
package grpc
