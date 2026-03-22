// Package dhids defines Daggerheart-specific domain entity identifier newtypes.
//
// These types were extracted from the core ids package to enforce the boundary
// between system-agnostic and system-specific identifiers. The types are
// string-based newtypes that marshal to and from JSON transparently.
//
// This is a leaf package with no internal imports, safe to reference from any
// Daggerheart domain or infrastructure package without import cycles.
package dhids
