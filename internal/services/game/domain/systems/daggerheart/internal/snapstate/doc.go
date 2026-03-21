// Package snapstate contains the Daggerheart campaign snapshot aggregate
// state and all supporting value types: SnapshotState, CharacterProfile,
// character class/subclass/companion runtime state, adversary/environment/
// countdown state, subclass track progression helpers, and factory
// functions.
//
// Imports are limited to sibling packages (ids, profile, projectionstore,
// contentstore) and internal sub-packages (mechanics, rules) that do not
// import snapstate back, keeping the dependency graph acyclic.
//
// The root daggerheart package re-exports all public symbols via type
// aliases and function variables so external consumers are unaffected.
package snapstate
