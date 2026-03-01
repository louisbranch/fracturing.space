// Package contracts provides small deterministic helpers used by domain
// contract tests so declaration-parity assertions stay consistent across
// packages.
package contracts

import "reflect"

// EqualSlices reports whether two slices contain the same values in the same
// order. Contract tests use this to assert explicit ordering for command and
// event type declarations.
func EqualSlices[T comparable](left, right []T) bool {
	return reflect.DeepEqual(left, right)
}

// HasDuplicates reports whether a slice contains any repeated value.
func HasDuplicates[T comparable](values []T) bool {
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
