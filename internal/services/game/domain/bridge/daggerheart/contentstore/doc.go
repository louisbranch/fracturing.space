// Package contentstore defines Daggerheart-owned content/catalog persistence
// contracts.
//
// Why this package exists:
//   - Daggerheart catalog vocabulary belongs to the Daggerheart system, not the
//     shared game storage boundary.
//   - Daggerheart gRPC content APIs, gameplay helpers, importer tooling, and
//     sqlite content persistence can all depend on one system-owned contract.
//   - Shared storage contracts stay focused on genuinely cross-system service
//     storage concerns rather than accumulating reference-system content types.
package contentstore
