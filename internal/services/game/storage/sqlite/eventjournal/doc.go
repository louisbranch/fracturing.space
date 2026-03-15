// Package eventjournal implements the SQLite-backed game event journal.
//
// Why this package exists:
//   - It owns immutable event append/query behavior and journal integrity checks.
//   - It owns audit writes and event-backed outbox provider seams that live on
//     the journal database.
//   - It keeps journal concerns separate from the core projection backend so the
//     root sqlite projection package no longer acts as the authority for both
//     append and materialization.
package eventjournal
