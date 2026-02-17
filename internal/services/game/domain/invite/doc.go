// Package invite models campaign onboarding and trust transitions.
//
// Invites are short-lived coordination artifacts that bridge internal participant
// identity, email/user claims, and campaign access assignment. They intentionally
// stay explicit because they are often the last thing checked before a participant
// can meaningfully act in a campaign.
//
// The package defines invite command deciders, invite lifecycle states, and folds
// used to validate claim/revoke/update behavior.
package invite
