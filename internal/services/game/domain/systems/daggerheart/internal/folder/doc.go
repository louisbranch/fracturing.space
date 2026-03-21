// Package folder implements the Daggerheart event fold logic that
// projects domain events into in-memory snapshot state. The Folder
// routes events by type and applies deterministic state mutations.
//
// The LevelUpApplier dependency is injected at construction so the
// folder package does not depend on the root daggerheart package.
package folder
