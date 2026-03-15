// Package daggerheartprojection provides the SQLite-backed Daggerheart
// gameplay projection backend.
//
// This package owns Daggerheart-specific projection persistence while reusing
// the shared root projections database and transaction model. The root
// `storage/sqlite` package remains the owner of projection DB lifecycle and
// exact-once apply orchestration; this package owns only the Daggerheart
// gameplay rows and conversions.
package daggerheartprojection
