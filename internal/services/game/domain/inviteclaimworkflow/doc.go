// Package inviteclaimworkflow owns the intentional invite-claim write-path
// exception that binds a participant seat and claims the invite atomically.
//
// The participant and invite aggregates still own their local bind/claim rules.
// This sibling workflow package exists only to translate one transport intent
// into one command decision that emits `participant.bound` followed by
// `invite.claimed` in a single journal append.
package inviteclaimworkflow
