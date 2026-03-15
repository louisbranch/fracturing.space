// Package statetransport owns shared Daggerheart gameplay-state protobuf
// mapping used by multiple root transport wrappers.
//
// It keeps common character-state response shaping out of the root Daggerheart
// package without incorrectly assigning that responsibility to any one mutation
// slice.
package statetransport
