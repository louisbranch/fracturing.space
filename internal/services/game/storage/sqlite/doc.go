// Package sqlite implements game persistence contracts for event journal, projection
// materialization, and content/catalog data.
//
// Why this package exists:
// - It is the concrete backend where the write model and projection model meet.
// - It owns migration and schema-compatibility behavior for campaign history durability.
// - It provides deterministic adapters so command execution and replay paths share the same persistence shape.
//
// The backend uses generated SQL query helpers and embedded migrations; only this
// package translates domain-shaped records into concrete SQL rows/transactions.
package sqlite
