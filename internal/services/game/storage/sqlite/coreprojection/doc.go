// Package coreprojection implements the shared SQLite projection backend for
// the game service.
//
// Why this package exists:
//   - It owns shared projection CRUD and lookup behavior for campaign, seat,
//     invite, character, session, scene, snapshot, fork, and watermark state.
//   - It owns projection-runtime seams that must run against the projection
//     database, including exact-once checkpoint apply and transaction-scoped
//     system-store rebinding.
//   - It keeps shared projection behavior separate from the immutable event
//     journal so one package no longer pretends to own both sides of the
//     event-sourced boundary.
//
// Reading order:
//   - Start with `store.go` for the backend root and transaction/open seams.
//   - Then read `store_projection_*.go` for projection record CRUD by concern.
//   - Read `store_conversion_*.go` when a change affects row-to-domain mapping.
//   - Read `store_projection_apply_once.go`, `store_snapshots.go`, and
//     `store_watermark.go` for projection-runtime support behavior.
//
// Non-goals:
//   - owning immutable event journal persistence
//   - owning integration-outbox delivery persistence
//   - hiding all projection concerns behind one broad helper file
package coreprojection
