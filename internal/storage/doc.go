// Package storage defines the persistence interfaces for the Duality engine.
//
// It provides a high-level abstraction for storing campaign metadata,
// participants, characters, and session states. Implementation of these
// interfaces (e.g., using bbolt) can be found in subpackages.
//
// # Error Types
//
// The package defines common error types used across storage implementations:
//   - ErrNotFound: Indicates a requested record is missing.
//   - ErrActiveSessionExists: Indicates a conflict when starting a new session.
package storage
