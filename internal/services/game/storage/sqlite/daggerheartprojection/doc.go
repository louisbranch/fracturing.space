// Package daggerheartprojection provides the SQLite-backed Daggerheart
// gameplay projection backend.
//
// This package owns Daggerheart-specific projection persistence while reusing
// the shared projections database and transaction model from
// `storage/sqlite/coreprojection`. It owns only the Daggerheart gameplay rows
// and conversions.
package daggerheartprojection
