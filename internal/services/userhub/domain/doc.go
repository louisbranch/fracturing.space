// Package domain contains userhub aggregation logic for user-facing dashboard
// summaries.
//
// The package owns read-only composition and prioritization logic across
// upstream services (game, social, notifications). It intentionally does not
// write domain state; source services remain authoritative for mutations.
package domain
