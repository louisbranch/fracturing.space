// Package sqlite provides SQLite-backed admin persistence.
//
// It stores administrative state only and intentionally remains separate from
// game event state so operator tooling cannot accidentally bypass domain rules.
package sqlite
