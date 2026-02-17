// Package command defines the canonical command envelope and contract used across
// the write path.
//
// Commands express business intent from API callers and tooling. They are the
// stable boundary before domain deciders so that business rules are evaluated only
// against normalized, ownership-aware inputs.
//
// The package-level registry and definitions exist to keep command behavior
// consistent for: authorization ownership (core vs system), payload compatibility,
// gate policy, and actor identity defaults.
package command
