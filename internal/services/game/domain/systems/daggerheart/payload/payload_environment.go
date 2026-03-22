package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
)

// --- Environment entities ---

// EnvironmentEntityCreatePayload captures the payload for
// sys.daggerheart.environment_entity.create commands.
type EnvironmentEntityCreatePayload struct {
	EnvironmentEntityID dhids.EnvironmentEntityID `json:"environment_entity_id"`
	EnvironmentID       string                    `json:"environment_id"`
	Name                string                    `json:"name"`
	Type                string                    `json:"type"`
	Tier                int                       `json:"tier"`
	Difficulty          int                       `json:"difficulty"`
	SessionID           ids.SessionID             `json:"session_id"`
	SceneID             ids.SceneID               `json:"scene_id"`
	Notes               string                    `json:"notes,omitempty"`
}

// EnvironmentEntityCreatedPayload captures the payload for
// sys.daggerheart.environment_entity_created events.
type EnvironmentEntityCreatedPayload = EnvironmentEntityCreatePayload

// EnvironmentEntityUpdatePayload captures the payload for
// sys.daggerheart.environment_entity.update commands.
type EnvironmentEntityUpdatePayload struct {
	EnvironmentEntityID dhids.EnvironmentEntityID `json:"environment_entity_id"`
	EnvironmentID       string                    `json:"environment_id"`
	Name                string                    `json:"name"`
	Type                string                    `json:"type"`
	Tier                int                       `json:"tier"`
	Difficulty          int                       `json:"difficulty"`
	SessionID           ids.SessionID             `json:"session_id"`
	SceneID             ids.SceneID               `json:"scene_id"`
	Notes               string                    `json:"notes,omitempty"`
}

// EnvironmentEntityUpdatedPayload captures the payload for
// sys.daggerheart.environment_entity_updated events.
type EnvironmentEntityUpdatedPayload = EnvironmentEntityUpdatePayload

// EnvironmentEntityDeletePayload captures the payload for
// sys.daggerheart.environment_entity.delete commands.
type EnvironmentEntityDeletePayload struct {
	EnvironmentEntityID dhids.EnvironmentEntityID `json:"environment_entity_id"`
	Reason              string                    `json:"reason,omitempty"`
}

// EnvironmentEntityDeletedPayload captures the payload for
// sys.daggerheart.environment_entity_deleted events.
type EnvironmentEntityDeletedPayload = EnvironmentEntityDeletePayload
