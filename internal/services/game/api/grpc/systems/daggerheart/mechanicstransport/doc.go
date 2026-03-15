// Package mechanicstransport owns Daggerheart deterministic mechanics and
// read-only roll helpers.
//
// The root Daggerheart gRPC package keeps the public constructor and service
// registration surface stable. This package owns the mechanics-specific
// transport implementation: seed resolution, deterministic roll handling,
// rules metadata reads, and explanation payload shaping.
package mechanicstransport
