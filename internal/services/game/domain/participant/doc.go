// Package participant models campaign membership and control rights.
//
// Participants tie users/roles/controllers to a campaign and are the unit that
// enforces permissions and identity checks for commands that mutate campaign
// state.
//
// This package is responsible for:
//   - command validation for join/leave/profile updates/seating changes,
//   - replaying participant events into compact membership state,
//   - and keeping access/identity fields aligned with campaign authorization.
package participant
