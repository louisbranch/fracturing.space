// Package domain defines the entities and lifecycle state for game sessions.
//
// A Session represents a single continuous play period within a Campaign.
// It tracks when the play started, its current status, and when it ended.
//
// # Session Lifecycle
//
// Sessions move through several statuses:
//   - Active: The session is currently ongoing. Only one session can be active per campaign.
//   - Paused: Play is temporarily suspended.
//   - Ended: The session is finished and record-keeping is complete.
package domain
