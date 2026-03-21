package scene

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// CreatePayload captures the payload for scene.create commands and scene.created events.
type CreatePayload struct {
	SceneID      ids.SceneID       `json:"scene_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	CharacterIDs []ids.CharacterID `json:"character_ids"`
}

// UpdatePayload captures the payload for scene.update commands and scene.updated events.
type UpdatePayload struct {
	SceneID     ids.SceneID `json:"scene_id"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
}

// EndPayload captures the payload for scene.end commands and scene.ended events.
type EndPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	Reason  string      `json:"reason,omitempty"`
}

// CharacterAddedPayload captures the payload for scene.character_added events.
type CharacterAddedPayload struct {
	SceneID     ids.SceneID     `json:"scene_id"`
	CharacterID ids.CharacterID `json:"character_id"`
}

// CharacterRemovedPayload captures the payload for scene.character_removed events.
type CharacterRemovedPayload struct {
	SceneID     ids.SceneID     `json:"scene_id"`
	CharacterID ids.CharacterID `json:"character_id"`
}

// CharacterTransferPayload captures the payload for scene.character.transfer commands.
type CharacterTransferPayload struct {
	SourceSceneID ids.SceneID     `json:"source_scene_id"`
	TargetSceneID ids.SceneID     `json:"target_scene_id"`
	CharacterID   ids.CharacterID `json:"character_id"`
}

// TransitionPayload captures the payload for scene.transition commands.
type TransitionPayload struct {
	SourceSceneID ids.SceneID `json:"source_scene_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	NewSceneID    ids.SceneID `json:"new_scene_id"`
}

// GateOpenedPayload captures the payload for scene.gate_opened events.
type GateOpenedPayload struct {
	SceneID  ids.SceneID    `json:"scene_id"`
	GateID   ids.GateID     `json:"gate_id"`
	GateType string         `json:"gate_type"`
	Reason   string         `json:"reason,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// GateResolvedPayload captures the payload for scene.gate_resolved events.
type GateResolvedPayload struct {
	SceneID    ids.SceneID    `json:"scene_id"`
	GateID     ids.GateID     `json:"gate_id"`
	Decision   string         `json:"decision,omitempty"`
	Resolution map[string]any `json:"resolution,omitempty"`
}

// GateAbandonedPayload captures the payload for scene.gate_abandoned events.
type GateAbandonedPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	GateID  ids.GateID  `json:"gate_id"`
	Reason  string      `json:"reason,omitempty"`
}

// SpotlightSetPayload captures the payload for scene.spotlight_set events.
type SpotlightSetPayload struct {
	SceneID       ids.SceneID     `json:"scene_id"`
	SpotlightType SpotlightType   `json:"spotlight_type"`
	CharacterID   ids.CharacterID `json:"character_id,omitempty"`
}

// SpotlightClearedPayload captures the payload for scene.spotlight_cleared events.
type SpotlightClearedPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	Reason  string      `json:"reason,omitempty"`
}

// PlayerPhaseStartedPayload captures the payload for scene.player_phase_started events.
type PlayerPhaseStartedPayload struct {
	SceneID              ids.SceneID         `json:"scene_id"`
	PhaseID              string              `json:"phase_id"`
	ActingCharacterIDs   []ids.CharacterID   `json:"acting_character_ids,omitempty"`
	ActingParticipantIDs []ids.ParticipantID `json:"acting_participant_ids,omitempty"`
}

// PlayerPhasePostedPayload captures the payload for scene.player_phase_posted events.
type PlayerPhasePostedPayload struct {
	SceneID       ids.SceneID       `json:"scene_id"`
	PhaseID       string            `json:"phase_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
	CharacterIDs  []ids.CharacterID `json:"character_ids,omitempty"`
	SummaryText   string            `json:"summary_text,omitempty"`
}

// PlayerPhaseYieldedPayload captures the payload for scene.player_phase_yielded events.
type PlayerPhaseYieldedPayload struct {
	SceneID       ids.SceneID       `json:"scene_id"`
	PhaseID       string            `json:"phase_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
}

// PlayerPhaseReviewStartedPayload captures the payload for scene.player_phase_review_started events.
type PlayerPhaseReviewStartedPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	PhaseID string      `json:"phase_id"`
}

// PlayerPhaseUnyieldedPayload captures the payload for scene.player_phase_unyielded events.
type PlayerPhaseUnyieldedPayload struct {
	SceneID       ids.SceneID       `json:"scene_id"`
	PhaseID       string            `json:"phase_id"`
	ParticipantID ids.ParticipantID `json:"participant_id"`
}

// PlayerPhaseRevisionRequest captures one participant-scoped return-for-correction target.
type PlayerPhaseRevisionRequest struct {
	ParticipantID ids.ParticipantID `json:"participant_id"`
	Reason        string            `json:"reason,omitempty"`
	CharacterIDs  []ids.CharacterID `json:"character_ids,omitempty"`
}

// PlayerPhaseRevisionsRequestedPayload captures the payload for scene.player_phase_revisions_requested events.
type PlayerPhaseRevisionsRequestedPayload struct {
	SceneID   ids.SceneID                  `json:"scene_id"`
	PhaseID   string                       `json:"phase_id"`
	Revisions []PlayerPhaseRevisionRequest `json:"revisions,omitempty"`
}

// PlayerPhaseAcceptedPayload captures the payload for scene.player_phase_accepted events.
type PlayerPhaseAcceptedPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	PhaseID string      `json:"phase_id"`
}

// PlayerPhaseEndedPayload captures the payload for scene.player_phase_ended events.
type PlayerPhaseEndedPayload struct {
	SceneID ids.SceneID `json:"scene_id"`
	PhaseID string      `json:"phase_id"`
	Reason  string      `json:"reason,omitempty"`
}

type GMInteractionBeatType string

const (
	GMInteractionBeatTypeFiction     GMInteractionBeatType = "fiction"
	GMInteractionBeatTypePrompt      GMInteractionBeatType = "prompt"
	GMInteractionBeatTypeResolution  GMInteractionBeatType = "resolution"
	GMInteractionBeatTypeConsequence GMInteractionBeatType = "consequence"
	GMInteractionBeatTypeGuidance    GMInteractionBeatType = "guidance"
)

type GMInteractionIllustration struct {
	ImageURL string `json:"image_url,omitempty"`
	Alt      string `json:"alt,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

type GMInteractionBeat struct {
	BeatID string                `json:"beat_id"`
	Type   GMInteractionBeatType `json:"type"`
	Text   string                `json:"text,omitempty"`
}

// GMInteractionCommittedPayload captures the payload for scene.gm_interaction_committed events.
type GMInteractionCommittedPayload struct {
	SceneID       ids.SceneID                `json:"scene_id"`
	InteractionID string                     `json:"interaction_id"`
	PhaseID       string                     `json:"phase_id,omitempty"`
	ParticipantID ids.ParticipantID          `json:"participant_id"`
	Title         string                     `json:"title,omitempty"`
	CharacterIDs  []ids.CharacterID          `json:"character_ids,omitempty"`
	Illustration  *GMInteractionIllustration `json:"illustration,omitempty"`
	Beats         []GMInteractionBeat        `json:"beats,omitempty"`
}
