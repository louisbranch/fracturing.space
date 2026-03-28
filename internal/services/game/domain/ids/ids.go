// Package ids defines domain entity identifier newtypes.
//
// All entity identifiers are distinct named string types, preventing
// accidental cross-entity aliasing at compile time. The types marshal
// to and from JSON transparently (underlying string encoding).
//
// This is a leaf package with no internal imports, safe to reference
// from any domain or infrastructure package without import cycles.
package ids

import "strings"

// CampaignID identifies an aggregate root campaign.
type CampaignID string

func (id CampaignID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id CampaignID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// ParticipantID identifies a participant within a campaign.
type ParticipantID string

func (id ParticipantID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id ParticipantID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// CharacterID identifies a character within a campaign.
type CharacterID string

func (id CharacterID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id CharacterID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// SessionID identifies a session within a campaign.
type SessionID string

func (id SessionID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id SessionID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// SceneID identifies a scene within a session.
type SceneID string

func (id SceneID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id SceneID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// InviteID identifies an invite within a campaign.
type InviteID string

func (id InviteID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id InviteID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// UserID identifies an external authenticated user.
type UserID string

func (id UserID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id UserID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

// GateID identifies a session or scene gate.
type GateID string

func (id GateID) String() string { return string(id) }

// IsZero reports whether the identifier is empty or whitespace-only.
func (id GateID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }
