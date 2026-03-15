package coreevent

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	domainevent "github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// Type aliases the canonical runtime event type for scenario tooling/tests.
type Type = domainevent.Type

const (
	TypeCampaignCreated             Type = "campaign.created"
	TypeParticipantJoined           Type = "participant.joined"
	TypeSessionStarted              Type = "session.started"
	TypeSessionEnded                Type = "session.ended"
	TypeSessionGateOpened           Type = "session.gate_opened"
	TypeSessionGateResponseRecorded Type = "session.gate_response_recorded"
	TypeSessionGateResolved         Type = "session.gate_resolved"
	TypeSessionGateAbandoned        Type = "session.gate_abandoned"
	TypeSessionSpotlightSet         Type = "session.spotlight_set"
	TypeSessionSpotlightCleared     Type = "session.spotlight_cleared"
	TypeSessionActiveSceneSet       Type = "session.active_scene_set"
	TypeSessionOOCPaused            Type = "session.ooc_paused"
	TypeSessionOOCPosted            Type = "session.ooc_posted"
	TypeSessionOOCReadyMarked       Type = "session.ooc_ready_marked"
	TypeSessionOOCReadyCleared      Type = "session.ooc_ready_cleared"
	TypeSessionOOCResumed           Type = "session.ooc_resumed"
	TypeCharacterCreated            Type = "character.created"
	TypeCharacterDeleted            Type = "character.deleted"
	TypeRollResolved                Type = "action.roll_resolved"
	TypeOutcomeApplied              Type = "action.outcome_applied"
	TypeOutcomeRejected             Type = "action.outcome_rejected"

	TypeSceneCreated              Type = "scene.created"
	TypeSceneUpdated              Type = "scene.updated"
	TypeSceneEnded                Type = "scene.ended"
	TypeSceneCharacterAdded       Type = "scene.character_added"
	TypeSceneCharacterRemoved     Type = "scene.character_removed"
	TypeSceneGateOpened           Type = "scene.gate_opened"
	TypeSceneGateResolved         Type = "scene.gate_resolved"
	TypeSceneGateAbandoned        Type = "scene.gate_abandoned"
	TypeSceneSpotlightSet         Type = "scene.spotlight_set"
	TypeSceneSpotlightCleared     Type = "scene.spotlight_cleared"
	TypeScenePlayerPhaseStarted   Type = "scene.player_phase_started"
	TypeScenePlayerPhasePosted    Type = "scene.player_phase_posted"
	TypeScenePlayerPhaseYielded   Type = "scene.player_phase_yielded"
	TypeScenePlayerPhaseUnyielded Type = "scene.player_phase_unyielded"
	TypeScenePlayerPhaseEnded     Type = "scene.player_phase_ended"
)

// SessionGateOpenedPayload aliases the canonical session gate payload.
type SessionGateOpenedPayload = session.GateOpenedPayload

// RollResolvedPayload aliases the canonical roll resolved payload.
type RollResolvedPayload = action.RollResolvePayload

// SceneCreatePayload aliases the canonical scene create payload.
type SceneCreatePayload = scene.CreatePayload
