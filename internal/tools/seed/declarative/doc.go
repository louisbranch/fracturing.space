// Package declarative provides idempotent, manifest-driven local data seeding.
//
// The declarative runner coordinates auth, social, game, and discovery APIs
// to create reusable local development datasets without relying on direct DB
// writes. It intentionally excludes auth account-profile surfaces.
package declarative
