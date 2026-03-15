// Package workflowruntime owns the shared Daggerheart workflow write/runtime
// support used by multiple sibling transport packages.
//
// The root Daggerheart service still assembles transport dependencies, but the
// actual session replay checks and Daggerheart system-command construction live
// here so they do not remain as root-only helpers.
package workflowruntime
