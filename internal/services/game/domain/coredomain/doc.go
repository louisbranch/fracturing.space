// Package coredomain defines the package-owned contract surface core domain
// packages expose to aggregate replay and engine bootstrap.
//
// This package intentionally carries only the non-aggregate registration hooks.
// Aggregate composes these contracts with aggregate-owned fold adapters so core
// domains can publish one cohesive descriptor without depending on aggregate
// state types.
package coredomain
