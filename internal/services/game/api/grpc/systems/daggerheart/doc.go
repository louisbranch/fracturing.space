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
// Package-local store contracts in stores.go define the intended gameplay and
// content dependencies instead of teaching core storage bundles as the default
// extension pattern.
//
// Transport entrypoints are intentionally split by contribution area:
//   - deterministic mechanics/read endpoints (`service.go`, `workflow_outcome_service.go`)
//   - session gameplay workflow handlers (`workflow_session_service.go`, `session_*flow.go`)
//   - state mutation applications (`*_application.go`, `actions_*.go`)
//   - content/catalog reads (`content_*`, `asset_service.go`)
//   - package-local write/runtime/store seams (`domain_write_helper.go`, `stores.go`)
//
// The package is intentionally isolated from `api/grpc/game`; shared transport
// helpers must stay in common internal packages, not by reaching back through
// the root game transport surface.
//
// This package implements systems.daggerheart.v1.DaggerheartService from
// api/proto/systems/daggerheart/v1/service.proto.
package daggerheart
