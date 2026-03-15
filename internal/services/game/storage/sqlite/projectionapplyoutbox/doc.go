// Package projectionapplyoutbox implements the SQLite backend for durable
// projection-apply outbox queue state stored alongside the event journal.
//
// Why this package exists:
//   - It makes projection apply queue ownership explicit instead of hiding it in
//     the root SQLite event store.
//   - It keeps event-journal append logic responsible only for enqueue hooks
//     while moving worker queue processing, inspection, and requeue behavior into
//     a dedicated backend package.
//   - It reduces the root `storage/sqlite` package toward journal and projection
//     primitives instead of another catch-all operational surface.
package projectionapplyoutbox
