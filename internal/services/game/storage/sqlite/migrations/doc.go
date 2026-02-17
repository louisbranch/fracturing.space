// Package migrations embeds SQL migration scripts used by SQLite backends.
//
// Why this package exists:
// - It centralizes schema history for events, projections, and catalog stores.
// - It allows upgrade and replay-safe evolution without manual operator SQL.
// - It supports both development bootstrap and production migration workflows.
package migrations
