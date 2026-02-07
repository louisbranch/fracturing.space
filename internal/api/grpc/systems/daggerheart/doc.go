// Package daggerheart provides the Daggerheart gRPC service implementation.
//
// This service implements Daggerheart-specific game mechanics including:
//   - Action rolls with Duality dice (Hope d12 + Fear d12)
//   - Outcome evaluation and explanation
//   - Probability calculations
//   - Session action rolls with event recording
//   - Outcome application (updates snapshot state)
//
// # Integration with State Services
//
// DaggerheartService calls state services to persist gameplay effects:
//   - SessionService: Record roll events
//   - SnapshotService: Update character hope/stress, GM fear
//
// # Proto Definitions
//
// This service implements the systems.daggerheart.v1.DaggerheartService from
// api/proto/systems/daggerheart/v1/service.proto.
package daggerheart
