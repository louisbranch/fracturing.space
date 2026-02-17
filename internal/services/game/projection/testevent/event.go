package testevent

import "time"

// Type identifies the type of a campaign event.
type Type string

// Campaign lifecycle events.
const (
	TypeCampaignCreated Type = "campaign.created"
	TypeCampaignForked  Type = "campaign.forked"
	TypeCampaignUpdated Type = "campaign.updated"
)

// Participant events.
const (
	TypeParticipantJoined  Type = "participant.joined"
	TypeParticipantLeft    Type = "participant.left"
	TypeParticipantUpdated Type = "participant.updated"
	TypeParticipantBound   Type = "participant.bound"
	TypeParticipantUnbound Type = "participant.unbound"
	TypeSeatReassigned     Type = "seat.reassigned"
)

// Invite events.
const (
	TypeInviteCreated Type = "invite.created"
	TypeInviteClaimed Type = "invite.claimed"
	TypeInviteRevoked Type = "invite.revoked"
	TypeInviteUpdated Type = "invite.updated"
)

// Character events.
const (
	TypeCharacterCreated Type = "character.created"
	TypeCharacterDeleted Type = "character.deleted"
	TypeCharacterUpdated Type = "character.updated"
	TypeProfileUpdated   Type = "character.profile_updated"
)

// Session events.
const (
	TypeSessionStarted          Type = "session.started"
	TypeSessionEnded            Type = "session.ended"
	TypeSessionGateOpened       Type = "session.gate_opened"
	TypeSessionGateResolved     Type = "session.gate_resolved"
	TypeSessionGateAbandoned    Type = "session.gate_abandoned"
	TypeSessionSpotlightSet     Type = "session.spotlight_set"
	TypeSessionSpotlightCleared Type = "session.spotlight_cleared"
)

// ActorType identifies who or what triggered an event.
type ActorType string

const (
	ActorTypeSystem      ActorType = "system"
	ActorTypeParticipant ActorType = "participant"
	ActorTypeGM          ActorType = "gm"
)

// Event represents an immutable event in the unified event journal.
type Event struct {
	CampaignID     string
	Seq            uint64
	Hash           string
	PrevHash       string
	ChainHash      string
	Signature      string
	SignatureKeyID string
	Timestamp      time.Time
	Type           Type
	SessionID      string
	RequestID      string
	InvocationID   string
	ActorType      ActorType
	ActorID        string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	PayloadJSON    []byte
}

// Campaign lifecycle payloads.
type CampaignCreatedPayload struct {
	Name         string `json:"name"`
	Locale       string `json:"locale"`
	GameSystem   string `json:"game_system"`
	GmMode       string `json:"gm_mode"`
	Intent       string `json:"intent,omitempty"`
	AccessPolicy string `json:"access_policy,omitempty"`
	ThemePrompt  string `json:"theme_prompt,omitempty"`
}

type CampaignForkedPayload struct {
	ParentCampaignID string `json:"parent_campaign_id"`
	ForkEventSeq     uint64 `json:"fork_event_seq"`
	OriginCampaignID string `json:"origin_campaign_id"`
	CopyParticipants bool   `json:"copy_participants"`
}

type CampaignUpdatedPayload struct {
	Fields map[string]any `json:"fields"`
}

// Participant payloads.
type ParticipantJoinedPayload struct {
	ParticipantID  string `json:"participant_id"`
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Controller     string `json:"controller"`
	CampaignAccess string `json:"campaign_access"`
}

type ParticipantLeftPayload struct {
	ParticipantID string `json:"participant_id"`
	Reason        string `json:"reason,omitempty"`
}

type ParticipantUpdatedPayload struct {
	ParticipantID string         `json:"participant_id"`
	Fields        map[string]any `json:"fields"`
}

type ParticipantBoundPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
}

type ParticipantUnboundPayload struct {
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
	Reason        string `json:"reason,omitempty"`
}

type SeatReassignedPayload struct {
	ParticipantID string `json:"participant_id"`
	PriorUserID   string `json:"prior_user_id"`
	UserID        string `json:"user_id"`
	Reason        string `json:"reason,omitempty"`
}

// Invite payloads.
type InviteCreatedPayload struct {
	InviteID               string `json:"invite_id"`
	ParticipantID          string `json:"participant_id"`
	RecipientUserID        string `json:"recipient_user_id,omitempty"`
	Status                 string `json:"status"`
	CreatedByParticipantID string `json:"created_by_participant_id,omitempty"`
}

type InviteClaimedPayload struct {
	InviteID      string `json:"invite_id"`
	ParticipantID string `json:"participant_id"`
	UserID        string `json:"user_id"`
	JWTID         string `json:"jti"`
}

type InviteRevokedPayload struct {
	InviteID string `json:"invite_id"`
}

type InviteUpdatedPayload struct {
	InviteID string `json:"invite_id"`
	Status   string `json:"status"`
}

// Character payloads.
type CharacterCreatedPayload struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Notes       string `json:"notes,omitempty"`
}

type CharacterUpdatedPayload struct {
	CharacterID string         `json:"character_id"`
	Fields      map[string]any `json:"fields"`
}

type CharacterDeletedPayload struct {
	CharacterID string `json:"character_id"`
	Reason      string `json:"reason,omitempty"`
}

type ProfileUpdatedPayload struct {
	CharacterID   string         `json:"character_id"`
	SystemProfile map[string]any `json:"system_profile,omitempty"`
}

// Session payloads.
type SessionStartedPayload struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name,omitempty"`
}

type SessionEndedPayload struct {
	SessionID string `json:"session_id"`
}

type SessionGateOpenedPayload struct {
	GateID   string         `json:"gate_id"`
	GateType string         `json:"gate_type"`
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type SessionGateResolvedPayload struct {
	GateID     string         `json:"gate_id"`
	Decision   string         `json:"decision,omitempty"`
	Resolution map[string]any `json:"resolution,omitempty"`
}

type SessionGateAbandonedPayload struct {
	GateID string `json:"gate_id"`
	Reason string `json:"reason,omitempty"`
}

type SessionSpotlightSetPayload struct {
	SpotlightType string `json:"spotlight_type"`
	CharacterID   string `json:"character_id,omitempty"`
}

type SessionSpotlightClearedPayload struct {
	Reason string `json:"reason,omitempty"`
}
