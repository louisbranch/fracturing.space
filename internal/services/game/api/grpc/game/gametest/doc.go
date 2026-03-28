// Package gametest provides shared core store fakes and record fixtures for
// the game gRPC service and its entity-scoped subpackages. System-specific
// fakes live in their owning system testkits.
//
// This package must NOT import the parent game package to avoid import cycles.
package gametest
