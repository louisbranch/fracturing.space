// Package projectionapplyoutbox implements the SQLite backend for durable
// projection-apply outbox queue state stored alongside the event journal.
//
// Why this package exists:
//   - It makes projection apply queue ownership explicit instead of hiding it in
//     the event-journal backend.
//   - It keeps event-journal append logic responsible only for enqueue hooks
//     while moving worker queue processing, inspection, and requeue behavior into
//     a dedicated backend package.
//   - It keeps the SQLite backend family split between journal, projection, and
//     operational queue concerns instead of recreating another catch-all surface.
package projectionapplyoutbox
