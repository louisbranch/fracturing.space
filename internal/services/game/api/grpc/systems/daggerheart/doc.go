// Package daggerheart provides the Daggerheart gRPC service implementation.
//
// This service exposes deterministic Daggerheart mechanics, including:
//   - Action rolls with Duality dice (Hope d12 + Fear d12)
//   - Outcome evaluation and explanation
//   - Outcome probability calculations
//   - Ruleset metadata for replay and validation
//   - Generic dice rolling utilities
//   - Campaign/session mutation workflows for Daggerheart-specific mechanics
//
// The package includes both read-only mechanics endpoints and write-path
// endpoints that emit domain commands/events for campaign state mutation.
//
// This package implements systems.daggerheart.v1.DaggerheartService from
// api/proto/systems/daggerheart/v1/service.proto.
package daggerheart
