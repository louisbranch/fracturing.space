// Package adapter implements the Daggerheart event projection adapter that
// applies domain events to persistent projection storage. Each handler reads
// the current projection state, applies event-driven mutations, and writes
// back.
//
// The LevelUpApplier dependency is injected at construction so the adapter
// package does not depend on the root daggerheart package.
package adapter
