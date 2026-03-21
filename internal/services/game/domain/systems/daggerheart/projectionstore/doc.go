// Package projectionstore defines Daggerheart-owned projection-state persistence
// contracts.
//
// Why this package exists:
//   - Daggerheart projection vocabulary belongs to the Daggerheart system, not the
//     shared game storage boundary.
//   - The manifest adapter seam, gameplay reads, and replay helpers can all depend
//     on one system-owned contract without importing Daggerheart catalog content.
//   - Concrete backends such as sqlite can implement the same seam without
//     hard-wiring shared storage to one reference system's state model.
package projectionstore
