// Package check provides generic difficulty check primitives.
//
// This package contains system-agnostic difficulty checking functionality
// that can be used by any game system. It provides:
//
//   - Basic difficulty comparison (total vs target)
//   - Margin of success/failure calculations
//
// Game-system-specific success/failure interpretation (like Daggerheart's
// Hope/Fear flavoring) is built on top of these primitives in the systems/ packages.
package check
