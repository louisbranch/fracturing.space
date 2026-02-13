// Package grpc contains gRPC service implementations and middleware.
//
// This package is organized by concern:
//   - game: system-agnostic game services
//   - systems/daggerheart: Daggerheart-specific mechanics
//   - metadata: request metadata helpers and interceptors
//   - interceptors: cross-cutting gRPC middleware
//
// Future game systems will add their own subpackages under systems/.
package grpc
