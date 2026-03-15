// Package sqlite is the thin compatibility and constructor entrypoint for the
// SQLite game backends.
//
// Why this package exists:
//   - It preserves the historical import path used by tests and older tooling
//     while production code cuts over to the concrete sibling backends.
//   - It exposes explicit constructors for the extracted event-journal and shared
//     projection backends.
//   - It keeps root-level compatibility glue small enough that authority stays in
//     the dedicated backend packages instead of here.
//
// Concrete backend ownership now lives in sibling packages:
//   - `storage/sqlite/eventjournal` owns immutable event append/query, audit, and
//     event-backed outbox provider seams.
//   - `storage/sqlite/coreprojection` owns shared projection materialization,
//     exact-once apply, watermarks, snapshots, and projection-side transaction
//     orchestration.
//   - `storage/sqlite/integrationoutbox` owns worker delivery persistence.
//   - `storage/sqlite/daggerheartcontent` and
//     `storage/sqlite/daggerheartprojection` own system-specific persistence.
package sqlite
