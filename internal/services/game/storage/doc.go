// Package storage defines persistence interfaces for game services.
//
// It covers campaign metadata, participants, invites, characters, sessions,
// event journaling, snapshots, forks, telemetry, statistics, roll outcomes, and
// Daggerheart-specific extensions. Implementations (e.g., SQLite) live in
// subpackages.
//
// Common error types:
//   - ErrNotFound: requested record is missing
//   - ErrActiveSessionExists: conflict starting a new session
package storage
