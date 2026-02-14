package event

import (
	"strings"
	"time"
)

// Type identifies the type of a campaign event.
type Type string

// Campaign lifecycle events.
const (
	// TypeCampaignCreated records the creation of a campaign.
	TypeCampaignCreated Type = "campaign.created"
	// TypeCampaignForked records the forking of a campaign.
	TypeCampaignForked Type = "campaign.forked"
	// TypeCampaignUpdated records updates to campaign metadata.
	TypeCampaignUpdated Type = "campaign.updated"
)

// Participant events.
const (
	// TypeParticipantJoined records a participant joining a campaign.
	TypeParticipantJoined Type = "participant.joined"
	// TypeParticipantLeft records a participant leaving a campaign.
	TypeParticipantLeft Type = "participant.left"
	// TypeParticipantUpdated records updates to a participant.
	TypeParticipantUpdated Type = "participant.updated"
	// TypeParticipantBound records binding a user to a participant seat.
	TypeParticipantBound Type = "participant.bound"
	// TypeParticipantUnbound records unbinding a user from a participant seat.
	TypeParticipantUnbound Type = "participant.unbound"
	// TypeSeatReassigned records moving a seat between users.
	TypeSeatReassigned Type = "seat.reassigned"
)

// Invite events.
const (
	// TypeInviteCreated records the creation of an invite.
	TypeInviteCreated Type = "invite.created"
	// TypeInviteClaimed records an invite being claimed.
	TypeInviteClaimed Type = "invite.claimed"
	// TypeInviteRevoked records an invite being revoked.
	TypeInviteRevoked Type = "invite.revoked"
	// TypeInviteUpdated records updates to an invite.
	TypeInviteUpdated Type = "invite.updated"
)

// Character events.
const (
	// TypeCharacterCreated records the creation of a character.
	TypeCharacterCreated Type = "character.created"
	// TypeCharacterDeleted records the deletion of a character.
	TypeCharacterDeleted Type = "character.deleted"
	// TypeCharacterUpdated records updates to character metadata.
	TypeCharacterUpdated Type = "character.updated"
	// TypeProfileUpdated records updates to a character profile.
	TypeProfileUpdated Type = "character.profile_updated"
)

// Session events.
const (
	// TypeSessionStarted records the start of a session.
	TypeSessionStarted Type = "session.started"
	// TypeSessionEnded records the end of a session.
	TypeSessionEnded Type = "session.ended"
)

// Action events (gameplay actions within sessions).
// Events represent facts that have occurred, not commands/requests.
const (
	// TypeRollResolved records a roll resolution.
	TypeRollResolved Type = "action.roll_resolved"
	// TypeOutcomeApplied records a successful outcome application.
	TypeOutcomeApplied Type = "action.outcome_applied"
	// TypeOutcomeRejected records a rejected outcome application.
	TypeOutcomeRejected Type = "action.outcome_rejected"
	// TypeNoteAdded records a GM/player note.
	TypeNoteAdded Type = "action.note_added"
)

// ActorType identifies who or what triggered an event.
type ActorType string

const (
	// ActorTypeSystem indicates the event was triggered by the system.
	ActorTypeSystem ActorType = "system"
	// ActorTypeParticipant indicates the event was triggered by a participant.
	ActorTypeParticipant ActorType = "participant"
	// ActorTypeGM indicates the event was triggered by the GM.
	ActorTypeGM ActorType = "gm"
)

// Event represents an immutable event in the unified event journal.
type Event struct {
	// CampaignID is the campaign this event belongs to.
	CampaignID string
	// Seq is the event sequence number within the campaign (starts at 1).
	// Assigned by storage on append.
	Seq uint64
	// Hash is the content-addressed identity (SHA-256 truncated to 128-bit).
	// Assigned by storage on append.
	Hash string
	// PrevHash is the previous event's chain hash (empty for the first event).
	// Assigned by storage on append.
	PrevHash string
	// ChainHash links this event to the previous event hash (SHA-256).
	// Assigned by storage on append.
	ChainHash string
	// SignatureKeyID identifies the HMAC key used to sign the chain hash.
	// Assigned by storage on append.
	SignatureKeyID string
	// Signature is the HMAC signature of the chain hash.
	// Assigned by storage on append.
	Signature string
	// Timestamp is when the event occurred.
	Timestamp time.Time
	// Type identifies the kind of event.
	Type Type
	// SessionID groups events into sessions (empty for setup events).
	SessionID string
	// RequestID correlates related events (e.g., roll request to resolution).
	RequestID string
	// InvocationID tracks the MCP/gRPC invocation that triggered the event.
	InvocationID string
	// ActorType identifies who triggered the event.
	ActorType ActorType
	// ActorID is the participant ID if ActorType is participant or GM.
	ActorID string
	// EntityType is the type of entity affected (character, session, etc.).
	EntityType string
	// EntityID is the ID of the entity affected.
	EntityID string
	// SystemID identifies the game system for this event (optional for core-only events).
	SystemID string
	// SystemVersion identifies the system ruleset version (optional for core-only events).
	SystemVersion string
	// PayloadJSON holds event-specific data as JSON.
	PayloadJSON []byte
}

// IsValid reports whether the event type is usable.
func (t Type) IsValid() bool {
	return strings.TrimSpace(string(t)) != ""
}

// Domain returns the domain prefix of the event type (e.g., "campaign", "character").
func (t Type) Domain() string {
	for i, c := range t {
		if c == '.' {
			return string(t[:i])
		}
	}
	return string(t)
}
