// Package state contains the Daggerheart campaign snapshot aggregate
// state and all supporting value types: SnapshotState, CharacterProfile,
// character class/subclass/companion runtime state, adversary/environment/
// countdown state, subclass track progression helpers, and factory
// functions.
//
// Imports are limited to sibling packages (ids, profile, projectionstore,
// contentstore) and internal sub-packages (mechanics, rules) that do not
// import snapstate back, keeping the dependency graph acyclic.
//
// External callers should import this package directly instead of relying on
// compatibility aliases from the root daggerheart package.
package state
