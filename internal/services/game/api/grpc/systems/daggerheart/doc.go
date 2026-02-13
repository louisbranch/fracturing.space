// Package daggerheart provides the Daggerheart gRPC service implementation.
//
// This service exposes deterministic Daggerheart mechanics, including:
//   - Action rolls with Duality dice (Hope d12 + Fear d12)
//   - Outcome evaluation and explanation
//   - Outcome probability calculations
//   - Ruleset metadata for replay and validation
//   - Generic dice rolling utilities
//
// The service is read-only with respect to game state; it returns mechanics
// outcomes without mutating campaigns or snapshots.
//
// This package implements systems.daggerheart.v1.DaggerheartService from
// api/proto/systems/daggerheart/v1/service.proto.
package daggerheart
