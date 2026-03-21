// Package outcometransport owns the Daggerheart transport handlers that derive
// durable roll outcomes from already-resolved roll events.
//
// It keeps the root Daggerheart service thin while preserving the existing
// public gRPC surface. Read this package by workflow seam:
// `handler_apply_roll_outcome.go` for durable writes,
// `handler_attack_outcome.go` and `handler_reaction_outcome.go` for transport
// derivation, then `handler_helpers.go` for shared validation and idempotency.
package outcometransport
