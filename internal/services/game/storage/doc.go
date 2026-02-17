// Package storage defines the persistence contracts for both write-side event
// storage and read-model materialization.
//
// Why this package exists:
// - It gives commands and projections a stable persistence vocabulary.
// - It separates domain intent from concrete backends (sqlite, tests, mocks).
// - It keeps command-safety checks and read-model shaping in one boundary.
//
// Covered domains include campaigns, participants, characters, invites, sessions,
// events, snapshots, forks, telemetry, statistics, roll outcomes, and
// Daggerheart extensions.
//
// Concrete implementations are in subpackages (for example: internal/services/game/storage/sqlite).
//
// Common error types:
// - ErrNotFound: requested record is missing
// - ErrActiveSessionExists: conflict starting a new session
package storage
