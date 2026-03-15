// Package daggerheartcontent implements the SQLite backend for Daggerheart
// catalog/content persistence.
//
// Why this package exists:
//   - It keeps Daggerheart catalog persistence out of the root SQLite backend.
//   - It makes the system-specific content backend visible to contributors.
//   - It preserves one concrete SQLite adapter for Daggerheart catalog import,
//     readiness checks, and transport reads without widening the generic
//     event/projection store root.
package daggerheartcontent
