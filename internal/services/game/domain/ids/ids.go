// Package ids defines domain entity identifier newtypes.
//
// All entity identifiers are distinct named string types, preventing
// accidental cross-entity aliasing at compile time. The types marshal
// to and from JSON transparently (underlying string encoding).
//
// This is a leaf package with no internal imports, safe to reference
// from any domain or infrastructure package without import cycles.
package ids

// CampaignID identifies an aggregate root campaign.
type CampaignID string

func (id CampaignID) String() string { return string(id) }

// ParticipantID identifies a participant within a campaign.
type ParticipantID string

func (id ParticipantID) String() string { return string(id) }

// CharacterID identifies a character within a campaign.
type CharacterID string

func (id CharacterID) String() string { return string(id) }

// SessionID identifies a session within a campaign.
type SessionID string

func (id SessionID) String() string { return string(id) }

// SceneID identifies a scene within a session.
type SceneID string

func (id SceneID) String() string { return string(id) }

// InviteID identifies an invite within a campaign.
type InviteID string

func (id InviteID) String() string { return string(id) }

// UserID identifies an external authenticated user.
type UserID string

func (id UserID) String() string { return string(id) }

// GateID identifies a session or scene gate.
type GateID string

func (id GateID) String() string { return string(id) }

// AdversaryID identifies a Daggerheart adversary.
type AdversaryID string

func (id AdversaryID) String() string { return string(id) }

// CountdownID identifies a Daggerheart countdown.
type CountdownID string

func (id CountdownID) String() string { return string(id) }
