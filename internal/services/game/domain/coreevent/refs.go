package coreevent

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	domainevent "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// Type aliases the canonical runtime event type for scenario tooling/tests.
type Type = domainevent.Type

const (
	TypeCampaignCreated         Type = "campaign.created"
	TypeParticipantJoined       Type = "participant.joined"
	TypeSessionStarted          Type = "session.started"
	TypeSessionEnded            Type = "session.ended"
	TypeSessionGateOpened       Type = "session.gate_opened"
	TypeSessionGateResolved     Type = "session.gate_resolved"
	TypeSessionSpotlightSet     Type = "session.spotlight_set"
	TypeSessionSpotlightCleared Type = "session.spotlight_cleared"
	TypeCharacterCreated        Type = "character.created"
	TypeCharacterDeleted        Type = "character.deleted"
	TypeRollResolved            Type = "action.roll_resolved"
	TypeOutcomeApplied          Type = "action.outcome_applied"
	TypeOutcomeRejected         Type = "action.outcome_rejected"
)

// SessionGateOpenedPayload aliases the canonical session gate payload.
type SessionGateOpenedPayload = session.GateOpenedPayload

// RollResolvedPayload aliases the canonical roll resolved payload.
type RollResolvedPayload = action.RollResolvePayload
