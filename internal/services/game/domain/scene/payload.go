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
