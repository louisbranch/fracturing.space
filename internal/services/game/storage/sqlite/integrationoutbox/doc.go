// Package integrationoutbox implements the SQLite backend for durable
// game-owned integration outbox work.
//
// Why this package exists:
//   - It makes worker-facing integration delivery persistence a visible backend
//     concern instead of another responsibility hidden in the event-journal
//     backend.
//   - It keeps event-journal append logic responsible for enqueue triggers while
//     moving lease/ack/query behavior behind a dedicated backend boundary.
//   - It lets contributors reason about journal persistence and integration
//     delivery separately inside the SQLite backend family.
package integrationoutbox
